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
	"cybero/types"
	"errors"
	"log"
	"net/http"
	"path"
	"plugin"
	"strings"
)

var (
	authLogger *log.Logger
	// AuthEndpoint the auth endpoint
	AuthEndpoint = "auth"
)

// Auth the server security auth
type Auth struct {
	authActions  map[string]types.RestAPIHandler
	authConfig   types.RestAPIAuthConfig
	authProvider types.RestAPIAuthProvider
}

func (auth *Auth) signinAction(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (auth *Auth) refreshAction(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// Initialize initialize the the security auth
func (auth *Auth) Initialize(logger *log.Logger, config *types.RestAPIConfig) error {

	var ok bool

	authLogger = logger
	auth.authConfig = config.Auth
	providerFile := path.Join(auth.authConfig.Path, auth.authConfig.Provider+".so")
	module, err := plugin.Open(providerFile)

	if err != nil {
		authLogger.Printf("Auth: Error processing provider %q: %v\n", auth.authConfig.Provider, err)
		return err
	}

	symModule, err := module.Lookup("CyberoAuthProvider")

	if err != nil {
		authLogger.Printf("Auth: Error processing file %q: %v\n", auth.authConfig.Provider, err)
		return err
	}

	auth.authProvider, ok = symModule.(types.RestAPIAuthProvider)
	if !ok {
		authLogger.Printf("Auth: Error processing file %q: %v\n", auth.authConfig.Provider, err)
		return err
	}

	// Initialize plugin with arguments
	if err = auth.authProvider.Initialize(authLogger, auth.authConfig.Config); err != nil {
		authLogger.Printf("Auth: Error initializing provider %q: %v\n", auth.authConfig.Provider, err)
		return err
	}

	authLogger.Printf("Auth: Provider loaded and initialized: %q, version: %q\n", auth.authProvider.Name(), auth.authProvider.Version())

	auth.authActions = map[string]types.RestAPIHandler{
		"signin":  auth.signinAction,
		"refresh": auth.refreshAction,
	}

	return nil
}

// HandleRequest handle a request for the security auth
func (auth *Auth) HandleRequest(w http.ResponseWriter, r *http.Request) error {

	// remove /auth/ from url and split
	parts := strings.Split(r.URL.Path[len(AuthEndpoint)+2:], "/")

	// Check if it is an internal action
	if action, ok := auth.authActions[parts[0]]; ok {
		authLogger.Printf("API builtin action called %q", parts[0])
		return action(w, r)
	}

	return errors.New("Invalid operation")
}
