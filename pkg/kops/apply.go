package kops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"

	"github.com/sirupsen/logrus"
)

const channelTemplate = `kind: Addons
metadata:
  creationtimestamp: null
  name: %v
spec:
  addons:
    - manifest: %v.json
      name: %v
      version: 0.1.0
      id: %x  # sha
`

func Apply(ctx context.Context, file string) (err error) {
	out, err := Edit(ctx, file, "", "")
	if err != nil {
		return fmt.Errorf("could not render jsonnet: %v", err)
	}
	if out.Kops == nil {
		return fmt.Errorf("kops configuration is missing")
	}

	for _, c := range out.Kops.Channels {
		err := generateChannel(ctx, c, file)
		if err != nil {
			return fmt.Errorf("could not generate channel: %v", err)
		}
	}

	c := out.Kops.Create
	kopsEnv := os.Environ()
	for k, v := range out.Kops.Env {
		kopsEnv = append(kopsEnv, fmt.Sprintf("%v=%v", k, v))
	}

	createArgs := []string{"create", "cluster"}
	for k, v := range c {
		createArgs = append(createArgs, fmt.Sprintf("--%v=%v", k, v))
	}
	createArgs = append(createArgs, "--name="+out.Name)
	cmd := exec.CommandContext(ctx, "kops", createArgs...)
	if logrus.GetLevel() >= logrus.DebugLevel {
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	}
	cmd.Env = kopsEnv
	_ = cmd.Run()

	var ex string
	ex, err = os.Executable()
	if err != nil {
		return fmt.Errorf("could not get executable: %v", err)
	}

	updateCmd := exec.CommandContext(ctx, "kops",
		"edit", "cluster", "--name="+out.Name)
	updateCmd.Stdout, updateCmd.Stderr = os.Stdout, os.Stderr
	updateCmd.Env = append(kopsEnv, fmt.Sprintf("%v=%v %v %v", "EDITOR", ex, "edit", file))
	err = updateCmd.Run()
	if err != nil {
		return fmt.Errorf("could not edit cluster: %v", err)
	}

	wg := &sync.WaitGroup{}
	for _, ig := range out.Kops.InstanceGroups {
		wg.Add(1)
		go func(ig InstanceGroup, wg *sync.WaitGroup) {
			updateCmd := exec.CommandContext(ctx, "kops", "create", "ig", "--name="+out.Name, ig.Name)
			updateCmd.Env = append(kopsEnv, fmt.Sprintf("%v=%v %v %v %v", "EDITOR", ex, "edit-ig", file, ig.Name))
			_ = updateCmd.Run()

			updateCmd = exec.CommandContext(ctx, "kops", "edit", "ig", "--name="+out.Name, ig.Name)
			updateCmd.Env = append(kopsEnv, fmt.Sprintf("%v=%v %v %v %v", "EDITOR", ex, "edit-ig", file, ig.Name))
			updateCmd.Stdout, updateCmd.Stderr = os.Stdout, os.Stderr
			err = updateCmd.Run()
			wg.Done()
		}(ig, wg)
	}
	wg.Wait()
	return
}

const extCode = `kops={
  cluster:: %v,
  instanceGroup:: %v,
  channel:: function(bucket, cluster, name, apps=[]) {
    name: name,
    path: bucket + '/' + cluster + '/.channel',
    apps: apps,
  },
  file:: function(path) {
    type: 'file',
    app: '',
    path: path,
  },
}`

func Edit(ctx context.Context, file, clusterFile, igFile string) (*Cluster, error) {
	clusterData := ""

	args := []string{}
	if clusterFile == "" {
		clusterData = "{}"
	} else {
		b, err := ioutil.ReadFile(clusterFile)
		if err != nil {
			return nil, err
		}
		clusterData = string(b)
	}

	igData := ""
	if igFile == "" {
		igData = "function(n){name:n, value:{}}"
	} else {
		b, err := ioutil.ReadFile(igFile)
		if err != nil {
			return nil, err
		}
		igData = string(b)
	}
	args = append(args, "--ext-code", fmt.Sprintf(extCode, clusterData, igData))
	args = append(args, file)

	c := exec.CommandContext(ctx, "jsonnet", args...)
	out := bytes.NewBufferString("")
	c.Stdout = out
	c.Stderr = os.Stderr
	err := c.Run()
	if err != nil {
		return nil, err
	}

	// out2 := make(map[string]interface{})
	out2 := &Cluster{}
	json.Unmarshal(out.Bytes(), out2)
	return out2, nil
}
