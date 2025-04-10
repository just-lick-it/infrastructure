package infrastructure

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	filerotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var supportLogTypes = []string{"debug", "info", "warn", "error"}

const (
	green string = "\x1b[97;104m"
	reset string = "\x1b[0m"
)

type ProjectInfrastructure struct {
	options *ProjectInfrastructureOptions

	// Context for controlling resource release of ProjectInfrastructure
	cancel     context.Context
	cancelFunc context.CancelFunc

	// Control goroutine of project to gracefully exit
	GoroutineCancel     context.Context
	goroutineCancelFunc context.CancelFunc
	WaitGroup           sync.WaitGroup

	// Release of project resources
	releaseFunc func() error
}

func NewProjectInfrastructure(_ctx context.Context, _optionFuncs ...OptionFunc) (*ProjectInfrastructure, error) {
	ctx := context.Background()
	if _ctx != nil {
		ctx = _ctx
	}

	options := DefaultOptions()
	for _, optFunc := range _optionFuncs {
		optFunc(&options)
	}

	PM := &ProjectInfrastructure{
		options:     &options,
		releaseFunc: options.ReleaseFunc,
	}
	if err := PM.initLogrus(options); err != nil {
		return nil, err
	}
	PM.cancel, PM.cancelFunc = context.WithCancel(ctx)
	PM.GoroutineCancel, PM.goroutineCancelFunc = context.WithCancel(ctx)
	return PM, nil
}

// Release resources.
func (pm *ProjectInfrastructure) ResourceRelease() {
	pm.releaseFunc()

	pm.goroutineCancelFunc()
	pm.WaitGroup.Wait()
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
			logrus.Errorf("%+v", r)
		}
	}()

	pm.WaitGroup.Add(1)
	defer pm.WaitGroup.Done()

	if _exit_after_print {
		pm.logOutput(_module, _severity, _err, _print_stack)
		pm.WaitGroup.Done()
		pm.ResourceRelease()
		os.Exit(1)
	}
	pm.logOutput(_module, _severity, _err, _print_stack)
}

// Format error information.
func (pm *ProjectInfrastructure) logFormat(_err error, _module string) string {
	var log string

	if len(_module) > 10 {
		_module = _module[:10]
	}

	switch pm.options.LogOut {
	case "stdout":
		log = fmt.Sprintf("%v %s %-10s %s %+v",
			time.Now().Format("2006-01-02 15:04:05"),
			green,
			_module,
			reset,
			_err.Error(),
		)
	case "file":
		log = fmt.Sprintf("%v %-10s %+v",
			time.Now().Format("2006-01-02 15:04:05"),
			_module,
			_err.Error(),
		)
	}
	return log
}

// Format error chain information.
func (pm *ProjectInfrastructure) errorStackMsg(_module string) string {
	var log string

	if len(_module) > 10 {
		_module = _module[:10]
	}

	switch pm.options.LogOut {
	case "stdout":
		log = fmt.Sprintf("%v %s %-10s %s",
			time.Now().Format("2006-01-02 15:04:05"),
			green,
			_module,
			reset,
		)
	case "file":
		log = fmt.Sprintf("%v %-10s",
			time.Now().Format("2006-01-02 15:04:05"),
			_module,
		)
	}
	return log
}

// Print the log and determine whether to print the complete error chain.
func (pm *ProjectInfrastructure) logOutput(_module, _severity string, _err error, _print_stack bool) {
	switch _severity {
	case "debug":
		if _print_stack {
			logrus.Debugf(pm.errorStackMsg(_module)+"\n%+v", _err)
		} else {
			logrus.Debug(
				pm.logFormat(
					errors.Cause(_err),
					_module,
				),
			)
		}
	case "info":
		if _print_stack {
			logrus.Infof(pm.errorStackMsg(_module)+"\n%+v", _err)
		} else {
			logrus.Info(
				pm.logFormat(
					errors.Cause(_err),
					_module,
				),
			)
		}
	case "warn":
		if _print_stack {
			logrus.Warnf(pm.errorStackMsg(_module)+"\n%+v", _err)
		} else {
			logrus.Warn(
				pm.logFormat(
					errors.Cause(_err),
					_module,
				),
			)
		}
	case "error":
		if _print_stack {
			logrus.Errorf(pm.errorStackMsg(_module)+"\n%+v", _err)
		} else {
			logrus.Error(
				pm.logFormat(
					errors.Cause(_err),
					_module,
				),
			)
		}
	default:
		logrus.Error(fmt.Sprintf("[unsupport error type: %s]", _severity) +
			pm.logFormat(
				errors.Cause(_err),
				_module,
			),
		)
	}
}

func (pm *ProjectInfrastructure) initLogrus(_opts ProjectInfrastructureOptions) error {
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	switch _opts.LogOut {
	case "stdout":
		logrus.SetOutput(os.Stdout)
	case "file":
		w, err := filerotatelogs.New(
			_opts.LogPath,
			filerotatelogs.WithRotationCount(uint(_opts.LogMaxFileNum)),
			filerotatelogs.WithRotationSize(int64(_opts.LogMaxFileSize)),
		)
		if err != nil {
			return err
		}
		logrus.SetOutput(w)
	default:
		logrus.Warnf("unknown log output type: %s, use default stdout", _opts.LogOut)
		logrus.SetOutput(os.Stdout)
	}

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
