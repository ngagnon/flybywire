package logging

import (
	"io"
	"log"
)

// 2009/01/23 01:23:23 DEBUG Something happened -- user="bob" id=123
var debugMode = false

func Configure(debug bool, out io.Writer) {
	debugMode = debug
	log.SetOutput(out)
	log.SetPrefix("")
	log.SetFlags(log.Ldate | log.Ltime)
}

func Errorf(fmt string, v ...interface{}) {
	log.Printf("ERROR "+fmt, v...)
}

func Infof(fmt string, v ...interface{}) {
	log.Printf("INFO "+fmt, v...)
}

func Debugf(fmt string, v ...interface{}) {
	if debugMode {
		log.Printf("DEBUG "+fmt, v...)
	}
}

func Fatalf(fmt string, v ...interface{}) {
	log.Fatalf("FATAL "+fmt, v...)
}
