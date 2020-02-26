package main

import (
	"cybero/core"
	"fmt"
	"os"
	"os/signal"
)

var (
	restAPIServer *core.RestAPIServer
)

func gracefullShutdown(quit <-chan os.Signal, done chan<- bool) {
	// Do a gracefull shutdown of the server
	<-quit
	restAPIServer.Shutdown()
	close(done)
}

func startServer() error {
	// Setup os signal catch
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	// Setup API Server on unix socket
	restAPIServer = &core.RestAPIServer{}
	if err := restAPIServer.Initialize(); err != nil {
		return err
	}

	go gracefullShutdown(quit, done)
	<-done
	return nil
}

func main() {
	fmt.Println("Cybero - modular API server, developed by Alexandre Pires (c.alexandre.pires@gmail.com) (2020)")
	if err := startServer(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
