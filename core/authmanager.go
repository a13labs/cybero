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
	"net/http"
	"strings"
	"sync"
)

// AuthManager the server security auth
type AuthManager struct {
	authActions map[string]types.RestAPIHandler
}

var (
	// AuthEndpoint the auth endpoint
	AuthEndpoint = "auth"
	authSync     sync.Once
	auth         *AuthManager
)

func (auth *AuthManager) signinAction(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (auth *AuthManager) refreshAction(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// HandleRequest handle a request for the security auth
func (auth *AuthManager) HandleRequest(w http.ResponseWriter, r *http.Request) error {

	logger := GetLogManager().GetLogger()

	// remove /auth/ from url and split
	parts := strings.Split(r.URL.Path[len(AuthEndpoint)+2:], "/")

	// Check if it is an internal action
	if action, ok := auth.authActions[parts[0]]; ok {
		logger.Printf("API builtin action called %q", parts[0])
		return action(w, r)
	}

	return errors.New("Invalid operation")
}

// GetAuthManager initialize the the security auth
func GetAuthManager() *AuthManager {

	logger := GetLogManager().GetLogger()

	authSync.Do(func() {

		logger.Println("Auth: Initializing authentication Layer")
		auth = &AuthManager{}

		// Initialize authentication callbacks maps
		auth.authActions = map[string]types.RestAPIHandler{
			"signin":  auth.signinAction,
			"refresh": auth.refreshAction,
		}

	})

	return auth
}
