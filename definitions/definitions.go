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

package definitions

import (
	"log"
	"net/http"

	"github.com/dgrijalva/jwt-go"
)

// RestAPIModule a Rest module interface
type RestAPIModule interface {
	Initialize(logger *log.Logger, config map[string]interface{}) error
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

// RestAPIModuleConfig configuration of a module
type RestAPIModuleConfig struct {
	Enabled bool                   `json:"enabled"`
	Config  map[string]interface{} `json:"config"`
}

// RestAPIModulesConfig config part of modules
type RestAPIModulesConfig struct {
	Path          string                         `json:"path"`
	Configuration map[string]RestAPIModuleConfig `json:"configs"`
}

// RestAPIAuthProviderConfig config part of authentication
type RestAPIAuthProviderConfig struct {
	Config map[string]interface{} `json:"config"`
}

// RestAPIAuthConfig config part of authentication
type RestAPIAuthConfig struct {
	Path     string                    `json:"path"`
	Provider string                    `json:"provider"`
	Config   RestAPIAuthProviderConfig `json:"config"`
	Secret   []byte                    `json:"secret"`
}

// RestAPIConfig The server configuration structure
type RestAPIConfig struct {
	Socket  string               `json:"socket"`
	TLS     bool                 `json:"tls"`
	CertPEM string               `json:"certpem"`
	CertKey string               `json:"certkey"`
	LogFile string               `json:"logfile"`
	Modules RestAPIModulesConfig `json:"modules"`
	Auth    RestAPIAuthConfig    `json:"auth"`
}

// RestAPICredentials json signin structure
type RestAPICredentials struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

// RestAPIClaims  json claim structure
type RestAPIClaims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

// RestAPIAuthProvider a Rest module interface
type RestAPIAuthProvider interface {
	Initialize(logger *log.Logger, config RestAPIModuleConfig) error
	IsInitialized() bool
	Name() string
	Version() string
	Authenticate(credential *RestAPICredentials) bool
}