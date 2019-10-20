package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"
)

var bufSize = 8 * 1024
var retriesCnt = 5
var dialTimeOut = 2 * time.Second
var (
	l string
	r string

)


func handler(conn net.Conn, r string) {
	var i = 0
	for i < retriesCnt {
		fmt.Println("Dial timeout: ", dialTimeOut)
		fmt.Println("Dial " + strconv.Itoa(i) + " times")
		client, err := net.DialTimeout("tcp", r, dialTimeOut)
		if err != nil {
			fmt.Println("Dial remote failed", err)
			i++
			continue
		}
		fmt.Println("To: Connected to remote ", r)
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
		break
	}

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
		fmt.Println("From: Accepted connection: ", conn.RemoteAddr().String())
		go handler(conn, r)
	}
}
