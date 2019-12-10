package opa

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/open-policy-agent/opa/rego"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

// OPA represents an OpenPolicyAgent query instance
type OPA struct {
	query         rego.PreparedEvalQuery
	queryString   string
	failUndefined bool
	failDefined   bool
}

func isEmpty(x interface{}) bool {
	switch v := x.(type) {
	case []interface{}:
		return len(v) == 0
	default:
		return false
	}
}

func (o *OPA) RunFile(path string) (bool, []string, error) {
	inp, err := ioutil.ReadFile(path)
	if err != nil {
		return false, nil, err
	}

	issues := []string{}
	for _, part := range strings.Split(strings.Trim(strings.TrimSpace(string(inp)), "."), "---\n") {
		if strings.TrimSpace(part) == "" {
			continue
		}

		obj := map[string]interface{}{}
		err := json.Unmarshal([]byte(part), &obj)
		if err != nil {
			return false, nil, err
		}

		ans, err := o.Run(obj)
		if isEmpty(ans) && o.failUndefined {
			issues = append(issues, "OPA query returned empty or undefined")
		} else if !isEmpty(ans) && o.failDefined {
			out, err := json.MarshalIndent(ans, "", "  ")
			if err != nil {
				return false, nil, err
			}
			issues = append(issues, string(out))
		}
	}

	return len(issues) == 0, issues, nil
}

func (o *OPA) Run(obj interface{}) (interface{}, error) {
	res, err := o.query.Eval(context.Background(), rego.EvalInput(obj))
	if err != nil {
		return "", err
	}

	for _, expr := range res {
		for _, v := range expr.Expressions {
			if v.Text == o.queryString {
				return v.Value, nil
			}
		}
	}

	return "", fmt.Errorf("Query value '%v' not found", o.queryString)

}

// AddOPAOpts adds command line options for OPA to a command
func AddOPAOpts(cmd *cobra.Command) {
	cmd.Flags().StringArray("opa-data", nil, "An OPA .rego file to import")
	cmd.Flags().Bool("opa-fail", false, "exits with non-zero exit code on undefined/empty OPA eval result")
	cmd.Flags().Bool("opa-fail-defined", false, "exits with non-zero exit code on defined/nonempty OPA eval result")
	cmd.Flags().String("opa-query", "", "The query to run with the created resource as input")
}

// FromFlags creates an OPA query ready to run
func FromFlags(flags *flag.FlagSet) (*OPA, error) {
	queryString, err := flags.GetString("opa-query")
	if err != nil {
		return nil, err
	}
	if queryString == "" {
		return nil, nil
	}
	dataFiles, err := flags.GetStringArray("opa-data")
	if err != nil {
		return nil, err
	}

	modules := []func(*rego.Rego){
		rego.Query(queryString),
	}
	for _, dataFile := range dataFiles {
		b, err := ioutil.ReadFile(dataFile)
		if err != nil {
			return nil, err
		}
		modules = append(modules, rego.Module(dataFile, string(b)))
	}

	failUndefined, err := flags.GetBool("opa-fail")
	if err != nil {
		return nil, err
	}

	failDefined, err := flags.GetBool("opa-fail-defined")
	if err != nil {
		return nil, err
	}

	if failDefined && failUndefined {
		return nil, fmt.Errorf("--opa-fail and --opa-fail-defined cannot both be true")
	}

	query, err := rego.New(modules...).PrepareForEval(context.Background())
	if err != nil {
		return nil, err
	}

	return &OPA{
		query:         query,
		queryString:   queryString,
		failDefined:   failDefined,
		failUndefined: failUndefined,
	}, nil
}
