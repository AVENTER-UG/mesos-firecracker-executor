package main

import (
	"os"
	"strings"

	mesosutil "github.com/AVENTER-UG/mesos-util"
	util "github.com/AVENTER-UG/util"

	cfg "github.com/AVENTER-UG/mesos-firecracker-executor/types"
)

var config cfg.Config
var framework mesosutil.FrameworkConfig

func init() {

	config.MesosAgentHostname = os.Getenv("MESOS_AGENT_ENDPOINT")
	config.AgentID = os.Getenv("MESOS_SLAVE_ID")
	config.ExecutorID = os.Getenv("MESOS_EXECUTOR_ID")
	config.FrameworkID = os.Getenv("MESOS_FRAMEWORK_ID")
	config.LogLevel = util.Getenv("LOGLEVEL", "debug")
	config.AppName = "Mesos Mainframe Framework"

	// Skip SSL Verification
	if strings.Compare(os.Getenv("SKIP_SSL"), "true") == 0 {
		config.SkipSSL = true
	} else {
		config.SkipSSL = false
	}
}
