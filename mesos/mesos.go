package mesos

import (
	"context"
	"net"

	"github.com/AVENTER-UG/mesos-firecracker-executor/mesosdriver"
	util "github.com/AVENTER-UG/util"
	"github.com/firecracker-microvm/firecracker-go-sdk"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
	"github.com/sirupsen/logrus"
)

type ExecDriver interface {
	getFirecrackerConfig(vmmID string) (firecracker.Config, error)
}

type Firecracker struct {
	Ctx     context.Context
	ID      string
	Machine *firecracker.Machine
	IP      net.IP
	Socket  string
	Driver  *mesosdriver.ExecutorDriver
}

// NewExecutor returns a properly configured sidecarExecutor.
func NewExecutor() *Firecracker {
	id, _ := util.GenUUID()

	return &Firecracker{
		Ctx: context.TODO(),
		ID:  id,
	}
}

// handleLaunch puts given task in unacked tasks and launches it
func (e *Firecracker) LaunchTask(taskInfo *mesos.TaskInfo) {
	logrus.Info("Handled LAUNCH event")
	logrus.Debug(taskInfo)

	util.Copy("/usr/libexec/mesos/rootfs.ext4", "/tmp/"+e.ID+"-rootfs.ext4")

	fcConfig, err := getFirecrackerConfig(e.ID)
	if err != nil {
		logrus.WithField("func", "handleLaunch").Error("Get Firecracker Config: ", err.Error())
	}

	e.Machine, err = firecracker.NewMachine(e.Ctx, fcConfig)
	if err != nil {
		logrus.WithField("func", "handleLaunch").Error("Could not create Firecracker machine: ", err.Error())
	}

	err = e.Machine.Start(e.Ctx)
	if err != nil {
		logrus.WithField("func", "handleLaunch").Error("Could not start Firecracker machine: ", err.Error())
	}

	//logrus.WithField("ip", vmm.Machine.Cfg.NetworkInterfaces[0].StaticConfiguration.IPConfiguration.IPAddr.IP).Info("machine started")
}

// KillTask is a Mesos callback that will try very hard to kill off a running
// task/container.
func (exec *Firecracker) KillTask(taskID *mesos.TaskID) {
	logrus.Info("Handled Task Kill event")
	logrus.Debug(taskID)
}
