package nodemanager

import "github.com/opencontainers/runtime-spec/specs-go"

// Node runtime information
type Node struct {
	Name       string
	Runtime    string
	Env        []string
	Devices    []specs.LinuxDevice
	Caps       []string
	Priviliged bool
	GIDs       string
}

// NodeTask Task information
type NodeTask struct {
	Name string
	Args []string
	Env  []string
	Cwd  string
	PID  uint32
}
