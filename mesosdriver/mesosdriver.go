package mesosdriver

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/AVENTER-UG/util/util"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/backoff"
	"github.com/mesos/mesos-go/api/v1/lib/encoding"
	"github.com/mesos/mesos-go/api/v1/lib/encoding/codecs"
	"github.com/mesos/mesos-go/api/v1/lib/executor"
	"github.com/mesos/mesos-go/api/v1/lib/executor/calls"
	"github.com/mesos/mesos-go/api/v1/lib/executor/config"
	"github.com/mesos/mesos-go/api/v1/lib/executor/events"
	"github.com/mesos/mesos-go/api/v1/lib/httpcli"
	"github.com/mesos/mesos-go/api/v1/lib/httpcli/httpexec"
	"github.com/pborman/uuid"
	logrus "github.com/sirupsen/logrus"
)

const (
	agentAPIPath      = "/api/v1/executor"
	driverHttpTimeout = 10 * time.Second
)

// A TaskDelegate is responsible for launching and killing tasks
type TaskDelegate interface {
	LaunchTask(taskInfo *mesos.TaskInfo)
	KillTask()
	Heartbeat()
}

// The ExecutorDriver does all the work of interacting with Mesos and the Agent
// and calls back to the TaskDelegate to handle starting and stopping tasks.
type ExecutorDriver struct {
	cli            calls.Sender
	cfg            *config.Config
	framework      mesos.FrameworkInfo
	executor       mesos.ExecutorInfo
	agent          mesos.AgentInfo
	unackedTasks   map[mesos.TaskID]mesos.TaskInfo
	unackedUpdates map[string]executor.Call_Update
	quitChan       chan struct{}
	subscriber     calls.SenderFunc

	delegate TaskDelegate
}

func (driver *ExecutorDriver) maybeReconnect() <-chan struct{} {
	if driver.cfg.Checkpoint {
		return backoff.Notifier(1*time.Second, driver.cfg.SubscriptionBackoffMax*3/4, nil)
	}
	return nil
}

// unacknowledgedTasks generates the value of the UnacknowledgedTasks field of
// a Subscribe call.
func (driver *ExecutorDriver) unacknowledgedTasks() (result []mesos.TaskInfo) {
	if n := len(driver.unackedTasks); n > 0 {
		result = make([]mesos.TaskInfo, 0, n)
		for k := range driver.unackedTasks {
			result = append(result, driver.unackedTasks[k])
		}
	}
	return
}

// unacknowledgedUpdates generates the value of the UnacknowledgedUpdates field
// of a Subscribe call.
func (driver *ExecutorDriver) unacknowledgedUpdates() (result []executor.Call_Update) {
	if n := len(driver.unackedUpdates); n > 0 {
		result = make([]executor.Call_Update, 0, n)
		for k := range driver.unackedUpdates {
			result = append(result, driver.unackedUpdates[k])
		}
	}
	return
}

// eventLoop is the main handler event loop of the driver. Called from Run()
func (driver *ExecutorDriver) eventLoop(decoder encoding.Decoder, h events.Handler) error {
	var err error

	logrus.WithField("func", "mesosdriver.eventLoop").Info("Entering event loop")
	ctx := context.Background()

	event := make(chan error)

	go func() {
		for {
			var e executor.Event
			if err = decoder.Decode(&e); err == nil {
				err = h.HandleEvent(ctx, &e)
			} else {
				logrus.WithField("func", "mesosdriver.eventLoop").Debug("Event loop error: ", err.Error())
			}

			select {
			case event <- err:
			case <-driver.quitChan:
				return
			}
		}
	}()

OUTER:
	for {
		select {
		case <-driver.quitChan:
			logrus.WithField("func", "mesosdriver.eventLoop").Info("Event loop canceled")
			return nil
		case err = <-event:
			if err != nil {
				break OUTER
			}
		}
	}

	logrus.Info("Exiting event loop")

	return err
}

