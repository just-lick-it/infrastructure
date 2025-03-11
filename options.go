package infrastructure

import (
	"io"
	"os"
)

var (
	_defaultLogLevel    = "debug"
	_defaultLogOut      = os.Stdout
	_defaultLogPath     = "./project.log"
	_defaultMaxFileNum  = 10
	_defaultMaxFileSize = 10485760
	_defaultErrChanLen  = 20
)

type OptionFunc func(*ProjectInfrastructureOptions)

type ProjectInfrastructureOptions struct {
	LogLevel       string
	LogOut         io.Writer
	LogPath        string
	LogMaxFileNum  uint
	LogMaxFileSize uint

	ErrChanLen uint

	ReleaseFunc ResourceReleaseFunc
}

func DefaultOptions() ProjectInfrastructureOptions {
	return ProjectInfrastructureOptions{
		LogLevel:       _defaultLogLevel,
		LogOut:         _defaultLogOut,
		LogPath:        _defaultLogPath,
		LogMaxFileNum:  uint(_defaultMaxFileNum),
		LogMaxFileSize: uint(_defaultMaxFileSize),
		ErrChanLen:     uint(_defaultErrChanLen),
		ReleaseFunc:    nil,
	}
}

func WithLogLevel(_level string) OptionFunc {
	return func(o *ProjectInfrastructureOptions) {
		o.LogLevel = _level
	}
}

func WithLogOutput(_out io.Writer) OptionFunc {
	return func(o *ProjectInfrastructureOptions) {
		o.LogOut = _out
	}
}

func WithLogPath(_path string) OptionFunc {
	return func(o *ProjectInfrastructureOptions) {
		o.LogPath = _path
	}
}

func WithLogMaxFileNum(_num uint) OptionFunc {
	return func(o *ProjectInfrastructureOptions) {
		o.LogMaxFileNum = _num
	}
}

func WithLogMaxFileSize(_size uint) OptionFunc {
	return func(o *ProjectInfrastructureOptions) {
		o.LogMaxFileSize = _size
	}
}

func WithResourceRleaseFunc(_func ResourceReleaseFunc) OptionFunc {
	return func(o *ProjectInfrastructureOptions) {
		o.ReleaseFunc = _func
	}
}

func WithErrChanLen(_len uint) OptionFunc {
	return func(o *ProjectInfrastructureOptions) {
		o.ErrChanLen = _len
	}
}
