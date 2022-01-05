package main

import (
	"log"
)

const (
	printError int = iota
	fatalError
)

//**********************Error Handling************************

func handleErrPrint(out ...interface{}) interface{} {
	if out[1] != nil {
		chkPrint(out[1].(error))
	}
	return out[0]
}

func handleErrFatal(out ...interface{}) interface{} {
	chkFatal(out[1].(error))
	return out[0]
}

func chkFatal(e error) {
	if e != nil {
		log.Fatalln(e)
	}
}

func chkPrint(err error) {
	if err != nil {
		log.Println(err)
	}
}
