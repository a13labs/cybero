package main

import (
	"context"
	"cybero/core"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"
)

// defaultLogger the global logger
var defaultLogger *log.Logger

var (
	cyberoConfigFile string
	cyberoScktFile   string
	cyberoLogfile    string
	cyberoTCPAddress string
	cyberoPemFile    string
	cyberoKeyFile    string
	cyberoModules    string
	cyberoUseTLS     bool
	cyberoUseTCP     bool
	unixHTTPServer   *core.RestAPIServer
	tcpHTTPServer    *core.RestAPIServer
)

func gracefullShutdown(quit <-chan os.Signal, done chan<- bool) {
	// Do a gracefull shutdown of the server
	<-quit
	defaultLogger.Println("Server is shutting down...")

	if unixHTTPServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		unixHTTPServer.HTTPServer.SetKeepAlivesEnabled(false)
		if err := unixHTTPServer.HTTPServer.Shutdown(ctx); err != nil {
			defaultLogger.Fatalf("Could not gracefully shutdown the unix socker server: %v\n", err)
		}
	}

	if tcpHTTPServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		tcpHTTPServer.HTTPServer.SetKeepAlivesEnabled(false)
		if err := tcpHTTPServer.HTTPServer.Shutdown(ctx); err != nil {
			defaultLogger.Fatalf("Could not gracefully shutdown the TCP server: %v\n", err)
		}
	}

	close(done)
}

func runServer() {

	// Initialize logging to a file
	logfile, err := os.OpenFile(cyberoLogfile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer logfile.Close()
	defaultLogger = log.New(logfile, "", log.LstdFlags)

	// Setup os signal catch
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	// Setup API Server on unix socket
	unixHTTPServer := core.RestAPIServer{}
	if unixHTTPServer.Init(defaultLogger, cyberoConfigFile, cyberoModules) == nil {
		if unixHTTPServer.ListenUnixSocket(cyberoScktFile) != nil {
			defaultLogger.Printf("Failed to bind server on unix socket %q: %v\n", cyberoScktFile, err)
		}
	} else {
		defaultLogger.Printf("Failed to initialize server on unix socket %q: %v\n", cyberoScktFile, err)
	}

	// Setup API Server on tcp socket if enabled
	if cyberoUseTCP {
		tcpHTTPServer := core.RestAPIServer{}
		if tcpHTTPServer.Init(defaultLogger, cyberoConfigFile, cyberoModules) == nil {
			if cyberoUseTLS {
				// We have TLS enable, setup a secure socket, otherwise a non encrypted socket
				if tcpHTTPServer.ListenTCPSocketTLS(cyberoTCPAddress, cyberoPemFile, cyberoKeyFile) != nil {
					defaultLogger.Printf("Failed to bind server on tcp secure socket %q: %v\n", cyberoScktFile, err)
				}
			} else if tcpHTTPServer.ListenTCPSocket(cyberoTCPAddress) != nil {
				defaultLogger.Printf("Failed to bind server on tcp socket %q: %v\n", cyberoScktFile, err)
			}
		} else {
			defaultLogger.Printf("Failed to initialize server on unix socket %q: %v\n", cyberoScktFile, err)
		}
	}

	go gracefullShutdown(quit, done)
	<-done
	defaultLogger.Println("Server gracefull shutdown")
	os.Remove(cyberoScktFile)
}

func main() {
	fmt.Println("Cybero - modular API server, developed by Alexandre Pires (c.alexandre.pires@gmail.com) (2020)")
	flag.StringVar(&cyberoLogfile, "logfile", "/var/log/cybero-server.log", "Log file name")
	flag.StringVar(&cyberoConfigFile, "config", "/etc/cybero/daemon.json", "Service config file")
	flag.StringVar(&cyberoScktFile, "unix", "/var/run/cybero.socket", "Unix socket file")
	flag.StringVar(&cyberoTCPAddress, "ipaddr", ":8888", "TCP bind address")
	flag.BoolVar(&cyberoUseTLS, "tls", false, "Use TLS encryption")
	flag.BoolVar(&cyberoUseTCP, "tcp", true, "Enable TCP connection")
	flag.StringVar(&cyberoPemFile, "pem", "/etc/cybero/cert.pem", "TLS PEM file")
	flag.StringVar(&cyberoKeyFile, "key", "/etc/cybero/cert.key", "TLS key file")
	flag.StringVar(&cyberoModules, "modules", "/usr/lib/cybero", "Modiles location")
	flag.Parse()
	runServer()
	os.Exit(0)
}
