package main

import (
	"fmt"

	"github.com/matheush9/tcp-server/server"
)

func main() {
	sc := server.ServerConfig{ConnectionTimeout: 60, IPV4Address: "127.0.0.1", Port: 34093}
	sc.SetupServer()
	clientRequestChan := make(chan server.ClientRequest)
	go sc.HandleConnections(clientRequestChan)
	for clientRequest := range clientRequestChan {
		fmt.Println(clientRequest) //test
	}
}
