package main

import (
	"log"
	"syscall"
	"time"
	"web_server/netUtils"
)

type tcpServer struct {
	connectionTimeout    float64
	IPV4Address          string
	port                 int
	serverFileDescriptor int
	clientAddresses      map[int]syscall.SockaddrInet4
}

const CONNECTION_TIMEOUT float64 = 60 //seconds
const KILOBYTE = 1024                 //bytes

func main() {
	IPAddr, err := netUtils.IPStringToIPBytes("127.0.0.1")
	if err != nil {
		log.Fatal("the IP provided is not valid: ", err)
	}
	serverFileDescriptor, err := setupServer(IPAddr, 34093)
	if err != nil {
		log.Fatal("error while booting the server: ", err)
	}
	defer func() {
		err = closeSocket(serverFileDescriptor)
		if err != nil {
			log.Print("error while closing the server socket: ", err)
		}
	}()
	clientFileDescriptorChan := make(chan int)
	go waitClientConnections(serverFileDescriptor, clientFileDescriptorChan)
	for clientFileDescriptor := range clientFileDescriptorChan {
		go handleRequest(clientFileDescriptor)
	}
}

func (ts tcpServer) handleRequest(clientFileDescriptor int) {
	defer func() {
		err := ts.closeSocket(clientFileDescriptor)
		if err != nil {
			log.Printf("error while trying to close the client socket with IP: %v and port: %d: %v",
				ts.clientAddresses[clientFileDescriptor].Addr,
				ts.clientAddresses[clientFileDescriptor].Port,
				err)
		}
	}()
	request, err := readRequest(clientFileDescriptor)
	if err != nil {
		log.Printf("error while trying to read the request from IP: %v and port: %d: %v",
			ts.clientAddresses[clientFileDescriptor].Addr,
			ts.clientAddresses[clientFileDescriptor].Port,
			err)
		return
	}
	if len(request) > 0 {
		log.Print("request received: ", string(request))
		err = sendResponse(clientFileDescriptor, request)
		if err != nil {
			log.Printf("error while trying to send a response to IP: %v and port: %d: %v",
				ts.clientAddresses[clientFileDescriptor].Addr,
				ts.clientAddresses[clientFileDescriptor].Port,
				err)
		}
	}
}

func sendResponse(clientFileDescriptor int, request []byte) error {
	response := append([]byte("I get your message: "), request...)
	_, err := syscall.Write(clientFileDescriptor, response)
	if err != nil {
		return err
	}
	return nil
}

func setupServer(IPAddress netUtils.IP, port int) (serverFileDescriptor int, err error) {
	serverFileDescriptor, err = syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
	if err != nil {
		return serverFileDescriptor, err
	}
	for counter := 0; counter < 60; counter++ {
		if err = syscall.Bind(serverFileDescriptor, &syscall.SockaddrInet4{Addr: IPAddress, Port: port}); err == syscall.EADDRINUSE {
			log.Println("address in use, trying to bind again in 5 secs until 5 minutes")
			time.Sleep(5 * time.Second)
		} else if err != nil {
			return serverFileDescriptor, err
		} else {
			break
		}
	}
	err = syscall.Listen(serverFileDescriptor, 1)
	if err != nil {
		return serverFileDescriptor, err
	}
	log.Println("the server is listening for connections...")
	return serverFileDescriptor, nil
}

func (ts *tcpServer) listenClientConnections(serverFileDescriptor int, clientFileDescriptorChan chan int) {
	log.Println("the server is listening for connections...")
	for {
		clientFileDescriptor, clientSocket, err := syscall.Accept(serverFileDescriptor)
		if err != nil {
			log.Print("error while trying to accept a connection: ", err)
			continue
		}
		sockAddrInet4, ok := clientSocket.(*syscall.SockaddrInet4)
		if !ok {
			log.Print("sockaddrinet4 assertion from sockaddr failed")
			continue
		}
		ts.clientAddresses[clientFileDescriptor] = *sockAddrInet4
		clientFileDescriptorChan <- clientFileDescriptor
		log.Printf("connected with IP: %v and port: %d", sockAddrInet4.Addr, sockAddrInet4.Port)
	}
}

func readRequest(clientFileDescriptor int) (request []byte, err error) {
	clientInputChan := make(chan []byte)

	go func(chan []byte) {
		buffer := make([]byte, KILOBYTE/2)
		for {
			bytesRead, err := syscall.Read(clientFileDescriptor, buffer)
			if err != nil {
				return
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
			log.Println("request read timeout reached")
			return
		}
	}
}

func (ts *tcpServer) closeSocket(fileDescriptor int) (err error) {
	if err = syscall.Close(fileDescriptor); err != nil {
		return err
	}
	log.Printf("socket with IP: %v and port: %d is closed",
		ts.clientAddresses[fileDescriptor].Addr,
		ts.clientAddresses[fileDescriptor].Port)
	if fileDescriptor != ts.serverFileDescriptor {
		delete(ts.clientAddresses, fileDescriptor)
	}
	return nil
}
