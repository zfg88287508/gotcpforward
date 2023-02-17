package main

import (
	"context"
	"flag"
	"github.com/gotcpforward/gotcpforward/common"
	"github.com/gotcpforward/gotcpforward/signal"
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
		sugar.Infof("Failed to listen on %v", l, err)
		return
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			sugar.Infof("Failed to accept listener. %v", err)
			return
		}
		sugar.Infof("From: Accepted connection: %v", conn.RemoteAddr().String())
		//go handler(conn, r)
		go handler(conn, r)
	}
}

func handler(conn net.Conn, r string) {
	dialer := &net.Dialer{
		Timeout:   DialTimeout,
		KeepAlive: IdleTimeout,
	}
	client, err := dialer.Dial("tcp", r)
	if err != nil {
		sugar.Infof("Dial remote failed %v", err)

		return
	}
	sugar.Infof("To: Connected to remote %v", r)

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
		conn.SetDeadline(time.Now())
		client.SetDeadline(time.Now())
	}

	timer := signal.CancelAfterInactivity(connCtx, cancelFunc, DefaultProxyIdleTimeout, sugar)

	inConn := common.NewIdleTimeoutConnV3(conn, timer.Update, sugar)
	outConn := common.NewIdleTimeoutConnV3(client, timer.Update, sugar)

	go copySync(inConn, outConn, wg)
	go copySync(outConn, inConn, wg)

	wg.Wait()

	sugar.Infof(" finish copy")

	if conn != nil {
		defer conn.Close()
	}
	if client != nil {
		defer client.Close()
	}
	sugar.Infof(" close connections")
}
