package infrastructure

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var supportLogTypes = []string{"debug", "info", "warn", "error"}

const (
	green string = "\x1b[97;104m"
	reset string = "\x1b[0m"
)

type ResourceReleaseFunc func() error

type finalError struct {
	err error

	module string

	severity string

	cancelFunc context.CancelFunc

	printStack bool

	exitAfterPrint bool
}

func (e *finalError) Error() string {
	return fmt.Sprintf("%+v", e.err)
}

type ProjectInfrastructure struct {
	errChan chan error

	errEndChan chan struct{}

	// Context for controlling resource release of ProjectInfrastructure
	cancel     context.Context
	cancelFunc context.CancelFunc

	// Control goroutine of project to gracefully exit
	GoroutineCancel     context.Context
	goroutineCancelFunc context.CancelFunc
	WaitGroup           sync.WaitGroup

	// Release of project resources
	releaseFunc ResourceReleaseFunc
}

func NewProjectManagement(_ctx context.Context, _optionFuncs ...OptionFunc) (*ProjectInfrastructure, error) {
	ctx := context.Background()
	if _ctx != nil {
		ctx = _ctx
	}

	options := DefaultOptions()
	for _, optFunc := range _optionFuncs {
		optFunc(&options)
	}

	PM := &ProjectInfrastructure{
		errChan:     make(chan error, options.ErrChanLen),
		errEndChan:  make(chan struct{}),
		releaseFunc: options.ReleaseFunc,
	}
	if err := PM.initLogrus(options); err != nil {
		return nil, err
	}
	PM.cancel, PM.cancelFunc = context.WithCancel(ctx)
	PM.GoroutineCancel, PM.goroutineCancelFunc = context.WithCancel(ctx)

	go PM.processError()

	return PM, nil
}

// Receive and handle error chain
func (pm *ProjectInfrastructure) processError() {
	for {
		select {
		case <-pm.errEndChan:
			logrus.Info("projectmanagement module stopped")
			return
		case _error := <-pm.errChan:
			if _error == nil {
				continue
			}

			_f_error, ok := _error.(*finalError)
			if !ok {
				logrus.Error(_error.Error())
				continue
			}

			if _f_error.err != nil {
				pm.logOutput(_f_error)
			}
		}
	}
}

// Release resources.
func (pm *ProjectInfrastructure) ResourceRelease() {
	pm.releaseFunc()

	pm.goroutineCancelFunc()
	pm.WaitGroup.Wait()

	close(pm.errEndChan)
	close(pm.errChan)

	time.Sleep(100 * time.Millisecond)
}

/*
Transmit the error chain to the exception handling module

@module: project module name

@severity: log level <debug/info/warn/error>

@err:	final error <error>

@exit_after_print: exit main program after printing the exception log <true/false>

@print_stack: print error chain, default severity is error <true/false>
*/
func (pm *ProjectInfrastructure) ErrorTransmit(_module, _severity string, _err error, _exit_after_print, _print_stack bool) {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("send error to closed channel: %+v", r)
		}
	}()

	if _exit_after_print {
		cancel, cancelFunc := context.WithCancel(pm.cancel)
		pm.errChan <- &finalError{
			err:            _err,
			cancelFunc:     cancelFunc,
			module:         _module,
			severity:       _severity,
			printStack:     _print_stack,
			exitAfterPrint: _exit_after_print,
		}
		<-cancel.Done()
		pm.ResourceRelease()
		os.Exit(1)
	}

	pm.errChan <- &finalError{
		err:            _err,
		cancelFunc:     nil,
		module:         _module,
		severity:       _severity,
		printStack:     _print_stack,
		exitAfterPrint: _exit_after_print,
	}
}

// Format error information.
func (pm *ProjectInfrastructure) logFormat(_err error, _module string) string {
	if len(_module) > 10 {
		_module = _module[:10]
	}
	log := fmt.Sprintf("%v %s %-10s %s %+v",
		time.Now().Format("2006-01-02 15:04:05"),
		green, _module, reset,
		_err.Error(),
	)
	return log
}

// Format error chain information.
func (pm *ProjectInfrastructure) errorStackMsg(_module string) string {
	if len(_module) > 10 {
		_module = _module[:10]
	}
	return fmt.Sprintf("%v %s %-10s %s",
		time.Now().Format("2006-01-02 15:04:05"),
		green, _module, reset,
	)
}

// Print the log and determine whether to print the complete error chain and exit after printing is complete.
func (pm *ProjectInfrastructure) logOutput(_err *finalError) {
	switch _err.severity {
	case "debug":
		if _err.printStack {
			logrus.Debugf(pm.errorStackMsg(_err.module)+"\n%+v", _err.err)
		} else {
			logrus.Debug(
				pm.logFormat(
					errors.Cause(_err.err),
					_err.module,
				),
			)
		}
	case "info":
		if _err.printStack {
			logrus.Infof(pm.errorStackMsg(_err.module)+"\n%+v", _err.err)
		} else {
			logrus.Info(
				pm.logFormat(
					errors.Cause(_err.err),
					_err.module,
				),
			)
		}
	case "warn":
		if _err.printStack {
			logrus.Warnf(pm.errorStackMsg(_err.module)+"\n%+v", _err.err)
		} else {
			logrus.Warn(
				pm.logFormat(
					errors.Cause(_err.err),
					_err.module,
				),
			)
		}
	case "error":
		if _err.printStack {
			logrus.Errorf(pm.errorStackMsg(_err.module)+"\n%+v", _err.err)
		} else {
			logrus.Error(
				pm.logFormat(
					errors.Cause(_err.err),
					_err.module,
				),
			)
		}
	default:
		logrus.Error(fmt.Sprintf("[unsupport error type: %s]", _err.severity) +
			pm.logFormat(
				errors.Cause(_err.err),
				_err.module,
			),
		)
	}

	if _err.exitAfterPrint {
		_err.cancelFunc()
	}
}

func (pm *ProjectInfrastructure) initLogrus(_opts ProjectInfrastructureOptions) error {
	logrus.SetOutput(_opts.LogOut)

	switch _opts.LogLevel {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		return errors.Errorf("invalid log level %s, valid values are %s", _opts.LogLevel, supportLogTypes)
	}
	return nil
}
