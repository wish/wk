package types

type Cluster struct {
	Name string
	Kops *Kops
}

type Kops struct {
	Env     map[string]string
	Create  map[string]interface{}
	Cluster map[string]interface{}

	InstanceGroups []InstanceGroup
	Channels       []Channel
}

type InstanceGroup struct {
	Name  string
	Value map[string]interface{}
}

type Channel struct {
	Name   string
	Path   string
	Apps   []App
	Folder string
}

type App struct {
	Helm
	File

	App  string
	Type string
}

type Helm struct {
	Values map[string]interface{}
}

type File struct {
	Path string
}
