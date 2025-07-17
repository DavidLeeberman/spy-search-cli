package agent

import "spysearch/tools"

// this is an agent package
type Agent struct {
	Tools  []tools.Tool // a list of tool
	Steps  int          // number of step allow the agent to run
	Mmeory []string     // save the memory
}

// all agent need a run function
type AgentInterface interface {
	Run()
}

// we call our agent spy agent
type SpyAgent Agent

// main logic here
func (s SpyAgent) Run() {
}
