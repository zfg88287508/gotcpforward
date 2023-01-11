package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

var (
	l           string
	r           string
	DialTimeout = 3 * time.Second
	IdleTimeout = 20 * time.Second
)

func handler(conn net.Conn, r string) {
	dialer := &net.Dialer{
		Timeout:   DialTimeout,
		KeepAlive: IdleTimeout,
	}
	client, err := dialer.Dial("tcp", r)
	if err != nil {
		fmt.Println("Dial remote failed", err)

		return
	}
	fmt.Println("To: Connected to remote ", r)

	copySync := func(w io.Writer, r io.Reader, wg *sync.WaitGroup) {
		defer wg.Done()
		if _, err := io.Copy(w, r); err != nil && err != io.EOF {
			fmt.Printf("failed to copy  tunnel: %v\n", err)
		}

		fmt.Printf(" finished copying\n")
	}
	wg := &sync.WaitGroup{}
	wg.Add(2)

	go copySync(conn, client, wg)
	go copySync(client, conn, wg)

	wg.Wait()

	fmt.Println(" finish copy")

	if conn != nil {
		defer conn.Close()
	}
	if client != nil {
		defer client.Close()
	}
	fmt.Println(" close connections")
}

func main() {
	flag.StringVar(&l, "l", "", "listen host:port")
	flag.StringVar(&r, "r", "", "remote host:port")
	flag.Parse()
	if len(l) <= 0 {
		flag.PrintDefaults()
		os.Exit(-1)
	}
	if len(r) <= 0 {
		flag.PrintDefaults()
		os.Exit(-1)
	}

	fmt.Println("Listen on:", l)
	fmt.Println("Forward request to:", r)
	listener, err := net.Listen("tcp", l)

	fmt.Println("Dial timeout: ", DialTimeout)
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
		//go handler(conn, r)
		go handler(conn, r)
	}
}
