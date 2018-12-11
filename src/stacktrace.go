package main

import (
	"runtime"

	"github.com/getsentry/raven-go"
	"github.com/pkg/errors"
)

func NewRavenStackTrace(client *raven.Client, myerr error, skip int) *raven.Stacktrace {
	st := getErrorStackTraceConverted(myerr, 3, client.IncludePaths())
	if st == nil {
		st = raven.NewStacktrace(skip, 3, client.IncludePaths())
	}
	return st
}

func getErrorStackTraceConverted(err error, context int, appPackagePrefixes []string) *raven.Stacktrace {
	st := getErrorCauseStackTrace(err)
	if st == nil {
		return nil
	}
	return convertStackTrace(st, context, appPackagePrefixes)
}

func getErrorCauseStackTrace(err error) errors.StackTrace {
	// This code is inspired by github.com/pkg/errors.Cause().
	var st errors.StackTrace
	for err != nil {
		s := getErrorStackTrace(err)
		if s != nil {
			st = s
		}
		err = getErrorCause(err)
	}
	return st
}

func convertStackTrace(st errors.StackTrace, context int, appPackagePrefixes []string) *raven.Stacktrace {
	// This code is borrowed from github.com/getsentry/raven-go.NewStacktrace().
	var frames []*raven.StacktraceFrame
	for _, f := range st {
		frame := convertFrame(f, context, appPackagePrefixes)
		if frame != nil {
			frames = append(frames, frame)
		}
	}
	if len(frames) == 0 {
		return nil
	}
	for i, j := 0, len(frames)-1; i < j; i, j = i+1, j-1 {
		frames[i], frames[j] = frames[j], frames[i]
	}
	return &raven.Stacktrace{Frames: frames}
}

func convertFrame(f errors.Frame, context int, appPackagePrefixes []string) *raven.StacktraceFrame {
	// This code is borrowed from github.com/pkg/errors.Frame.
	pc := uintptr(f) - 1
	fn := runtime.FuncForPC(pc)
	var file string
	var line int
	if fn != nil {
		file, line = fn.FileLine(pc)
	} else {
		file = "unknown"
	}
	return raven.NewStacktraceFrame(pc, file, line, context, appPackagePrefixes)
}

func getErrorStackTrace(err error) errors.StackTrace {
	ster, ok := err.(interface {
		StackTrace() errors.StackTrace
	})
	if !ok {
		return nil
	}
	return ster.StackTrace()
}

func getErrorCause(err error) error {
	cer, ok := err.(interface {
		Cause() error
	})
	if !ok {
		return nil
	}
	return cer.Cause()
}
