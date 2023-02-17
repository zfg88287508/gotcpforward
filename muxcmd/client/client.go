package main

import (
	"context"
	"flag"
	"github.com/gotcpforward/gotcpforward/common"
	"github.com/gotcpforward/gotcpforward/signal"
	"github.com/hashicorp/yamux"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

var (
	sugar       *zap.SugaredLogger
	l           string
	r           string
	DialTimeout = 3 * time.Second
	IdleTimeout = 20 * time.Second

	DefaultProxyIdleTimeout = 180 * time.Second
)

func main() {
	atom := zap.NewAtomicLevel()

	// To keep the example deterministic, disable timestamps in the output.
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	logger := zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atom,
	))
	defer logger.Sync()

	atom.SetLevel(zap.DebugLevel)

	sugar = logger.Sugar()

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

	sugar.Infof("Listen on: %v", l)
	sugar.Infof("Forward request to: %v", r)
	listener, err := net.Listen("tcp", l)

	if err != nil {
		sugar.Infof("Failed to listen on %v, %v", l, err)
		return
	}

	dialer := &net.Dialer{
		Timeout:   DialTimeout,
		KeepAlive: IdleTimeout,
	}
	outboundConn, err := dialer.Dial("tcp", r)
	if err != nil {
		sugar.Infof("Dial remote failed %v", err)

		return
	}
	sugar.Infof("To: Connected to remote %v", r)
	session, err := yamux.Client(outboundConn, nil)
	if err != nil {

		sugar.Infof("To: Failed to Connected to remote %v", r)
		return
	}

	// Open a new stream
	for {
		inboundConn, err := listener.Accept()
		if err != nil {
			sugar.Infof("Failed to accept listener. %v", err)
			continue
		}
		sugar.Infof("From: Accepted connection: %v ", inboundConn.RemoteAddr().String())
		//go handler(conn, r)

		sugar.Infof("opening stream")
		stream, err := session.Open()
		if err != nil {
			sugar.Infof("Open session failed, stream conn failed")
			continue
		}

		go handler(inboundConn, stream)
	}
}

func handler(inboundConn net.Conn, outboundConn net.Conn) {

	copySync := func(w io.Writer, r io.Reader, wg *sync.WaitGroup) {
		defer wg.Done()
		if _, err := io.Copy(w, r); err != nil && err != io.EOF {
			sugar.Infof("failed to copy  tunnel: %v\n", err)
		}

		sugar.Infof(" finished copying\n")
	}
	wg := &sync.WaitGroup{}
	wg.Add(2)
	connCtx, cancel := context.WithCancel(context.Background())
	cancelFunc := func() {
		sugar.Infof("链接已经超时，准备关闭链接\n")
		cancel()
	}

	timer := signal.CancelAfterInactivity(connCtx, cancelFunc, DefaultProxyIdleTimeout, sugar)

	idleInboundConn := common.NewIdleTimeoutConnV3(inboundConn, timer.Update, sugar)

	idleOutboundConn := common.NewIdleTimeoutConnV3(outboundConn, timer.Update, sugar)

	go copySync(idleInboundConn, idleOutboundConn, wg)
	go copySync(idleOutboundConn, idleInboundConn, wg)

	wg.Wait()

	sugar.Infof(" finish copy")

	if inboundConn != nil {
		defer inboundConn.Close()
	}
	if outboundConn != nil {
		defer outboundConn.Close()
	}
	sugar.Infof(" close connections")
}
