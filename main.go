package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"service/core"
)

// ServiceLogger the global logger
var ServiceLogger *log.Logger

var (
	servicePidFile  string
	serviceScktFile string
	serviceLogfile  string
)

func start() {

	logfile, err := os.OpenFile(serviceLogfile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer logfile.Close()

	ServiceLogger = log.New(logfile, "", log.LstdFlags)
	core.ModulesLogger = ServiceLogger

	addr, err := net.ResolveUnixAddr("unix", serviceScktFile)

	if err != nil {
		ServiceLogger.Printf("Failed open socket %q: %v\n", serviceScktFile, err)
		os.Exit(1)
	}

	listener, err := net.ListenUnix("unix", addr)

	if err != nil {
		ServiceLogger.Printf("Failed to listen in socket: %v\n", err)
		os.Exit(1)
	}

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel)

	go func() {
		for {
			conn, _ := listener.AcceptUnix()
			go handle(conn)
		}
	}()

	<-signalChannel
	os.Remove(serviceScktFile)
}

func handle(conn net.Conn) {

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	var request core.ServiceRequest

	if err := decoder.Decode(&request); err != nil {
		ServiceLogger.Printf("Failed decode message: %v\n", err)
		conn.Close()
		return
	}

	code, msg := core.ModuleHandle(&request)

	encoder.Encode(core.ServiceResponse{
		Status:   code,
		Response: msg,
	})

	conn.Close()
}

func main() {
	flag.StringVar(&serviceLogfile, "logfile", "/var/log/service.log", "Log file name")
	flag.StringVar(&servicePidFile, "pidfile", "/var/run/service.pid", "Pid file name")
	flag.StringVar(&serviceScktFile, "socketfile", "/var/run/service.socket", "Pid file name")
	flag.StringVar(&core.ModulesLocation, "modules", "/usr/lib/service", "Modiles location")
	flag.Parse()
	start()
}
