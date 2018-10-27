package config

type Processor struct {
	Name 					string 									`json: "name" yaml: "name"`
	Branch  			string 									`json:"branch" yaml: "branch"`
	Runtime 			string									`json:"runtime" yaml: "runtime"`
	NodeSelector	NodeSelector						`json:"nodeSelector" yaml: "nodeSelector"`
	Concurrency		Concurrency							`json:"concurrency" yaml: "concurrency"`
	Resources			Resources								`json:"resources" yaml: "resources"`
	Content 			[]string								`json:"content" yaml: "content"`
	Command 			[]string								`json:"command" yaml: "command"`
	Secrets				ConfigProperties				`json:"secrets" yaml: "secrets"`
	ConfigMaps		ConfigProperties				`json:"configmaps" yaml: "configmaps"`
	Input					Input										`json:"input" yaml: "input"`
}


type NodeSelector struct {
	Gpu						string									`json:"gpu" yaml: "gpu"`
	AntiAffinity	string									`json:"antiAffinity yaml: "antiAffinity"`
}

type Concurrency struct {
	Min						uint32									`json:"min yaml: "min"`
	Max						uint32									`json:"max" yaml "max"`
	Condition			string									`json:"condition" yaml "condition"`
}

type Resources struct {
	Cpu 					ResourceLimit						`json:"cpu" yaml: "cpu"`
	Mem 					ResourceLimit						`json:"mem" yaml: "mem"`
}

type ResourceLimit struct {
	Min						string									`json:"min" yaml: "min"`
	Max 					string									`json:"max" yaml: "max"`
}

type ConfigProperties struct {
	Name					string									`json:"name" yaml: "name"`
	Path 					string									`json:"path" yaml: "path"`
}

type Input struct {
	Type 					string									`json:"type" yaml: "type"`
	Name 					string									`json:"name" yaml: "name"`
	Version 			string									`json:"version" yaml: "version"`
	Filter 				string									`json:"filter" yaml: "filter"`
}
