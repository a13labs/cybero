package core

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// ServerLogger logger instance
var ServerLogger *log.Logger

// RestAPIServer A simple RestAPI server
type RestAPIServer struct {
	Endpoints  RestAPIEndpoints
	HTTPServer *http.Server
}

// APIHandle Add a new handler
func (s *RestAPIServer) APIHandle(url string, handler RestAPIHandler) {
	if s.Endpoints == nil {
		s.Endpoints = RestAPIEndpoints{}
	}
	s.Endpoints[url] = handler
}

func (s *RestAPIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	parts := strings.Split(r.URL.Path[1:], "/")

	if len(parts) > 0 {

		if parts[0] == "api" {

			ServerLogger.Printf("Processing API request\n")
			if handler, ok := s.Endpoints[parts[1]]; ok {
				if err := handler(w, r); err == nil {
					// Request was handled by registered endpoint
					return
				}
			}
		}
	}

	// If we arrive here means something went wrong
	encoder := json.NewEncoder(w)
	code, msg := -1, map[string]interface{}{"Error": fmt.Sprintf("Error, processing request\n")}

	encoder.Encode(RestAPIResponse{
		"Status":   code,
		"Response": msg,
	})
}

// ListenUnixSocket start API listener on a unix socket
func (s *RestAPIServer) ListenUnixSocket(socket string) error {

	addr, err := net.ResolveUnixAddr("unix", socket)

	if err != nil {
		ServerLogger.Printf("Failed open socket %q: %v\n", socket, err)
		return err
	}

	listener, err := net.ListenUnix("unix", addr)

	if err != nil {
		ServerLogger.Printf("Failed to listen in socket: %v\n", err)
		return err
	}

	server := http.Server{
		Handler:      s,
		ErrorLog:     ServerLogger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	s.HTTPServer = &server

	go server.Serve(listener)
	return nil
}

// ListenTCPSocket Listen API in a TCP socket
func (s *RestAPIServer) ListenTCPSocket(address string) error {

	listener, err := net.Listen("tcp", address)

	if err != nil {
		ServerLogger.Printf("Failed to listen in address %q: %v\n", address, err)
		return err
	}

	server := http.Server{
		Handler:      s,
		ErrorLog:     ServerLogger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	s.HTTPServer = &server

	go server.Serve(listener)
	return nil
}

// ListenTCPSocketTLS Listen API in a TCP socket
func (s *RestAPIServer) ListenTCPSocketTLS(address string, pemFile string, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(pemFile, keyFile)

	if err != nil {
		ServerLogger.Fatalf("Failed to load keys: %v\n", err)
		return err
	}

	config := tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}

	listener, err := tls.Listen("tcp", address, &config)

	if err != nil {
		ServerLogger.Printf("Failed to listen in address %q: %v\n", address, err)
		return err
	}

	server := http.Server{
		Handler:      s,
		ErrorLog:     ServerLogger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	s.HTTPServer = &server

	go server.Serve(listener)
	return nil
}
