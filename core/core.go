package core

import "net/http"

// RestAPIResponse represents a outgoing response
type RestAPIResponse map[string]interface{}

// RestAPIHandler signature of a RestHandler callback
type RestAPIHandler func(http.ResponseWriter, *http.Request) error

// RestAPIEndpoints map of endpoints
type RestAPIEndpoints map[string]RestAPIHandler
