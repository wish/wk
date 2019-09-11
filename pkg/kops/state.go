package kops

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type State struct {
	UpdateRequired bool
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

func saveState(s *State, path string) {
	sb, _ := json.Marshal(s)
	ioutil.WriteFile(path, sb, os.ModePerm)
}