// buildEventHandler returns an events.Handler that has been set up with
// callback functions for each event type.
func (driver *ExecutorDriver) buildEventHandler() events.Handler {
	return events.HandlerFuncs{
		executor.Event_SUBSCRIBED: func(_ context.Context, e *executor.Event) error {
			logrus.Info("Executor subscribed to events")
			driver.framework = e.Subscribed.FrameworkInfo
			driver.executor = e.Subscribed.ExecutorInfo
			driver.agent = e.Subscribed.AgentInfo
			return nil
		},

		executor.Event_LAUNCH: func(_ context.Context, e *executor.Event) error {
			driver.unackedTasks[e.Launch.Task.TaskID] = e.Launch.Task
			driver.delegate.LaunchTask(&e.Launch.Task)
			return nil
		},

		executor.Event_KILL: func(_ context.Context, e *executor.Event) error {
			logrus.Infof("Received kill from Mesos for %s", e.Kill.TaskID.Value)
			driver.delegate.KillTask()
			return nil
		},

		executor.Event_ACKNOWLEDGED: func(_ context.Context, e *executor.Event) error {
			logrus.WithField("func", "mesosdriver.buildEventHandler").Infof("Acknowledged: %s", e.Acknowledged.TaskID.Value)
			delete(driver.unackedTasks, e.Acknowledged.TaskID)
			delete(driver.unackedUpdates, string(e.Acknowledged.UUID))
			return nil
		},

		executor.Event_MESSAGE: func(_ context.Context, e *executor.Event) error {
			logrus.WithField("func", "mesosdriver.buildEventHandler").Debugf("MESSAGE: received %d bytes of message data", len(e.Message.Data))
			return nil
		},

		executor.Event_SHUTDOWN: func(_ context.Context, e *executor.Event) error {
			logrus.WithField("func", "mesosdriver.buildEventHandler").Info("Shutting down the executor")
			driver.Stop()
			return nil
		},

		executor.Event_ERROR: func(_ context.Context, e *executor.Event) error {
			logrus.WithField("func", "mesosdriver.buildEventHandler").Error("ERROR received")
			return errors.New(
				"received abort from Mesos, will attempt to re-subscribe",
			)
		},

		executor.Event_HEARTBEAT: func(_ context.Context, e *executor.Event) error {
			// We don't process heartbeats. In theory we ought to count how many we get
			// and force reconnect if we don't get one. But we already watch the
			// connection so it's just redundant. Ignore.
			logrus.WithField("func", "mesosdriver.buildEventHandler").Debug("Heartbeat received")
			driver.delegate.Heartbeat()
			return nil
		},
	}.Otherwise(func(_ context.Context, e *executor.Event) error {
		logrus.WithField("func", "mesosdriver.buildEventHandler").Error("unexpected event", e)
		return nil
	})
}

// SendStatusUpdate takes a new Mesos status and relays it to the agent
func (driver *ExecutorDriver) SendStatusUpdate(status mesos.TaskStatus) error {
	upd := calls.Update(status)
	resp, err := driver.cli.Send(context.TODO(), calls.NonStreaming(upd))
	if resp != nil {
		resp.Close()
	}
	if err != nil {
		logrus.WithField("func", "mesosdriver.SendStatusUpdate").Errorf("failed to send update: %+v", err)
		logDebugJSON(upd)
	} else {
		driver.unackedUpdates[string(status.UUID)] = *upd.Update
	}
	return err
}

// marshalJSON is a narrowly scoped interface used to allow logDebugJSON to
// properly format most Mesos messages.
type marshalJSON interface {
	MarshalJSON() ([]byte, error)
}

// logDebugJson prints failed messages to the logger when we can't talk to the
// Agent correctly.
func logDebugJSON(mk marshalJSON) {
	b, err := mk.MarshalJSON()
	if err == nil {
		logrus.WithField("func", "mesosdriver.logDebugJSON").Trace(string(b))
	}
}

// NewStatus returns a properly configured Mesos.TaskStatus
func (driver *ExecutorDriver) NewStatus(id mesos.TaskID) mesos.TaskStatus {
	return mesos.TaskStatus{
		TaskID:     id,
		Source:     mesos.SOURCE_EXECUTOR.Enum(),
		ExecutorID: &driver.executor.ExecutorID,
		UUID:       []byte(uuid.NewRandom()),
	}
}

