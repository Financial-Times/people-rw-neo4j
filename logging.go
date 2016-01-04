package main

import (
	"io"
	"log"
)

var (
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

func InitLogging(
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer) {

	flags := log.Ldate | log.Ltime | log.Lshortfile

	Info = log.New(infoHandle,
		"INFO ", flags)

	Warning = log.New(warningHandle,
		"WARNING ", flags)

	Error = log.New(errorHandle,
		"ERROR ", flags)
}
