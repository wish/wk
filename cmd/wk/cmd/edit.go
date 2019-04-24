package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/wish/wk/pkg/kops"
	"github.com/wish/wk/pkg/util"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

func init() {
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(editIGCmd)
}

var editCmd = &cobra.Command{
	Use:    "edit",
	Hidden: true,
	Args:   cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		editedFile := args[1]
		jsonData, err := readYAMLasJSON(editedFile)
		if err != nil {
			panic(err)
		}
		tmpfile, err := util.WriteTempFile(jsonData)
		if err != nil {
			panic(err)
		}
		defer os.Remove(tmpfile) // clean up

		out2, err := kops.Edit(context.Background(), args[0], tmpfile, "")
		if err != nil {
			panic(err)
		}

		p := out2.Kops.Cluster
		y, err := yaml.Marshal(p)
		if err != nil {
			panic(err)
		}
		ioutil.WriteFile(args[1], y, os.ModePerm)
	},
}

var editIGCmd = &cobra.Command{
	Use:    "edit-ig",
	Hidden: true,
	Args:   cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		editedFile := args[2]
		jsonData, err := readYAMLasJSON(editedFile)
		if err != nil {
			panic(err)
		}
		jsonData = append([]byte("function(n){name:n,value:"), jsonData...)
		jsonData = append(jsonData, []byte("}")...)
		tmpfile, err := util.WriteTempFile(jsonData)
		if err != nil {
			panic(err)
		}
		defer os.Remove(tmpfile) // clean up

		out2, err := kops.Edit(context.Background(), args[0], "", tmpfile)
		if err != nil {
			panic(err)
		}

		p := out2.Kops.InstanceGroups
		var i map[string]interface{}
		for _, ig := range p {
			if ig.Name == args[1] {
				i = ig.Value
			}
		}
		if i == nil {
			panic("name not found")
		}
		y, err := yaml.Marshal(i)
		if err != nil {
			panic(err)
		}
		ioutil.WriteFile(args[2], y, os.ModePerm)
	},
}

func readYAMLasJSON(path string) ([]byte, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read file: %v", err)
	}

	content, err := yaml.YAMLToJSON(b)
	if err != nil {
		return nil, fmt.Errorf("could read yaml: %v", err)
	}
	return content, nil
}
