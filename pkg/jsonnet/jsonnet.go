package jsonnet

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/wish/wk/pkg/types"
	"github.com/wish/wk/pkg/util"
)

const exttCode = `kops={
  cluster:: {},
  instanceGroup:: function(n){name:n, value:{}},
  channel:: function(bucket, cluster, name, apps=[], folder='') {
    name: name,
    path: bucket + '/' + cluster + '/' + name,
    apps: apps,
    folder: folder,
  },
  file:: function(path) {
    type: 'file',
    app: '',
    path: path,
  },
  apps:: function(path) {
    type: 'apps',
    app: '',
    path: path,
  },
}`

func getEnv() string {
	envs := os.Environ()
	envArg := "env={"
	for _, env := range envs {
		if strings.HasPrefix(env, "WK") {
			sp := strings.SplitN(env, "=", 2)
			envArg += fmt.Sprintf("%v:'%v',", sp[0], sp[1])
		}
	}
	envArg += "}"
	return envArg
}

func template(ctx context.Context, file string, extraArgs []string) ([]byte, string, error) {
	ctxDir, err := util.GetContextDir(file)
	if err != nil {
		return nil, "", err
	}

	tfile, err := util.WriteTempFile([]byte{})
	if err != nil {
		return nil, "", err
	}

	extraArgs = append(extraArgs, []string{
		"--ext-code", exttCode,
		"--ext-code", getEnv(),
		"--ext-code", "wk=true",
		"-J", ctxDir,
		"-o", tfile,
		file,
	}...)

	cmd := exec.CommandContext(ctx, "jsonnet", extraArgs...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, "", err
	}
	h, err := ioutil.ReadFile(tfile)
	if err != nil {
		return nil, "", err
	}
	return h, tfile, nil
}

func ExpandCluster(ctx context.Context, file string) (*types.Cluster, string, error) {
	h, tfile, err := template(ctx, file, []string{})
	if err != nil {
		return nil, "", err
	}
	cluster := &types.Cluster{}
	if err := json.Unmarshal(h, cluster); err != nil {
		return nil, "", err
	}
	return cluster, tfile, nil
}

func ExpandAppFile(ctx context.Context, file, cluster string) (bool, string, string, error) {
	h, tfile, err := template(ctx, file, []string{
		"-y", "--ext-code-file", "cluster=" + cluster,
	})
	if err != nil {
		return false, "", "", err
	}
	if len(strings.TrimSpace(string(h))) == 0 {
		if err := os.Remove(tfile); err != nil {
			return false, "", "", err
		}
		return true, tfile, "", nil
	}
	sum := sha256.Sum256(h)
	return false, tfile, fmt.Sprintf("%x", sum), nil
}
