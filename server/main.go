package main

import (
	"context"
	"cybero/core"
	"cybero/modules"
	"flag"
	"log"
	"os"
	"os/signal"
	"time"
)

// ServiceLogger the global logger
var ServiceLogger *log.Logger

var (
	cyberoConfigFile string
	cyberoScktFile   string
	cyberoLogfile    string
	cyberoTCPAddress string
	cyberoPemFile    string
	cyberoKeyFile    string
	cyberoUseTLS     bool
	cyberoUseTCP     bool
	unixHTTPServer   *core.RestAPIServer
	tcpHTTPServer    *core.RestAPIServer
)

func gracefullShutdown(quit <-chan os.Signal, done chan<- bool) {
	<-quit
	ServiceLogger.Println("Server is shutting down...")

	if unixHTTPServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		unixHTTPServer.HTTPServer.SetKeepAlivesEnabled(false)
		if err := unixHTTPServer.HTTPServer.Shutdown(ctx); err != nil {
			ServiceLogger.Fatalf("Could not gracefully shutdown the unix socker server: %v\n", err)
		}
	}

	if tcpHTTPServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		tcpHTTPServer.HTTPServer.SetKeepAlivesEnabled(false)
		if err := tcpHTTPServer.HTTPServer.Shutdown(ctx); err != nil {
			ServiceLogger.Fatalf("Could not gracefully shutdown the TCP server: %v\n", err)
		}
	}

	close(done)
}

func runServer() {

	logfile, err := os.OpenFile(cyberoLogfile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer logfile.Close()

	ServiceLogger = log.New(logfile, "", log.LstdFlags)
	modules.ModulesLogger = ServiceLogger
	core.ServerLogger = ServiceLogger

	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, os.Interrupt)

	unixHTTPServer := core.RestAPIServer{}
	err = unixHTTPServer.ListenUnixSocket(cyberoScktFile)
	unixHTTPServer.APIHandle("module", modules.ModuleHandle)

	if err != nil {
		ServiceLogger.Printf("Failed to start server on unix socket %q: %v\n", cyberoScktFile, err)
	}

	if cyberoUseTCP {
		tcpHTTPServer := core.RestAPIServer{}
		tcpHTTPServer.APIHandle("module", modules.ModuleHandle)

		if cyberoUseTLS {
			err = tcpHTTPServer.ListenTCPSocketTLS(cyberoTCPAddress, cyberoPemFile, cyberoKeyFile)
		} else {
			err = tcpHTTPServer.ListenTCPSocket(cyberoTCPAddress)
		}

		if err != nil {
			ServiceLogger.Printf("Failed to API server address %q: %v\n", ":8888", err)
		}
	}

	go gracefullShutdown(quit, done)
	<-done
	ServiceLogger.Println("Server gracefull shutdown")
	os.Remove(cyberoScktFile)
}

func main() {
	flag.StringVar(&cyberoLogfile, "logfile", "/var/log/cybero-server.log", "Log file name")
	flag.StringVar(&cyberoConfigFile, "config", "/etc/cybero/daemon.json", "Service config file")
	flag.StringVar(&cyberoScktFile, "unix", "/var/run/cybero.socket", "Unix socket file")
	flag.StringVar(&cyberoTCPAddress, "ipaddr", ":8888", "TCP bind address")
	flag.BoolVar(&cyberoUseTLS, "tls", false, "Use TLS encryption")
	flag.BoolVar(&cyberoUseTCP, "tcp", true, "Enable TCP connection")
	flag.StringVar(&cyberoPemFile, "pem", "/etc/cybero/cert.pem", "TLS PEM file")
	flag.StringVar(&cyberoKeyFile, "key", "/etc/cybero/cert.key", "TLS key file")
	flag.StringVar(&modules.ModulesLocation, "modules", "/usr/lib/cybero", "Modiles location")
	flag.Parse()
	runServer()
	os.Exit(0)
}
