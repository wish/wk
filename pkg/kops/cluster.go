package kops

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/wish/wk/pkg/jsonnet"
	"github.com/wish/wk/pkg/types"
	"github.com/wish/wk/pkg/util"
)

func ClusterApply(ctx context.Context, file, dryFile string, forceUpdate, preview bool) error {
	cluster, tfile, err := jsonnet.ExpandCluster(ctx, file)
	if err != nil {
		return err
	}
	if cluster.Kops == nil {
		return fmt.Errorf("kops configuration is missing")
	}
	if dryFile != "" {
		if err := os.MkdirAll(filepath.Dir(dryFile), os.ModePerm); err != nil {
			return err
		}
		if err := os.Rename(tfile, dryFile); err != nil {
			return err
		}
		return nil
	}

	s := &State{UpdateRequired: false}
	sb, err := json.Marshal(s)
	if err != nil {
		return err
	}
	statefile, err := util.WriteTempFile(sb)
	if err != nil {
		return err
	}

	kopsEnv := os.Environ()
	for k, v := range cluster.Kops.Env {
		kopsEnv = append(kopsEnv, fmt.Sprintf("%v=%v", k, v))
	}

	ex, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not get executable: %v", err)
	}

	logrus.Infoln("Editing cluster.")
	mode := "normal"
	if preview {
		mode = "preview"
	}

	eCmd := exec.CommandContext(ctx, "kops", "edit", "cluster", "--name="+cluster.Name)
	eCmd.Stdout, eCmd.Stderr = os.Stdout, os.Stderr
	eCmd.Env = append(kopsEnv, fmt.Sprintf("%v=%v %v %v %v %v %v", "EDITOR", ex, "cluster-edit", file, tfile, statefile, mode))
	if err = eCmd.Run(); err != nil {
		return fmt.Errorf("could not edit cluster: %v", err)
	}

	// TODO(tvi): Add preview for IGs as well.
	if preview {
		return nil
	}

	// wg := &sync.WaitGroup{}
	// TODO(tvi): Make concurrent.
	for _, ig := range cluster.Kops.InstanceGroups {
		logrus.Infoln("Editing instance group:", ig.Name)

		func(ig types.InstanceGroup, wg *sync.WaitGroup) {
			igCmd := exec.CommandContext(ctx, "kops", "create", "ig", "--name="+cluster.Name, ig.Name)
			igCmd.Env = append(kopsEnv, fmt.Sprintf("%v=%v %v %v %v %v %v", "EDITOR", ex, "cluster-edit-ig", file, tfile, statefile, ig.Name))
			_ = igCmd.Run()

			igCmd = exec.CommandContext(ctx, "kops", "edit", "ig", "--name="+cluster.Name, ig.Name)
			igCmd.Env = append(kopsEnv, fmt.Sprintf("%v=%v %v %v %v %v %v", "EDITOR", ex, "cluster-edit-ig", file, tfile, statefile, ig.Name))
			igCmd.Stdout, igCmd.Stderr = os.Stdout, os.Stderr
			err = igCmd.Run()

			// wg.Done()
		}(ig, nil)
		if err != nil {
			return err
		}
	}
	// wg.Wait()

	s = getState(statefile)
	if s.UpdateRequired || forceUpdate {
		logrus.Infoln("Update is required. Issuing update.")

		uCmd := exec.CommandContext(ctx, "kops", "update", "cluster", "--name="+cluster.Name, "-v1", "--yes")
		uCmd.Stdout, uCmd.Stderr = os.Stdout, os.Stderr
		uCmd.Env = kopsEnv
		if err = uCmd.Run(); err != nil {
			return fmt.Errorf("could not update cluster: %v", err)
		}
	} else {
		logrus.Infoln("Update is not required. Skipping.")
	}

	return nil
}

func ClusterEdit(ctx context.Context, args []string) error {
	renderedJsonnet, stateFile, mode, outFile := args[1], args[2], args[3], args[4]
	s := getState(stateFile)
	defer saveState(s, stateFile)

	cluster, err := ReadClusterFile(renderedJsonnet)
	if err != nil {
		return nil
	}

	data, err := ioutil.ReadFile(outFile)
	if err != nil {
		return err
	}
	eq, newData, err := patch(data, cluster.Kops.Cluster)
	if err != nil {
		return err
	}
	if mode == "preview" {
		return nil
	}
	if !eq {
		ioutil.WriteFile(outFile, newData, os.ModePerm)
		s.UpdateRequired = true
	}
	return nil
}

func ClusterEditIG(ctx context.Context, args []string) error {
	renderedJsonnet, stateFile, igName, outFile := args[1], args[2], args[3], args[4]
	s := getState(stateFile)
	defer saveState(s, stateFile)

	cluster, err := ReadClusterFile(renderedJsonnet)
	if err != nil {
		return err
	}

	var ptch map[string]interface{}
	for _, ig := range cluster.Kops.InstanceGroups {
		if ig.Name == igName {
			ptch = ig.Value
		}
	}

	data, err := ioutil.ReadFile(outFile)
	if err != nil {
		return err
	}
	eq, newData, err := patch(data, ptch)
	if err != nil {
		return err
	}
	if !eq {
		ioutil.WriteFile(outFile, newData, os.ModePerm)
		s.UpdateRequired = true
	}

	return nil
}
