// Copyright 2020 Alexandre Pires (c.alexandre.pires@gmail.com)

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// 	http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"context"
	"crypto/tls"
	"cybero/definitions"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var serverLogger *log.Logger

// RestAPIServer A simple RestAPI server
type RestAPIServer struct {
	serverEndpoints  definitions.RestAPIEndpoints
	httpServer       *http.Server
	serverConfig     *definitions.RestAPIConfig
	serverConfigFile string
	serverLogfile    *os.File
}

func (rest *RestAPIServer) loadConfig(configFile string) error {

	fileDscr, err := os.Open(configFile)
	if err != nil {
		fmt.Printf("RestAPIServer: Error loading config file %q: %v\n", configFile, err)
		return err
	}
	defer fileDscr.Close()

	decoder := json.NewDecoder(fileDscr)

	err = decoder.Decode(rest.serverConfig)
	if err != nil {
		fmt.Printf("RestAPIServer: Error loading config file %q: %v\n", configFile, err)
		return err
	}

	return nil
}

// Initialize Initialize a Rest server
func (rest *RestAPIServer) Initialize() error {

	var err error

	rest.serverConfig = &definitions.RestAPIConfig{}

	// Try to load configuration from arguments
	flag.StringVar(&rest.serverConfigFile, "config", "", "Service config file")
	flag.StringVar(&rest.serverConfig.LogFile, "logfile", "", "Log file name")
	flag.StringVar(&rest.serverConfig.Socket, "socket", "", "Unix socket file")
	flag.BoolVar(&rest.serverConfig.TLS, "tls", false, "Use TLS encryption")
	flag.StringVar(&rest.serverConfig.CertPEM, "pem", "", "TLS PEM file")
	flag.StringVar(&rest.serverConfig.CertKey, "key", "", "TLS key file")
	flag.StringVar(&rest.serverConfig.Modules.Path, "modules", "", "Modiles location")
	flag.Parse()

	// load configuration file
	if rest.serverConfigFile != "" {
		if err := rest.loadConfig(rest.serverConfigFile); err != nil {
			return err
		}
	}

	// Initialize logging to a file
	if rest.serverLogfile, err = os.OpenFile(rest.serverConfig.LogFile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err != nil {
		log.Println(err)
	}
	serverLogger = log.New(rest.serverLogfile, "", log.LstdFlags)

	// get the socket to listen
	socket := rest.serverConfig.Socket

	if strings.HasPrefix(socket, "unix://") {

		// Unix socket listener
		socketFile := strings.ReplaceAll(socket, "unix://", "")
		if err := rest.listenUnixSocket(socketFile); err != nil {
			return err
		}

	} else if strings.HasPrefix(socket, "tcp://") {
		// Setup API Server on tcp socket if enabled

		addr := strings.ReplaceAll(socket, "tcp://", "")

		if rest.serverConfig.TLS {

			// We have TLS enable, setup a secure socket, otherwise a non encrypted socket
			if err := rest.listenTCPSocketTLS(addr, rest.serverConfig.CertPEM, rest.serverConfig.CertKey); err != nil {
				serverLogger.Printf("RestAPIServer: Failed to bind server on tcp secure socket %q: %v\n", addr, err)
				return err
			}
		} else if err := rest.listenTCPSocket(addr); err != nil {
			serverLogger.Printf("RestAPIServer: Failed to bind server on tcp socket %q: %v\n", addr, err)
			return err
		}

	} else {
		serverLogger.Printf("RestAPIServer: Invalid socket to listen %q: %v\n", socket, err)
		return errors.New("Invalid socket")
	}

	// Create internal endpoints
	auth := Auth{}
	auth.Initialize(serverLogger, rest.serverConfig)
	api := API{}
	api.Initialize(serverLogger, rest.serverConfig)

	// Assign internal endpoints
	rest.APIHandler(AuthEndpoint, auth.HandleRequest)
	rest.APIHandler(APIEndpoint, api.HandleRequest)
	return nil
}

// APIHandler Add a new handler
func (rest *RestAPIServer) APIHandler(url string, handler definitions.RestAPIHandler) {
	if rest.serverEndpoints == nil {
		rest.serverEndpoints = definitions.RestAPIEndpoints{}
	}
	rest.serverEndpoints[url] = handler
}

func (rest *RestAPIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	parts := strings.Split(r.URL.Path[1:], "/")

	if len(parts) != 0 {
		serverLogger.Printf("Processing API request\n")
		if handler, ok := rest.serverEndpoints[parts[0]]; ok {
			if err := handler(w, r); err == nil {
				// Request was handled by registered endpoint
				return
			}
		}
	}

	// If we arrive here means something went wrong
	encoder := json.NewEncoder(w)
	code, msg := -1, map[string]interface{}{"Error": fmt.Sprintf("Error, processing request\n")}

	encoder.Encode(definitions.RestAPIResponse{
		"Status":   code,
		"Response": msg,
	})
}

// listenUnixSocket start API listener on a unix socket
func (rest *RestAPIServer) listenUnixSocket(socket string) error {

	addr, err := net.ResolveUnixAddr("unix", socket)

	if err != nil {
		serverLogger.Printf("Failed open socket %q: %v\n", socket, err)
		return err
	}

	listener, err := net.ListenUnix("unix", addr)

	if err != nil {
		serverLogger.Printf("Failed to listen in socket: %v\n", err)
		return err
	}

	server := http.Server{
		Handler:      rest,
		ErrorLog:     serverLogger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	rest.httpServer = &server

	go server.Serve(listener)
	return nil
}

// ListenTCPSocket Listen API in a TCP socket
func (rest *RestAPIServer) listenTCPSocket(address string) error {

	listener, err := net.Listen("tcp", address)

	if err != nil {
		serverLogger.Printf("Failed to listen in address %q: %v\n", address, err)
		return err
	}

	server := http.Server{
		Handler:      rest,
		ErrorLog:     serverLogger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	rest.httpServer = &server

	go server.Serve(listener)
	return nil
}

// listenTCPSocketTLS Listen API in a TCP socket
func (rest *RestAPIServer) listenTCPSocketTLS(address string, pemFile string, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(pemFile, keyFile)

	if err != nil {
		serverLogger.Fatalf("Failed to load keys: %v\n", err)
		return err
	}

	config := tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}

	listener, err := tls.Listen("tcp", address, &config)

	if err != nil {
		serverLogger.Printf("Failed to listen in address %q: %v\n", address, err)
		return err
	}

	server := http.Server{
		Handler:      rest,
		ErrorLog:     serverLogger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	rest.httpServer = &server

	go server.Serve(listener)
	return nil
}

// Shutdown shutdown server
func (rest *RestAPIServer) Shutdown() {
	// Do a gracefull shutdown of the server
	serverLogger.Println("Server is shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rest.httpServer.SetKeepAlivesEnabled(false)

	if err := rest.httpServer.Shutdown(ctx); err != nil {
		serverLogger.Fatalf("Could not gracefully shutdown the TCP server: %v\n", err)
	}

	// get the socket to listen
	socket := rest.serverConfig.Socket

	if strings.HasPrefix(socket, "unix://") {
		socketFile := strings.ReplaceAll(socket, "unix://", "")
		defer os.Remove(socketFile)
	}
	defer rest.serverLogfile.Close()
}
