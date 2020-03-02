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
	"cybero/types"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// RestAPIServer A simple RestAPI server
type RestAPIServer struct {
	serverEndpoints types.RestAPIEndpoints
	httpServer      *http.Server
}

// Initialize Initialize a Rest server
func (rest *RestAPIServer) Initialize() error {

	var err error

	cfg := GetConfigManager().GetConfig()
	logger := GetLogManager().GetLogger()

	// get the socket to listen
	socket := cfg.Socket

	if strings.HasPrefix(socket, "unix://") {

		// Unix socket listener
		socketFile := strings.ReplaceAll(socket, "unix://", "")
		if err := rest.listenUnixSocket(socketFile); err != nil {
			return err
		}

	} else if strings.HasPrefix(socket, "tcp://") {
		// Setup API Server on tcp socket if enabled

		addr := strings.ReplaceAll(socket, "tcp://", "")

		if cfg.TLS {

			// We have TLS enable, setup a secure socket, otherwise a non encrypted socket
			if err := rest.listenTCPSocketTLS(addr, cfg.CertPEM, cfg.CertKey); err != nil {
				logger.Printf("RestAPIServer: Failed to bind server on tcp secure socket %q: %v\n", addr, err)
				return err
			}
		} else if err := rest.listenTCPSocket(addr); err != nil {
			logger.Printf("RestAPIServer: Failed to bind server on tcp socket %q: %v\n", addr, err)
			return err
		}

	} else {
		logger.Printf("RestAPIServer: Invalid socket to listen %q: %v\n", socket, err)
		return err
	}

	// Assign internal endpoints
	rest.APIHandler(AuthEndpoint, GetAuthManager().HandleRequest)
	rest.APIHandler(APIEndpoint, GetAPIManager().HandleRequest)
	return nil
}

// APIHandler Add a new handler
func (rest *RestAPIServer) APIHandler(url string, handler types.RestAPIHandler) {
	if rest.serverEndpoints == nil {
		rest.serverEndpoints = types.RestAPIEndpoints{}
	}
	rest.serverEndpoints[url] = handler
}

func (rest *RestAPIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	logger := GetLogManager().GetLogger()

	w.Header().Set("Content-Type", "application/json")

	parts := strings.Split(r.URL.Path[1:], "/")

	if len(parts) != 0 {
		logger.Printf("Processing API request\n")
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

	encoder.Encode(types.RestAPIResponse{
		"Status":   code,
		"Response": msg,
	})
}

// listenUnixSocket start API listener on a unix socket
func (rest *RestAPIServer) listenUnixSocket(socket string) error {

	logger := GetLogManager().GetLogger()
	addr, err := net.ResolveUnixAddr("unix", socket)

	if err != nil {
		logger.Printf("Failed open socket %q: %v\n", socket, err)
		return err
	}

	listener, err := net.ListenUnix("unix", addr)

	if err != nil {
		logger.Printf("Failed to listen in socket: %v\n", err)
		return err
	}

	server := http.Server{
		Handler:      rest,
		ErrorLog:     logger,
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

	logger := GetLogManager().GetLogger()
	listener, err := net.Listen("tcp", address)

	if err != nil {
		logger.Printf("Failed to listen in address %q: %v\n", address, err)
		return err
	}

	server := http.Server{
		Handler:      rest,
		ErrorLog:     logger,
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

	logger := GetLogManager().GetLogger()
	cert, err := tls.LoadX509KeyPair(pemFile, keyFile)

	if err != nil {
		logger.Fatalf("Failed to load keys: %v\n", err)
		return err
	}

	config := tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}

	listener, err := tls.Listen("tcp", address, &config)

	if err != nil {
		logger.Printf("Failed to listen in address %q: %v\n", address, err)
		return err
	}

	server := http.Server{
		Handler:      rest,
		ErrorLog:     logger,
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

	logger := GetLogManager().GetLogger()
	cfg := GetConfigManager().GetConfig()

	// Do a gracefull shutdown of the server
	logger.Println("Server is shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rest.httpServer.SetKeepAlivesEnabled(false)

	if err := rest.httpServer.Shutdown(ctx); err != nil {
		logger.Fatalf("Could not gracefully shutdown the TCP server: %v\n", err)
	}

	// get the socket to listen
	socket := cfg.Socket

	if strings.HasPrefix(socket, "unix://") {
		socketFile := strings.ReplaceAll(socket, "unix://", "")
		defer os.Remove(socketFile)
	}
}
