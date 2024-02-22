package main

import "github.com/matheush9/tcp-server/server"

func main() {
	ts := server.ServerConfig{ConnectionTimeout: 60, IPV4Address: "127.0.0.1", Port: 34093}
	ts.SetupServer()
	ts.HandleConnections()
}
