package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
)

var bufSize = 8 * 1024
var (
	l string
	r string
)


func handler(conn net.Conn, r string) {
	client, err := net.Dial("tcp", r)
	if err != nil {
		fmt.Println("Dial remote failed", err)
		return
	}
	fmt.Println("Connected to remote ", r)
	go func() {
		defer client.Close()
		defer conn.Close()
		clientbuf := make([]byte, bufSize)
		io.CopyBuffer(client, conn, clientbuf)
	}()
	go func() {
		defer client.Close()
		defer conn.Close()
		serverbuf := make([]byte, bufSize)
		io.CopyBuffer(conn, client, serverbuf)
	}()
}
func main() {
	flag.StringVar(&l, "l", "", "listen host:port")
	flag.StringVar(&r, "r", "", "remote host:port")
	flag.Parse()
	if len(l) <= 0 {
		flag.PrintDefaults();
		os.Exit(-1)
	}
	if len(r) <= 0 {
		flag.PrintDefaults()
		os.Exit(-1)
	}
	fmt.Println("Listen on:", l)
	fmt.Println("Forward request to:", r)
	listener, err := net.Listen("tcp", l)
	if err != nil {
		fmt.Println("Failed to listen on ", l, err)
		return
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Failed to accept listener. ", err)
			return
		}
		fmt.Println("Accepted connection")
		go handler(conn, r)
	}
}
