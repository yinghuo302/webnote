package main

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"webnote/config"
	"webnote/db"
	"webnote/server"
)

func main() {
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
	engine.Use(gin.Logger(), gin.Recovery())
	engine.MaxMultipartMemory = 6 << 20
	server.Init(engine)
	srv := &http.Server{Addr: config.Conf.Address + ":" + strconv.Itoa(config.Conf.Port), Handler: http.MaxBytesHandler(engine, 6<<20), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
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
		return srv.Shutdown(shutdown)
	}
}
