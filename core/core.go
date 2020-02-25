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
