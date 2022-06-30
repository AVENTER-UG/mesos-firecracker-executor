package main

import (
	"log"
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

func logConfig() {
	logrus.Infof("Environment ---------------------------")
	envVars := os.Environ()
	sort.Strings(envVars)
	for _, setting := range envVars {

		if strings.HasPrefix(setting, "MESOS") ||
			strings.HasPrefix(setting, "EXECUTOR") ||
			strings.HasPrefix(setting, "VAULT") ||
			(setting == "HOME") {

			pair := strings.Split(setting, "=")
			logrus.Infof(" * %-30s: %s", pair[0], pair[1])
		}
	}
	logrus.Infof("---------------------------------------")
}

func main() {

	os.Setenv("MESOS_SANDBOX", "/tmp")

	logConfig()

	nExec := mesos.NewExecutor()

	cfg, err := mesosconfig.FromEnv()
	if err != nil {
		log.Fatal("failed to load configuration: " + err.Error())
	}

	nExec.Driver = mesosdriver.NewExecutorDriver(&cfg, nExec)
	err = nExec.Driver.Run()
	if err != nil {
		logrus.Errorf("Immediate Exit: Error from executor driver: %s", err)
		return
	}

	logrus.Info("Sidecar Executor exiting")

}
