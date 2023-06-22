package mesos

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/AVENTER-UG/mesos-firecracker-executor/mesosdriver"
	util "github.com/AVENTER-UG/util"
	"github.com/firecracker-microvm/firecracker-go-sdk"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/executor/config"
	"github.com/sirupsen/logrus"
)

type Firecracker struct {
	Ctx         context.Context
	ID          string
	Machine     *firecracker.Machine
	IP          net.IP
	Socket      string
	Driver      *mesosdriver.ExecutorDriver
	MesosConfig *config.Config
	Settings    map[string]string
	Started     bool
	Task        *mesos.TaskInfo
	Kill        bool
}

// NewExecutor returns a properly configured sidecarExecutor.
func NewExecutor(mesosConfig *config.Config, settings map[string]string) *Firecracker {
	id, _ := util.GenUUID()

	return &Firecracker{
		Ctx:         context.TODO(),
		ID:          id,
		MesosConfig: mesosConfig,
		Settings:    settings,
		Kill:        false,
	}
}

// handleLaunch puts given task in unacked tasks and launches it
func (e *Firecracker) LaunchTask(taskInfo *mesos.TaskInfo) {

	commands := strings.Join(os.Args[1:], " ")

	logrus.WithField("func", "mesos.LaunchTask").Info("Handle Launch Event")

	e.Task = taskInfo

	logrus.WithField("func", "LaunchTask").Trace("commands: ", commands)
	logrus.WithField("func", "LaunchTask").Trace("TaskInfo: ", e.Task)

	util.Copy(e.Settings["FIRECRACKER_WORKDIR"]+"/rootfs.ext4", "/tmp/"+e.ID+"-rootfs.ext4")

	fcConfig := e.getFirecrackerConfig(e.ID)
	var err error

	// Create Machine
	e.Machine, err = firecracker.NewMachine(e.Ctx, fcConfig)
	if err != nil {
		logrus.WithField("func", "mesos.LaunchTask").Error("Could not create Firecracker machine: ", err.Error())
	}

	// Start Machine
	err = e.Machine.Start(e.Ctx)
	if err != nil {
		logrus.WithField("func", "mesos.LaunchTask").Error("Could not start Firecracker machine: ", err.Error())
	}

	if len(e.Machine.Cfg.NetworkInterfaces) > 0 {
		e.Started = true
		e.IP = e.Machine.Cfg.NetworkInterfaces[0].StaticConfiguration.IPConfiguration.IPAddr.IP
		logrus.WithField("func", "mesos.LaunchTask").Info("machine started with IP: ", e.IP)
		e.updateStatus(mesos.TASK_RUNNING)
		go e.heartbeatLoop()
	}
}

// KillTask is a Mesos callback that will try very hard to kill off a running
// task/container.
func (e *Firecracker) KillTask() {
	logrus.WithField("func", "mesos.KillTask").Info("Handle KillTask Event")
	logrus.WithField("func", "mesos.KillTask").Debug("", e.Task.TaskID)

	err := e.Machine.StopVMM()
	if err != nil {
		logrus.WithField("func", "mesos.KillTask").Error("Could not kill Firecracker machine: ", err.Error())
	}
	e.updateStatus(mesos.TASK_KILLED)
	e.Kill = true
}

// Heartbeat of the vmm-agent
func (e *Firecracker) Heartbeat() {
	select {
	case <-time.After(10 * time.Second):

		port := e.Settings["FIRECRACKER_AGENT_PORT"]
		client := &http.Client{}
		req, _ := http.NewRequest("GET", "http://"+e.IP.String()+":"+port+"/health", nil)
		req.Close = true
		res, err := client.Do(req)

		if err != nil {
			logrus.WithField("func", "mesos.HealthCheck").Error("Connection Error. ", err.Error())
			if e.Kill {
				logrus.WithField("func", "mesos.HealthCheck").Info("Exit")
				e.Driver.Stop()
			}
			return
		}

		defer res.Body.Close()

		if res.StatusCode != 200 {
			logrus.WithField("func", "mesos.HealthCheck").Error("HTTP Return not 200. Set TASK_FAILED: ", err.Error())
			e.updateStatus(mesos.TASK_FAILED)
			e.KillTask()
			return
		}
	}
}

func (e *Firecracker) updateStatus(state mesos.TaskState) {
	status := e.Driver.NewStatus(e.Task.TaskID)
	status.State = state.Enum()
	err := e.Driver.SendStatusUpdate(status)
	if err != nil {
		e.Driver.ThrowError(e.Task.TaskID, fmt.Errorf("error while updating task status"))
	}
}

// HeartbeatLoop is the main loop for the hearbeat
func (e *Firecracker) heartbeatLoop() {
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()
	for ; true; <-ticker.C {
		e.Heartbeat()
	}
}
