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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// APIManager location to find modukes
type APIManager struct {
	apiActions map[string]types.CyberoHandler
}

var (
	// APIEndpoint the api endpoint
	APIEndpoint = "api"
	apiSync     sync.Once
	api         *APIManager
)

func listAction(w http.ResponseWriter, r *http.Request) error {

	modules := []map[string]interface{}{
		map[string]interface{}{"name": "builtin", "version": "-"},
	}

	for _, moduleImpl := range GetModuleManager().GetAPIModules() {
		module := moduleImpl.(types.CyberoHandlerModule)
		modules = append(modules, map[string]interface{}{
			"name":     module.Name(),
			"version":  module.Version(),
			"endpoint": module.Endpoint(),
		})
	}

	encoder := json.NewEncoder(w)
	code, msg := 0, map[string]interface{}{"modules": modules}

	encoder.Encode(types.CyberoResponse{
		"Status":   code,
		"Response": msg,
	})

	return nil
}

func infoAction(w http.ResponseWriter, r *http.Request) error {

	encoder := json.NewEncoder(w)

	module := r.URL.Query().Get("module")
	code, msg := -1, map[string]interface{}{"Error": fmt.Sprintf("Error module does not exits %q\n", module)}

	if module == "" {
		code, msg = 0, map[string]interface{}{"Info": fmt.Sprintf("Builtin module, contains builtin functions to handle modules.\n")}
	} else if module, ok := GetModuleManager().GetAPIModule(module); ok {
		code, msg = 0, map[string]interface{}{"Info": module.Info()}
	}

	encoder.Encode(types.CyberoResponse{
		"Status":   code,
		"Response": msg,
	})

	return nil
}

func helpAction(w http.ResponseWriter, r *http.Request) error {

	encoder := json.NewEncoder(w)

	module := r.URL.Query().Get("module")
	action := r.URL.Query().Get("action")
	code, msg := -1, map[string]interface{}{"Error": fmt.Sprintf("Error module or action does not exits %q\n", module)}

	if module == "" {

		if action == "list" {
			code, msg = 0, map[string]interface{}{"Help": "Returns a list of available modules"}
		}

		if action == "info" {
			code, msg = 0, map[string]interface{}{"Help": "Returns information about a specific module"}
		}

	} else if module, ok := GetModuleManager().GetAPIModule(module); ok {
		code, msg = 0, map[string]interface{}{"Help": module.Help(action)}
	}

	encoder.Encode(types.CyberoResponse{
		"Status":   code,
		"Response": msg,
	})

	return nil
}

// HandleRequest pass the request to an external module
func (api *APIManager) HandleRequest(w http.ResponseWriter, r *http.Request) error {

	logger := GetLogManager().GetLogger()

	// our endpoint is /api
	endpoint := "/" + APIEndpoint

	// we should at least recieve /api/
	if len(r.URL.Path) < len(endpoint)+1 {
		logger.Printf("API: No module called %q\n", r.URL.Path)
		return errors.New("No module called")
	}

	// remove /api/ from url and split
	parts := strings.Split(r.URL.Path[len(endpoint)+1:], "/")

	if len(parts) == 0 {
		logger.Printf("API: No module called %q\n", r.URL.Path[len(APIEndpoint)+2:])
		return errors.New("No module called")
	}

	// Check if it is an internal action
	if action, ok := api.apiActions[parts[0]]; ok {
		logger.Printf("API: builtin action called %q", parts[0])
		return action(w, r)
	}

	// Check if is an action related to a module
	if module, ok := GetModuleManager().GetAPIModule(parts[0]); ok {
		logger.Printf("API: module %q called", parts[0])
		return module.HandleRequest(w, r)
	}

	return errors.New("Invalid operation")
}

// GetAPIManager Initialize modules compoment
func GetAPIManager() *APIManager {

	logger := GetLogManager().GetLogger()

	apiSync.Do(func() {

		api = &APIManager{}
		logger.Printf("APIManager: Initializing modules\n")

		// Setup API actions callbacks
		api.apiActions = map[string]types.CyberoHandler{
			"list": listAction,
			"info": infoAction,
			"help": helpAction,
		}
	})

	if api == nil {
		logger.Println("APIManager: Something wen't wrong when creating API manager!")
	}

	return api
}
