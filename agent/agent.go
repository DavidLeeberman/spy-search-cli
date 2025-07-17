package agent

import (
	"spysearch/models"
	"spysearch/tools"
)

// this is an agent package
type Agent struct {
	Tools  []tools.Tool // a list of tool
	Steps  int          // number of step allow the agent to run
	Mmeory []string     // save the memory
	m      models.OllamaClient
}

// all agent need a run function
type AgentInterface interface {
	Run(p string)
}

// we call our agent spy agent
type SpyAgent Agent

// main logic here
func (s SpyAgent) Run(p string) {

	// init
	_, err := s.m.Completion(p, s.Tools)
	if err != nil {
		// shall we really panic here ?
		panic(err.Error())
	}
	s.Steps -= 1

	for {
		if s.Steps <= 0 {
			break
		}
		// what should i do here ?

		s.Steps -= 1
	}

}