func (driver *ExecutorDriver) Stop() {
	close(driver.quitChan)
}

// NewExecutorDriver returns a properly configured ExecutorDriver
func NewExecutorDriver(mesosConfig *config.Config, delegate TaskDelegate) *ExecutorDriver {
	scheme := "http"
	if util.Getenv("LIBPROCESS_SSL_ENABLED", "false") == "true" {
		scheme = "https"
	}

	agentApiUrl := url.URL{
		Scheme: scheme,
		Host:   mesosConfig.AgentEndpoint,
		Path:   agentAPIPath,
	}

	httpConfig := []httpcli.ConfigOpt{
		httpcli.Transport(func(t *http.Transport) {
			t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}),
	}

	http := httpcli.New(
		httpcli.Endpoint(agentApiUrl.String()),
		httpcli.Codec(codecs.ByMediaType[codecs.MediaTypeProtobuf]),
		httpcli.Do(httpcli.With(httpConfig[0])),
	)

	callOptions := executor.CallOptions{
		calls.Framework(mesosConfig.FrameworkID),
		calls.Executor(mesosConfig.ExecutorID),
	}

	driver := &ExecutorDriver{
		cli: calls.SenderWith(
			httpexec.NewSender(http.Send),
			callOptions...,
		),
		unackedTasks:   make(map[mesos.TaskID]mesos.TaskInfo),
		unackedUpdates: make(map[string]executor.Call_Update),
		delegate:       delegate,
		cfg:            mesosConfig,
		quitChan:       make(chan struct{}),
	}

	driver.subscriber = calls.SenderWith(
		httpexec.NewSender(http.Send, httpcli.Close(true)),
		callOptions...,
	)

	return driver
}

// Run makes sure we're subscribed to events, and restarts the event loop until
// we're told to stop.
func (driver *ExecutorDriver) Run() error {
	shouldReconnect := driver.maybeReconnect()

	disconnectTime := time.Now()
	handler := driver.buildEventHandler()

	for {
		// Function block to ensure response is closed
		func() {
			subscribe := calls.Subscribe(
				driver.unacknowledgedTasks(),
				driver.unacknowledgedUpdates(),
			)

			logrus.WithField("func", "mesosdriver.Run").Info("Subscribing to agent for events")

			resp, err := driver.subscriber.Send(
				context.TODO(),
				calls.NonStreaming(subscribe),
			)

			if resp != nil {
				defer resp.Close()
			}

			if err != nil && err != io.EOF {
				logrus.WithField("func", "mesosdriver.Run").Error(err.Error())
				return
			}

			// We're connected, start decoding events
			err = driver.eventLoop(resp, handler)
			disconnectTime = time.Now()

			if err != nil && err != io.EOF {
				logrus.WithField("func", "mesosdriver.Run").Error(err.Error())
				return
			}

			logrus.WithField("func", "mesosdriver.Run").Info("Disconnected from Agent")
		}()

		select {
		case <-driver.quitChan:
			logrus.WithField("func", "mesosdriver.Run").Info("Shutting down gracefully because we were told to")
			return nil
		default:
		}

		if !driver.cfg.Checkpoint {
			logrus.WithField("func", "mesosdriver.Run").Info("Exiting gracefully because framework checkpointing is NOT enabled")
			return nil
		}

		if time.Since(disconnectTime) > driver.cfg.RecoveryTimeout {
			return fmt.Errorf(
				"Failed to re-establish subscription with agent within %v, aborting",
				driver.cfg.RecoveryTimeout,
			)
		}

		logrus.WithField("func", "mesosdriver.Run").Info("Waiting for reconnect timeout")

		<-shouldReconnect // wait for some amount of time before retrying subscription
	}
}

// ThrowError will create a error object and send state update to mesos
func (driver *ExecutorDriver) ThrowError(taskID mesos.TaskID, err error) {
	message := err.Error()
	status := driver.NewStatus(taskID)
	status.State = mesos.TASK_FAILED.Enum()
	status.Message = &message

	driver.SendStatusUpdate(status)
}
