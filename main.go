package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/infobsmi/bsmi-go/idle_conn"
	"github.com/panjf2000/ants/v2"
)

var (
	l           string
	r           string
	DialTimeout = 2 * time.Second
	IdleTimeout = 2 * time.Minute
)

func handler(conn net.Conn, r string) {
	client, err := net.DialTimeout("tcp", r, DialTimeout)
	if err != nil {
		fmt.Println("Dial remote failed", err)

		return
	}
	fmt.Println("To: Connected to remote ", r)

	idleCbw := &idle_conn.IdleConn[net.Conn]{
		Conn: conn,
	}
	idleCbr := &idle_conn.IdleConn[net.Conn]{
		Conn: client,
	}

	ants.Submit(func() { copySync(idleCbw, idleCbr) })
	ants.Submit(func() { copySync(idleCbr, idleCbw) })

}

func copySync(w io.Writer, r io.Reader) {
	if _, err := io.Copy(w, r); err != nil && err != io.EOF {
		fmt.Printf(" failed to copy : %v\n", err)
	}

	fmt.Printf(" finished copying\n")
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
		ants.Submit(func() { handler(conn, r) })
	}
}
