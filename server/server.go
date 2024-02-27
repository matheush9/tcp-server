package server

import (
	"errors"
	"fmt"
	"log"
	"syscall"
	"time"

	"github.com/matheush9/tcp-server/iputils"
)

type ServerConfig struct {
	ConnectionTimeout    float64
	IPV4Address          string
	Port                 int
	serverFileDescriptor int
	clientAddresses      map[int]syscall.SockaddrInet4
}

type ClientRequest struct {
	Request       string
	ClientAddress syscall.SockaddrInet4
	clientFd      int
}

const kilobyte = 1024 //bytes

func (sc *ServerConfig) HandleConnections(clientReqChan chan ClientRequest) (err error) {
	if sc.serverFileDescriptor == 0 {
		return errors.New("you have to configure the server before handling connections")
	}
	clientFileDescriptorChan := make(chan int)

	go sc.listenClientConnections(sc.serverFileDescriptor, clientFileDescriptorChan)
	for clientFileDescriptor := range clientFileDescriptorChan {
		requestChan := make(chan string)

		go sc.handleRequest(clientFileDescriptor, requestChan)
		for request := range requestChan {
			clientReqChan <- ClientRequest{
				Request:       request,
				ClientAddress: sc.clientAddresses[clientFileDescriptor],
				clientFd:      clientFileDescriptor}
		}
	}
	return nil
}

func (sc ServerConfig) handleRequest(clientFileDescriptor int, requestChan chan string) {
	request, err := sc.readRequest(clientFileDescriptor)
	if err != nil {
		log.Printf("error while trying to read the request from IP: %v and port: %d: %v",
			sc.clientAddresses[clientFileDescriptor].Addr,
			sc.clientAddresses[clientFileDescriptor].Port,
			err)
		return
	}
	if len(request) > 0 {
		log.Print("request received: ", string(request))
	}
	requestChan <- request
	close(requestChan)
}

func (sc *ServerConfig) SendResponse(clientReq ClientRequest, response string) error {
	if clientReq.clientFd == 0 {
		return errors.New("invalid file descriptor")
	}
	defer func() {
		err := sc.closeSocket(clientReq.clientFd)
		if err != nil {
			log.Printf("error while trying to close the client socket with IP: %v and port: %d: %v",
				sc.clientAddresses[clientReq.clientFd].Addr,
				sc.clientAddresses[clientReq.clientFd].Port,
				err)
		}
	}()
	_, err := syscall.Write(clientReq.clientFd, []byte(response))
	if err != nil {
		return err
	}
	return nil
}

func (sc ServerConfig) CloseServerSocket() (err error) {
	err = sc.closeSocket(sc.serverFileDescriptor)
	if err != nil {
		return err
	}
	return nil
}

func (sc *ServerConfig) SetupServer() (err error) {
	sc.clientAddresses = make(map[int]syscall.SockaddrInet4)
	IPAddress, err := iputils.IPStringToIPBytes(sc.IPV4Address)
	if err != nil {
		return fmt.Errorf("the IP provided is not valid: %v", err)
	}
	sc.serverFileDescriptor, err = syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
	if err != nil {
		return fmt.Errorf("error while creating server socket: %v", err)
	}
	defer func() {
		if err != nil {
			err = sc.CloseServerSocket()
			if err != nil {
				log.Printf("error while closing the server socket: %v", err)
			}
		}
	}()
	for counter := 0; counter < 60; counter++ {
		if err = syscall.Bind(sc.serverFileDescriptor, &syscall.SockaddrInet4{Addr: IPAddress, Port: sc.Port}); err == syscall.EADDRINUSE {
			log.Println("address in use, trying to bind again in 5 secs until 5 minutes")
			time.Sleep(5 * time.Second)
		} else if err != nil {
			return fmt.Errorf("error while binding the server socket: %v", err)
		} else {
			break
		}
	}
	err = syscall.Listen(sc.serverFileDescriptor, 1)
	if err != nil {
		return fmt.Errorf("error while preparing server socket to accept connections: %v", err)
	}

	return nil
}

func (sc *ServerConfig) listenClientConnections(serverFileDescriptor int, clientFileDescriptorChan chan int) {
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
		sc.clientAddresses[clientFileDescriptor] = *sockAddrInet4
		clientFileDescriptorChan <- clientFileDescriptor
		log.Printf("connected with IP: %v and port: %d", sockAddrInet4.Addr, sockAddrInet4.Port)
	}
}

func (sc ServerConfig) readRequest(clientFileDescriptor int) (request string, err error) {
	clientInputChan := make(chan []byte)

	go func(chan []byte) {
		buffer := make([]byte, kilobyte/2)
		var requestBytes []byte
		for {
			bytesRead, err := syscall.Read(clientFileDescriptor, buffer)
			if err != nil {
				return
			}
			requestBytes := append(requestBytes, buffer[:bytesRead]...)
			clientInputChan <- requestBytes
			if len(buffer) != bytesRead || bytesRead == 0 {
				close(clientInputChan)
				request = string(requestBytes)
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
		case <-time.After(time.Second * time.Duration(sc.ConnectionTimeout)):
			log.Println("request read timeout reached")
			return
		}
	}
}

func (sc *ServerConfig) closeSocket(fileDescriptor int) (err error) {
	if err = syscall.Close(fileDescriptor); err != nil {
		return err
	}
	log.Printf("socket with IP: %v and port: %d is closed",
		sc.clientAddresses[fileDescriptor].Addr,
		sc.clientAddresses[fileDescriptor].Port)
	if fileDescriptor != sc.serverFileDescriptor {
		delete(sc.clientAddresses, fileDescriptor)
	}
	return nil
}
