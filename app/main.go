package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		fmt.Println("Waiting for connection...")
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		fmt.Println("Connection accepted")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		go func() {
			handler(ctx, conn)
		}()
	}
}

func handler(ctx context.Context, conn net.Conn) {
	fmt.Println("Handling request...")
	handleRequest(ctx, conn)
	fmt.Println("Request handled")
	fmt.Println("Handling response...")
	handleResponse(ctx, conn)
	fmt.Println("Response handled")
}

func handleRequest(ctx context.Context, conn net.Conn) {
	request := make([]byte, 1024)
	_, err := conn.Read(request)
	if err != nil {
		fmt.Println("Error reading request: ", err.Error())
		os.Exit(1)
	}

	fmt.Println(string(request))
}

func handleResponse(ctx context.Context, conn net.Conn) {
	response := "HTTP/1.1 200 OK\r\n\r\n"
	conn.Write([]byte(response))
	conn.Close()
}
