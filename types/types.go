package types

import (
	"net/http"

	mesosproto "github.com/AVENTER-UG/mesos-util/proto"
)

// Config is a struct of the framework configuration
type Config struct {
	LogLevel           string
	MinVersion         string
	AppName            string
	EnableSyslog       bool
	MesosAgentHostname string
	FrameworkID        string
	AgentID            string
	ExecutorID         string
	ExecutorHostname   string
	ExecutorPort       string
	SkipSSL            bool
	SSLKey             string
	SSLCrt             string
	Listen             string
}

// Executor represents an executor
type Executor struct {
	AgentInfo     mesosproto.AgentInfo // AgentInfo contains agent info returned by the agent
	Cli           *http.Client         // Cli is the mesos HTTP cli
	Req           *http.Request
	ExecutorID    string                   // Executor ID returned by the agent when running the executor
	ExecutorInfo  mesosproto.ExecutorInfo  // Executor info returned by the agent after registration
	FrameworkID   string                   // Framework ID returned by the agent when running the executor
	FrameworkInfo mesosproto.FrameworkInfo // Framework info returned by the agent after registration
	Shutdown      bool                     // Shutdown the executor (used to stop loop event and gently kill the executor)
	TaskInfo      mesosproto.TaskInfo      // Task info sent by the agent on launch event
}
