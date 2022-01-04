package main

import "log"

func ChkFatal(e error, msg string) {
	if e != nil {
		log.Fatalln(msg+":", e)
	}
}

func ChkPrint(err error) {
	if err != nil {
		log.Println(err)
	}
}
