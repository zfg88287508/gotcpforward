package main

import (
	"context"
	"flag"
	"github.com/gotcpforward/gotcpforward/common"
	"github.com/gotcpforward/gotcpforward/signal"
	"github.com/gotcpforward/gotcpforward/task"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"net"
	"os"
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

	cm := "main"
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

	sugar.Infof(cm+" Listen on: %v", l)
	sugar.Infof(cm+" Forward request to: %v", r)
	listener, err := net.Listen("tcp", l)

	if err != nil {
		sugar.Infof(cm+" Failed to listen on %v", l, err)
		return
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			sugar.Infof(cm+" Failed to accept listener. %v", err)
			return
		}
		sugar.Infof(cm+"From: Accepted connection: %v", conn.RemoteAddr().String())
		//go handler(conn, r)
		go handler(conn, r)
	}
}

func handler(conn net.Conn, r string) {
	cm := "handler"
	dialer := &net.Dialer{
		Timeout:   DialTimeout,
		KeepAlive: IdleTimeout,
	}
	client, err := dialer.Dial("tcp", r)
	if err != nil {
		sugar.Infof(cm+" Dial remote failed %v", err)

		return
	}
	sugar.Infof(cm+" To: Connected to remote %v", r)

	copySync := func(w io.Writer, r io.Reader) error {

		if _, err := io.Copy(w, r); err != nil && err != io.EOF {
			sugar.Infof(cm+" failed to copy  tunnel: %v\n", err)
			return err
		}

		sugar.Infof(cm + " finished copying\n")
		return nil
	}

	connCtx, cancel := context.WithCancel(context.Background())

	cancelFunc := func() {
		sugar.Infof(cm + " 链接已经超时，准备关闭链接\n")
		cancel()
		conn.SetDeadline(time.Now())
		client.SetDeadline(time.Now())
	}

	timer := signal.CancelAfterInactivity(connCtx, cancelFunc, DefaultProxyIdleTimeout, sugar)

	inConn := common.NewIdleTimeoutConnV3(conn, timer.Update, sugar)
	outConn := common.NewIdleTimeoutConnV3(client, timer.Update, sugar)

	_ = task.Run(context.Background(), func() error {
		return copySync(inConn, outConn)
	}, func() error {
		return copySync(outConn, inConn)
	})

	sugar.Infof(cm + " finish copy")

	if conn != nil {
		defer conn.Close()
	}
	if client != nil {
		defer client.Close()
	}
	sugar.Infof(cm + " close connections")
}
