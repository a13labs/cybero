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

package types

import (
	"log"
	"net/http"

	"github.com/dgrijalva/jwt-go"
)

// CyberoModule the basic module interface
type CyberoModule interface {
	Initialize(*log.Logger, map[string]interface{}) error
	IsInitialized() bool
	Name() string
	Version() string
	Info() string
}

// CyberoHandlerModule implements a Cybero
type CyberoHandlerModule interface {
	CyberoModule
	Actions() map[string]interface{}
	Help(string) string
	Endpoint() string
	HandleRequest(w http.ResponseWriter, r *http.Request) error
}

// CyberoAuthModule implements a Cybero
type CyberoAuthModule interface {
	CyberoModule
	Authenticate(*CyberoCredentials) bool
}

// CyberoResponse represents a outgoing response
type CyberoResponse map[string]interface{}

// CyberoHandler signature of a RestHandler callback
type CyberoHandler func(http.ResponseWriter, *http.Request) error

// CyberoEndpoints map of endpoints
type CyberoEndpoints map[string]CyberoHandler

// CyberoModuleConfig configuration of a module
type CyberoModuleConfig struct {
	Enabled bool                   `json:"enabled"`
	Config  map[string]interface{} `json:"config"`
}

// CyberoModulesConfig config part of modules
type CyberoModulesConfig struct {
	Path          string                        `json:"path"`
	Configuration map[string]CyberoModuleConfig `json:"configs"`
}

// CyberoAuthConfig config part of authentication
type CyberoAuthConfig struct {
	Path     string                 `json:"path"`
	Provider string                 `json:"provider"`
	Config   map[string]interface{} `json:"config"`
	Secret   []byte                 `json:"secret"`
}

// CyberoServerConfig The server configuration structure
type CyberoServerConfig struct {
	Socket  string              `json:"socket"`
	TLS     bool                `json:"tls"`
	CertPEM string              `json:"certpem"`
	CertKey string              `json:"certkey"`
	LogFile string              `json:"logfile"`
	Modules CyberoModulesConfig `json:"modules"`
	Auth    CyberoAuthConfig    `json:"auth"`
}

// CyberoCredentials json signin structure
type CyberoCredentials struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

// CyberoClaims  json claim structure
type CyberoClaims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}
