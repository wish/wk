package kops

import (
	"encoding/json"
	"io/ioutil"
	"reflect"
	"strings"

	godebug_diff "github.com/kylelemons/godebug/diff"
	"github.com/sirupsen/logrus"

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

// diff returns whether the structures are different, and a textual diff of the changes
func diff(old, new map[string]interface{}) (bool, string) {
	if !reflect.DeepEqual(old, new) {
		var A, B []byte
		var err error
		if A, err = json.MarshalIndent(&old, "", "  "); err != nil {
			logrus.Fatalf("Error marshalling old to JSON: %v", err)
		}
		if B, err = json.MarshalIndent(&new, "", "  "); err != nil {
			logrus.Fatalf("Error marshalling new to JSON: %v", err)
		}
		textDiff := util.Render(godebug_diff.DiffChunks(
			strings.Split(string(A), "\n"),
			strings.Split(string(B), "\n"),
		))

		return false, textDiff
	}

	return true, ""
}
