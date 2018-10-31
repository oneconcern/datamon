package config

type Processor struct {
	Name 					string 									`json: "name" yaml: "name"`
	Branch  			string 									`json:"branch" yaml: "branch"`
	Runtime 			string									`json:"runtime" yaml: "runtime"`
	Resources			Resources								`json:"resources" yaml: "resources"`
	Content 			[]string								`json:"content" yaml: "content"`
	Command 			string									`json:"command" yaml: "command"`
	Dep 					string									`json:"dep" yaml: "dep"`
	Port 					int32 									`json:"port" yaml: "port"`

}

type Resources struct {
	Cpu 					ResourceLimit						`json:"cpu" yaml: "cpu"`
	Mem 					ResourceLimit						`json:"mem" yaml: "mem"`
}

type ResourceLimit struct {
	Min						string									`json:"min" yaml: "min"`
	Max 					string									`json:"max" yaml: "max"`
}


