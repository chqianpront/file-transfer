package main

import (
	"fmt"
	"os"

	"chen.com/file-trans/server"
)

func main() {
	osArgsLen := len(os.Args)
	if osArgsLen < 2 {
		fmt.Printf("arguments is not enough, current is %d", osArgsLen)
		os.Exit(1)
	}
	actionType := os.Args[1]
	clientType := os.Args[2]
	environmentType := os.Args[3]
	if len(environmentType) <= 0 {
		environmentType = "dev"
	}
	fmt.Printf("action is %s, client type is %s, environment is %s\n", actionType, clientType, environmentType)
	if actionType == "start" {
		if clientType == "server" {
			server.StartServer(environmentType)
		} else if clientType == "client" {

		} else {
			fmt.Printf("unkown client type %s", clientType)
		}
	} else if actionType == "stop" {

	} else {
		fmt.Printf("unkown action type %s", actionType)
		os.Exit(1)
	}
}
