package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/natefinch/lumberjack.v2"
	"webnote/config"
	"webnote/db"
	"webnote/server"
)

const applicationLogDir = "/var/log/webnote"

func newRotatingLogger(filename string, maxSize, maxBackups int) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		Compress:   true,
		LocalTime:  true,
	}
}

func initLogging() error {
	if err := os.MkdirAll(applicationLogDir, 0750); err != nil {
		return err
	}
	rotating := newRotatingLogger(filepath.Join(applicationLogDir, "webnote.log"), 30, 7)
	writer := io.MultiWriter(os.Stdout, rotating)
	log.SetOutput(writer)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC)
	gin.DefaultWriter = writer
	gin.DefaultErrorWriter = writer
	return nil
}

func main() {
	if err := initLogging(); err != nil {
		log.Fatalf("initialize file logging: %v", err)
	}
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := config.Init(); err != nil {
		return err
	}
	for _, dir := range []string{config.Conf.DataDir, config.Conf.ImgDir} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	if err := db.InitDB(); err != nil {
		return err
	}
	defer db.Close()
	if !config.Conf.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	if err := engine.SetTrustedProxies([]string{"127.0.0.1", "::1"}); err != nil {
		return err
	}
	engine.Use(gin.LoggerWithWriter(log.Writer()), gin.RecoveryWithWriter(log.Writer()))
	engine.MaxMultipartMemory = 6 << 20
	server.Init(engine)
	srv := &http.Server{Addr: config.Conf.Address + ":" + strconv.Itoa(config.Conf.Port), Handler: http.MaxBytesHandler(engine, 6<<20), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("starting Webnote API on %s; application logs: %s", srv.Addr, filepath.Join(applicationLogDir, "webnote.log"))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	go func() {
		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		done <- err
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		log.Printf("shutting down Webnote API")
		return srv.Shutdown(shutdown)
	}
}
