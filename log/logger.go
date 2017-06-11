package log

import (
	"log"
	"runtime"
	"strconv"

	"cloud.google.com/go/logging"

	"golang.org/x/net/context"
	appenginelog "google.golang.org/appengine/log"
)

type severity int

const (
	Critical severity = iota
	Error
	Warning
	Info
	Debug
)

// DefaultLogger is default logger.
var DefaultLogger Logger

func init() {
	if DefaultLogger == nil {
		DefaultLogger = &StdLogger{}
	}
}

// Logger is interface output log.
type Logger interface {
	Log(svr severity, format string, args ...interface{})
}

// StdLogger is logger output log to stdout.
type StdLogger struct{}

// Log outputs log to stdout.
func (s *StdLogger) Log(svr severity, format string, args ...interface{}) {
	log.Printf(format, args...)
}

// AppEngineLogger is logger output log to appengine logging.
type AppEngineLogger struct{ ctx context.Context }

// Log outputs log to appengine logging.
func (a *AppEngineLogger) Log(svr severity, format string, args ...interface{}) {
	select {
	case <-a.ctx.Done():
		appenginelog.Errorf(a.ctx, getLogMessageContainRuntimeInfo("context is canceled"))
	default:
		msg := getLogMessageContainRuntimeInfo(format)
		switch svr {
		case Critical:
			appenginelog.Criticalf(a.ctx, msg, args...)
		case Error:
			appenginelog.Errorf(a.ctx, msg, args...)
		case Warning:
			appenginelog.Warningf(a.ctx, msg, args...)
		case Info:
			appenginelog.Infof(a.ctx, msg, args...)
		case Debug:
			appenginelog.Debugf(a.ctx, msg, args...)
		}
	}
}

// StackdriverLogging is logger output log to stackdriver.
type StackdriverLogging struct {
	c *logging.Client
}

func getLogMessageContainRuntimeInfo(format string) string {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	}
	return file + ":" + strconv.Itoa(line) + " - " + format
}
