package main

import (
	"context"
	"flag"
	"github.com/gotcpforward/gotcpforward/common"
	"github.com/gotcpforward/gotcpforward/signal"
	"github.com/gotcpforward/gotcpforward/task"
	"github.com/hashicorp/yamux"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"log"
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

	DefaultProxyIdleTimeout = 30 * time.Second
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

	undoLog := zap.RedirectStdLog(logger)
	defer undoLog()

	yamuxConfig := yamux.DefaultConfig()
	//设置日志
	yamuxConfig.LogOutput = log.Writer()

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

	var pSession *yamux.Session

	// Open a new stream
	for {
		inboundConn, err := listener.Accept()

		if err != nil {
			sugar.Infof("Failed to accept listener. %v", err)
			continue
		}

		if pSession == nil || pSession.IsClosed() {

			pSession, err = yamux.Client(getOneConn(r), yamuxConfig)
			if err != nil {

				sugar.Infof("To: Failed to Connected to remote %v, will try again", r)
				continue
			}

		}

		sugar.Infof("From: Accepted connection: %v ", inboundConn.RemoteAddr().String())
		//go handler(conn, r)

		sugar.Infof("opening stream")
		stream, err := pSession.Open()
		if err != nil {
			pSession.Close()

			sugar.Infof("Open session failed, stream conn failed")
			continue
		}

		go handler(inboundConn, stream)
	}
}

func getOneConn(s string) io.ReadWriteCloser {

	dialer := &net.Dialer{
		Timeout:   DialTimeout,
		KeepAlive: IdleTimeout,
	}

	outboundConn, err := dialer.Dial("tcp", r)
	if err != nil {
		sugar.Infof("Dial remote failed %v", err)

		return nil
	}
	sugar.Infof("To: Connected to remote %v", r)
	return outboundConn

}

func handler(inboundConn net.Conn, outboundConn net.Conn) {

	copySync := func(w io.Writer, r io.Reader) error {
		if _, err := io.Copy(w, r); err != nil && err != io.EOF {
			sugar.Infof("failed to copy  tunnel: %v\n", err)
			return err
		}

		sugar.Infof(" finished copying\n")
		return nil
	}

	connCtx, cancel := context.WithCancel(context.Background())
	cancelFunc := func() {
		sugar.Infof("链接已经超时，准备关闭链接\n")
		cancel()
		dlTime := time.Now().Add(IdleTimeout)
		if inboundConn != nil {
			inboundConn.SetDeadline(dlTime)
		}
		if outboundConn != nil {
			outboundConn.SetDeadline(dlTime)
		}
	}

	timer := signal.CancelAfterInactivity(connCtx, cancelFunc, DefaultProxyIdleTimeout, sugar)

	idleInboundConn := common.NewIdleTimeoutConnV3(inboundConn, timer.Update, sugar)

	idleOutboundConn := common.NewIdleTimeoutConnV3(outboundConn, timer.Update, sugar)

	_ = task.Run(
		context.Background(),
		func() error {
			return copySync(idleInboundConn, idleOutboundConn)
		},
		func() error {
			return copySync(idleOutboundConn, idleInboundConn)
		},
	)

	sugar.Infof(" finish copy")

	dlTime := time.Now().Add(IdleTimeout)
	if inboundConn != nil {
		inboundConn.SetDeadline(dlTime)
	}
	if outboundConn != nil {
		outboundConn.SetDeadline(dlTime)
	}
	sugar.Infof(" close connections")
}
