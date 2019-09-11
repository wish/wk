package kops

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"reflect"
	"strings"

	"github.com/imdario/mergo"
	"github.com/kylelemons/godebug/diff"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"

	"github.com/wish/wk/pkg/types"
	"github.com/wish/wk/pkg/util"
)

func ReadClusterFile(path string) (*types.Cluster, error) {
	out, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cluster := &types.Cluster{}
	if err := json.Unmarshal(out, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

func patch(data []byte, patch map[string]interface{}) (bool, []byte, error) {
	if bytes.HasPrefix(data, []byte{'#'}) && bytes.Contains(data, []byte("# ...")) {
		for _, line := range bytes.Split(data, []byte{'\n'}) {
			if !bytes.HasPrefix(line, []byte{'#'}) {
				break
			}
			logrus.Infoln(line)
		}
	}
	src := &map[string]interface{}{}
	dst := &map[string]interface{}{}
	var err error

	if err = yaml.Unmarshal(data, src); err != nil {
		return false, nil, err
	}
	if err = yaml.Unmarshal(data, dst); err != nil {
		return false, nil, err
	}
	if err = mergo.Merge(dst, patch, mergo.WithOverride); err != nil {
		return false, nil, err
	}

	if !reflect.DeepEqual(*src, *dst) {
		var A, B []byte
		if A, err = json.MarshalIndent(src, "", "  "); err != nil {
			return false, nil, err
		}
		if B, err = json.MarshalIndent(dst, "", "  "); err != nil {
			return false, nil, err
		}
		logrus.Infoln(util.Render(diff.DiffChunks(
			strings.Split(string(A), "\n"),
			strings.Split(string(B), "\n"),
		)))

		y, err := yaml.Marshal(dst)
		if err != nil {
			return false, nil, err
		}
		return false, y, nil
	}

	return true, nil, nil
}
