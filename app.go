package main

import (
	"github.com/AVENTER-UG/mesos-firecracker-executor/mesos"
	mesosutil "github.com/AVENTER-UG/mesos-util"
	util "github.com/AVENTER-UG/util"
	"github.com/sirupsen/logrus"
)

// BuildVersion of m3s
var BuildVersion string

// GitVersion is the revision and commit number
var GitVersion string

func main() {

	util.SetLogging(config.LogLevel, config.EnableSyslog, config.AppName)
	logrus.Println(config.AppName + " build " + BuildVersion + " git " + GitVersion)

	mesosutil.SetConfig(&framework)
	mesos.SetConfig(&config, &framework)

	// Create and run the executor
	if err := mesos.Executor(); err != nil {
		logrus.Fatal("An error occured while running the executor")
	}
}
