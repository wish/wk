package kops

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sirupsen/logrus"

	sigs_yaml "sigs.k8s.io/yaml"

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

		return CopyFile(tfile, dryFile)
	}

	s := newState()
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

	// This shouldn't be made concurrent, since kops as a tool cannot be run concurrently.
	// I tried. kops ended up overwriting one instancegroup with another
	for _, ig := range cluster.Kops.InstanceGroups {
		logrus.Infoln("Editing instance group:", ig.Name)

		func(ig types.InstanceGroup) {
			// TODO(akursell): This is usually pointless
			igCmd := exec.CommandContext(ctx, "kops", "create", "ig", "--name="+cluster.Name, ig.Name)
			igCmd.Env = append(kopsEnv, fmt.Sprintf("%v=%v %v %v %v %v %v %v", "EDITOR", ex, "cluster-edit-ig", file, tfile, statefile, mode, ig.Name))
			_ = igCmd.Run()

			igCmd = exec.CommandContext(ctx, "kops", "edit", "ig", "--name="+cluster.Name, ig.Name)
			igCmd.Env = append(kopsEnv, fmt.Sprintf("%v=%v %v %v %v %v %v %v", "EDITOR", ex, "cluster-edit-ig", file, tfile, statefile, mode, ig.Name))
			igCmd.Stdout, igCmd.Stderr = os.Stdout, os.Stderr
			err = igCmd.Run()
		}(ig)
		if err != nil {
			return err
		}
	}

	s = getState(statefile)
	if !preview && (s.requiresUpdate() || forceUpdate) {
		logrus.Infoln("Update is required. Issuing update.")

		uCmd := exec.CommandContext(ctx, "kops", "update", "cluster", "--name="+cluster.Name, "-v1", "--yes")
		uCmd.Stdout, uCmd.Stderr = os.Stdout, os.Stderr
		uCmd.Env = kopsEnv
		if err = uCmd.Run(); err != nil {
			return fmt.Errorf("could not update cluster: %v", err)
		}
	} else {
		logrus.Infoln("Not performing update.")
	}

	return nil
}

func ClusterEdit(ctx context.Context, args []string) error {
	renderedJsonnet, stateFile, mode, outFile := args[1], args[2], args[3], args[4]

	cluster, err := ReadClusterFile(renderedJsonnet)
	if err != nil {
		return nil
	}

	data, err := ioutil.ReadFile(outFile)
	if err != nil {
		return err
	}

	oldCluster := map[string]interface{}{}
	if err = sigs_yaml.Unmarshal(data, &oldCluster); err != nil {
		return err
	}
	newCluster := cluster.Kops.Cluster

	// Preserve the timestamps across applications. Otherwise it always shows a diff
	newCluster["metadata"].(map[string]interface{})["creationTimestamp"] = oldCluster["metadata"].(map[string]interface{})["creationTimestamp"]

	eq, diffText := diff(oldCluster, newCluster)
	updateState(stateFile, func(s *State) {
		s.Cluster = ObjectState{
			UpdateRequired: !eq,
			DiffText:       diffText,
		}
	})
	if !eq {
		logrus.Info(diffText)
	}

	if mode == "preview" {
		return nil
	}

	if !eq {
		newFileData, err := json.Marshal(newCluster)
		if err != nil {
			return err
		}
		ioutil.WriteFile(outFile, newFileData, os.ModePerm)
	}
	return nil
}

func ClusterEditIG(ctx context.Context, args []string) error {
	renderedJsonnet, stateFile, mode, igName, outFile := args[1], args[2], args[3], args[4], args[5]

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
	oldIG := map[string]interface{}{}
	if err = sigs_yaml.Unmarshal(data, &oldIG); err != nil {
		return err
	}
	// Preserve the timestamps across applications. Otherwise it always shows a diff
	ptch["metadata"].(map[string]interface{})["creationTimestamp"] = oldIG["metadata"].(map[string]interface{})["creationTimestamp"]

	eq, diffText := diff(oldIG, ptch)
	updateState(stateFile, func(s *State) {
		s.InstanceGroups[igName] = ObjectState{
			UpdateRequired: !eq,
			DiffText:       diffText,
		}
	})
	if !eq {
		logrus.Info(diffText)
	}
	if mode == "preview" {
		return nil
	}

	if !eq {
		newFileData, err := json.Marshal(ptch)
		if err != nil {
			return err
		}
		ioutil.WriteFile(outFile, newFileData, os.ModePerm)
	}

	return nil
}
