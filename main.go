package main

import (
	"flag"
	"fmt"
	"github.com/fatih/pool"
	"io"
	"net"
)

var (
	l string
	r string
)

func handler(conn net.Conn, p pool.Pool) {
	client, err := p.Get()
	if err != nil {
		fmt.Println("Dial remote failed", err)
		return
	}
	go func() {
		defer client.Close()
		defer conn.Close()
		io.Copy(client, conn)
	}()
	go func() {
		defer client.Close()
		defer conn.Close()
		io.Copy(conn, client)
	}()
}
func main() {
	flag.StringVar(&l, "l", "", "listen host:port")
	flag.StringVar(&r, "r", "", "remote host:port")
	flag.Parse()
	fmt.Println(l)
	fmt.Println(r)
	listener, err := net.Listen("tcp", l)
	if err != nil {
		fmt.Println("Failed to listen on ", l, err)
		return
	}
	factory := func() (net.Conn, error) { return net.Dial("tcp", r) }
	p, err := pool.NewChannelPool(5, 30, factory)
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Failed to accept listener. ", err)
			return
		}
		fmt.Println("Accepted connection")
		go handler(conn, p)
	}
}
