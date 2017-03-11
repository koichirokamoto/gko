package gko

import (
	"runtime"
	"strconv"

	"golang.org/x/net/context"
	"google.golang.org/appengine/log"
)

// InfoLog outputs info log.
func InfoLog(ctx context.Context, format string, args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	}
	log.Infof(ctx, file+":"+strconv.Itoa(line)+" - "+format, args...)
}

// DebugLog outputs debug log.
func DebugLog(ctx context.Context, format string, args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	}
	log.Debugf(ctx, file+":"+strconv.Itoa(line)+" - "+format, args...)
}

// CriticalLog outputs critical log.
func CriticalLog(ctx context.Context, format string, args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	}
	log.Criticalf(ctx, file+":"+strconv.Itoa(line)+" - "+format, args...)
}

// ErrorLog outputs error log.
func ErrorLog(ctx context.Context, format string, args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	}
	log.Errorf(ctx, file+":"+strconv.Itoa(line)+" - "+format, args...)
}

// WarningLog outputs warning log.
func WarningLog(ctx context.Context, format string, args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	}
	log.Warningf(ctx, file+":"+strconv.Itoa(line)+" - "+format, args...)
}
