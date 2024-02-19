package main

import (
	"log"
	"syscall"
	"time"
	"web_server/netUtils"
)

const CONNECTION_TIMEOUT float64 = 60 //seconds
const KILOBYTE = 1024                 //bytes

func main() {
	IPAddr, err := netUtils.IPStringToIPBytes("127.0.0.1")
	if err != nil {
		log.Panic(err)
	}
	serverFileDescriptor := setupServer(IPAddr, 34093)
	defer func() {
		if r := recover(); r != nil {
			log.Println("the server encountered a panic:", r)
		}
		closeSocket(serverFileDescriptor)
	}()
	clientFileDescriptorChan := make(chan int)
	go waitClientConnections(serverFileDescriptor, clientFileDescriptorChan)
	for clientFileDescriptor := range clientFileDescriptorChan {
		go handleRequest(clientFileDescriptor)
	}
}

func handleRequest(clientFileDescriptor int) {
	request := readRequest(clientFileDescriptor)
	if len(request) > 0 {
		log.Print("request received: ", string(request))
		sendResponse(clientFileDescriptor, request)
	}
	closeSocket(clientFileDescriptor)
}

func sendResponse(clientFileDescriptor int, request []byte) {
	response := append([]byte("I get your message: "), request...)
	_, err := syscall.Write(clientFileDescriptor, response)
	if err != nil {
		log.Panic(err)
	}
}

func setupServer(IPAddress netUtils.IP, port int) (serverFileDescriptor int) {
	serverFileDescriptor, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
	if err != nil {
		log.Panic(err)
	}
	for counter := 0; counter < 60; counter++ {
		if err = syscall.Bind(serverFileDescriptor, &syscall.SockaddrInet4{Addr: IPAddress, Port: port}); err == syscall.EADDRINUSE {
			log.Println("address in use, trying to bind again in 5 secs until 5 minutes")
			time.Sleep(5 * time.Second)
		} else if err != nil {
			log.Panic(err)
		} else {
			break
		}
	}
	err = syscall.Listen(serverFileDescriptor, 1)
	if err != nil {
		log.Panic(err)
	}
	log.Println("the server is listening for connections...")
	return serverFileDescriptor
}

func waitClientConnections(serverFileDescriptor int, clientFileDescriptorChan chan int) {
	for {
		clientFileDescriptor, clientSocket, err := syscall.Accept(serverFileDescriptor)
		if err != nil {
			log.Panic(err)
		}
		clientFileDescriptorChan <- clientFileDescriptor
		log.Println("connected with ", clientSocket)
	}
}

func readRequest(clientFileDescriptor int) (request []byte) {
	clientInputChan := make(chan []byte)

	go func(chan []byte) {
		buffer := make([]byte, KILOBYTE/2)
		for {
			bytesRead, err := syscall.Read(clientFileDescriptor, buffer)
			if err != nil {
				log.Panic(err)
			}
			request = append(request, buffer[:bytesRead]...)
			clientInputChan <- request
			if len(buffer) != bytesRead || bytesRead == 0 {
				close(clientInputChan)
				return
			}
		}
	}(clientInputChan)

	for {
		select {
		case _, ok := <-clientInputChan:
			if !ok {
				return
			}
		case <-time.After(time.Second * time.Duration(CONNECTION_TIMEOUT)):
			log.Println("timeout reached")
			return
		}
	}
}

func closeSocket(fileDescriptor int) {
	if err := syscall.Close(fileDescriptor); err != nil {
		log.Panic(err)
	}
	log.Printf("socket with file descriptor %d is closed", fileDescriptor)
}
