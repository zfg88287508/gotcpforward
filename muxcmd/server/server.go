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

	if err != nil {
		fmt.Println("Failed to listen on ", l, err)
		return
	}

	for {
		rawConn, err := listener.Accept()
		if err != nil {
			fmt.Println("rawConn failed")
			continue

		}

		// Setup server side of yamux
		log.Println("creating server session")
		session, err := yamux.Server(rawConn, nil)
		if err != nil {
			fmt.Println(" failed to create yamux serer")
			continue
		}

		dialer := &net.Dialer{
			Timeout:   DialTimeout,
			KeepAlive: IdleTimeout,
		}

		// Open a new stream
		for {

			// Accept a stream
			log.Println("accepting stream")
			inboundConn, err := session.Accept()
			if err != nil {
				fmt.Println("Failed to accept session of stream")
				break
			}

			outboundConn, err := dialer.Dial("tcp", r)
			if err != nil {
				fmt.Println("Dial remote failed", err)

				break
			}
			fmt.Println("To: Connected to remote ", r)

			go handler(inboundConn, outboundConn)
		}
	}
}
