package main

import (
	"os"
	"sort"
	"strings"

	"github.com/AVENTER-UG/mesos-firecracker-executor/mesos"
	"github.com/AVENTER-UG/mesos-firecracker-executor/mesosdriver"
	mesosconfig "github.com/mesos/mesos-go/api/v1/lib/executor/config"
	"github.com/sirupsen/logrus"
)

// BuildVersion of m3s
var BuildVersion string

// GitVersion is the revision and commit number
var GitVersion string

// Settings of the firecracker executor
var Settings map[string]string

func logConfig() {
	logrus.Infof("Environment ---------------------------")
	envVars := os.Environ()
	sort.Strings(envVars)
	Settings = make(map[string]string)
	for _, setting := range envVars {

		if strings.HasPrefix(setting, "MESOS") ||
			strings.HasPrefix(setting, "FIRECRACKER") ||
			(setting == "HOME") {

			pair := strings.Split(setting, "=")
			logrus.Infof(" * %-30s: %s", pair[0], pair[1])
			Settings[pair[0]] = pair[1]
		}
	}
	logrus.Infof("---------------------------------------")
}

func main() {

	logConfig()

	cfg, err := mesosconfig.FromEnv()
	if err != nil {
		logrus.WithField("func", "main").Fatal("failed to load configuration: " + err.Error())
	}

	var level logrus.Level
	level, err = logrus.ParseLevel("TRACE")
	if err != nil {
		return
	}
	logrus.SetLevel(level)

	nExec := mesos.NewExecutor(&cfg, Settings)
	nExec.Driver = mesosdriver.NewExecutorDriver(&cfg, nExec)
	err = nExec.Driver.Run()
	if err != nil {
		logrus.WithField("func", "main").Errorf("Immediate Exit: Error from executor driver: %s", err)
		return
	}

	logrus.WithField("func", "main").Info("Sidecar Executor exiting")

}
