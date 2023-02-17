package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/gotcpforward/gotcpforward/common"
	"github.com/gotcpforward/gotcpforward/signal"
	"github.com/hashicorp/yamux"
	"io"
	"log"
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

	DefaultProxyIdleTimeout = 180 * time.Second
)

func handler(inboundConn net.Conn, outboundConn net.Conn) {

	copySync := func(w io.Writer, r io.Reader, wg *sync.WaitGroup) {
		defer wg.Done()
		if _, err := io.Copy(w, r); err != nil && err != io.EOF {
			fmt.Printf("failed to copy  tunnel: %v\n", err)
		}

		fmt.Printf(" finished copying\n")
	}
	wg := &sync.WaitGroup{}
	wg.Add(2)
	connCtx, cancel := context.WithCancel(context.Background())
	cancelFunc := func() {
		fmt.Printf("链接已经超时，准备关闭链接\n")
		cancel()
	}

	timer := signal.CancelAfterInactivity(connCtx, cancelFunc, DefaultProxyIdleTimeout)

	idleInboundConn := common.NewIdleTimeoutConnV3(inboundConn, timer.Update)

	idleOutboundConn := common.NewIdleTimeoutConnV3(outboundConn, timer.Update)

	go copySync(idleInboundConn, idleOutboundConn, wg)
	go copySync(idleOutboundConn, idleInboundConn, wg)

	wg.Wait()

	fmt.Println(" finish copy")

	if inboundConn != nil {
		defer inboundConn.Close()
	}
	if outboundConn != nil {
		defer outboundConn.Close()
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

	dialer := &net.Dialer{
		Timeout:   DialTimeout,
		KeepAlive: IdleTimeout,
	}
	outboundConn, err := dialer.Dial("tcp", r)
	if err != nil {
		fmt.Println("Dial remote failed", err)

		return
	}
	fmt.Println("To: Connected to remote ", r)
	session, err := yamux.Client(outboundConn, nil)
	if err != nil {

		fmt.Println("To: Failed to Connected to remote ", r)
		return
	}

	// Open a new stream
	for {
		inboundConn, err := listener.Accept()
		if err != nil {
			fmt.Println("Failed to accept listener. ", err)
			return
		}
		fmt.Println("From: Accepted connection: ", inboundConn.RemoteAddr().String())
		//go handler(conn, r)

		log.Println("opening stream")
		stream, err := session.Open()
		if err != nil {
			fmt.Println("Open session failed, stream conn failed")
			return
		}

		go handler(inboundConn, stream)
	}
}
