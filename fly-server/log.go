package main

import (
	"io"
	golog "log"
)

// 2009/01/23 01:23:23 DEBUG Something happened -- user="bob" id=123
type logger struct {
	debug  bool
	logger *golog.Logger
}

var log logger

func (l *logger) Init(debug bool, out io.Writer) {
	l.debug = debug
	l.logger = golog.New(out, "", golog.Ldate|golog.Ltime)
}

func (l *logger) Errorf(fmt string, v ...interface{}) {
	l.logger.Printf("ERROR "+fmt, v...)
}

func (l *logger) Infof(fmt string, v ...interface{}) {
	l.logger.Printf("INFO "+fmt, v...)
}

func (l *logger) Debugf(fmt string, v ...interface{}) {
	if l.debug {
		l.logger.Printf("DEBUG "+fmt, v...)
	}
}

func (l *logger) Fatalf(fmt string, v ...interface{}) {
	l.logger.Fatalf("FATAL "+fmt, v...)
}
