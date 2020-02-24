package core

import (
	"log"
	"net/http"
)

// RestModule a Rest module interface
type RestModule interface {
	Init(logger *log.Logger, configFile string) error
	IsInitialized() bool
	Name() string
	Version() string
	Info() string
	Help(action string) string
	HandleRequest(w http.ResponseWriter, r *http.Request) error
}

// RestAPIResponse represents a outgoing response
type RestAPIResponse map[string]interface{}

// RestAPIHandler signature of a RestHandler callback
type RestAPIHandler func(http.ResponseWriter, *http.Request) error

// RestAPIEndpoints map of endpoints
type RestAPIEndpoints map[string]RestAPIHandler
