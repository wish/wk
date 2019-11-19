package kops

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	// "bitbucket.org/avd/go-ipc/sync"
)

// State represents the results of running `kops edit cluster` and `kops edit ig`
// across multiple resources
type State struct {
	Cluster        ObjectState
	InstanceGroups map[string]ObjectState
}

// ObjectState represents the results of running `kops edit cluster` or `kops edit ig`
type ObjectState struct {
	UpdateRequired bool
	DiffText       string
}

func (s *State) requiresUpdate() bool {
	if s.Cluster.UpdateRequired {
		return true
	}
	for _, ig := range s.InstanceGroups {
		if ig.UpdateRequired {
			return true
		}
	}
	return false
}

func (s *State) renderDiffs() string {
	r := ""
	if s.Cluster.DiffText != "" {
		r += fmt.Sprintf("Cluster changed:\n%v\n\n", s.Cluster.DiffText)
	}
	for name, ig := range s.InstanceGroups {
		if ig.DiffText != "" {
			r += fmt.Sprintf("Instance Group %v changed:\n%v\n\n", name, ig.DiffText)
		}
	}
	if r == "" {
		r = "No changes."
	}
	return r
}

func newState() *State {
	return &State{
		InstanceGroups: make(map[string]ObjectState),
	}
}

func updateState(path string, upFunc func(s *State)) {
	stateb, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	s := &State{}
	if err := json.Unmarshal(stateb, s); err != nil {
		panic(err)
	}

	upFunc(s)
	sb, _ := json.Marshal(s)
	ioutil.WriteFile(path, sb, os.ModePerm)
}

func getState(path string) *State {
	stateb, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	s := &State{}
	if err := json.Unmarshal(stateb, s); err != nil {
		panic(err)
	}
	return s
}
