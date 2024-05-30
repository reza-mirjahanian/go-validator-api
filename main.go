package main

import (
	"log"
)

// checkIfErr is a helper function to check if an error is not nil
func checkIfErr(err error) {
	if err != nil {
		log.Fatal("error occurred: %s" + err.Error())
	}
}

func main() {
	startRestAPIs()
}
