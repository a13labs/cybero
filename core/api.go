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
	"cybero/api/orchestrator"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"plugin"
	"strings"
)

// defaultPath location to find modukes
var (
	defaultPath       = "/usr/lib/cybero"
	defaultConfigFile = "/etc/cybero/daemon.json"
	defaultLogger     *log.Logger
	modulesCache      map[string]interface{}
	apiActions        map[string]RestAPIHandler
)

// getModule Get module based on name
func getModule(name string) (RestModule, error) {

	// Check if we have already the module on the cache
	moduleImpl, ok := modulesCache[name]

	if !ok {
		// No module on the cache, try to load it from modules folder
		module, err := plugin.Open(path.Join(defaultPath, name+".so"))

		if err != nil {
			defaultLogger.Printf("Error processing module %q: %v\n", name, err)
			return nil, err
		}

		_, err = module.Lookup("Name")

		if err != nil {
			defaultLogger.Printf("Error processing module %q: %v\n", name, err)
			return nil, err
		}

		_, err = module.Lookup("Version")

		if err != nil {
			defaultLogger.Printf("Error processing mod %q: %v\n", name, err)
			return nil, err
		}

		symModule, err := module.Lookup("CyberoModule")

		if err != nil {
			defaultLogger.Printf("Error processing file %q: %v\n", name, err)
			return nil, err
		}

		moduleImpl, ok := symModule.(RestModule)
		if !ok {
			defaultLogger.Printf("Error processing file %q: %v\n", name, err)
			return nil, err
		}

		// Initialize plugin with arguments
		if err = moduleImpl.Init(defaultLogger, defaultConfigFile); err != nil {
			defaultLogger.Printf("API: Error initializing module %q: %v\n", name, err)
			return nil, err
		}

		defaultLogger.Printf("API: Module loaded and initialized: %v\n", name)
		modulesCache[name] = moduleImpl
	}

	restModule := moduleImpl.(RestModule)
	if !restModule.IsInitialized() {
		restModule.Init(defaultLogger, defaultConfigFile)
	}

	return restModule, nil
}

// listAction Send a list of available modules
func listAction(w http.ResponseWriter, r *http.Request) error {

	modules := []map[string]interface{}{
		map[string]interface{}{"name": "builtin", "version": "-"},
	}

	for _, moduleImpl := range modulesCache {
		module := moduleImpl.(RestModule)
		modules = append(modules, map[string]interface{}{"name": module.Name(), "version": module.Version()})
	}

	encoder := json.NewEncoder(w)
	code, msg := 0, map[string]interface{}{"modules": modules}

	encoder.Encode(RestAPIResponse{
		"Status":   code,
		"Response": msg,
	})

	return nil
}

// infoAction Send information about a specific module
func infoAction(w http.ResponseWriter, r *http.Request) error {

	encoder := json.NewEncoder(w)

	module := r.URL.Query().Get("module")
	code, msg := -1, map[string]interface{}{"Error": fmt.Sprintf("Error module does not exits %q\n", module)}

	if module == "" {
		code, msg = 0, map[string]interface{}{"Info": fmt.Sprintf("Builtin module, contains builtin functions to handle modules.\n")}
	} else if module, err := getModule(module); err == nil {
		code, msg = 0, map[string]interface{}{"Info": module.Info()}
	}

	encoder.Encode(RestAPIResponse{
		"Status":   code,
		"Response": msg,
	})

	return nil
}

// helpAction Send help about a specific module
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

	} else if module, err := getModule(module); err == nil {
		code, msg = 0, map[string]interface{}{"Help": module.Help(action)}
	}

	encoder.Encode(RestAPIResponse{
		"Status":   code,
		"Response": msg,
	})

	return nil
}

// InitializeAPI Initialize modules compoment
func InitializeAPI(logger *log.Logger, configFile string, path string) {

	defaultLogger = logger
	defaultConfigFile = configFile
	defaultPath = path

	defaultLogger.Printf("API: Initializing modules\n")

	// Initialize modules cache
	modulesCache = map[string]interface{}{
		"orchestrator": orchestrator.Module,
	}

	filepath.Walk(defaultPath, func(fPath string, info os.FileInfo, err error) error {

		if err != nil {
			defaultLogger.Printf("API: Error accessing path %q: %v\n", fPath, err)
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// Extract module name from filename /xx/xx/module.xxx -> module
		moduleName := strings.ReplaceAll(filepath.Base(info.Name()), filepath.Ext(info.Name()), "")

		if _, err := getModule(moduleName); err != nil {
			defaultLogger.Printf("API: Error loading module %q: %v\n", moduleName, err)
			return err
		}

		return nil
	})

	apiActions = map[string]RestAPIHandler{
		"list": listAction,
		"info": infoAction,
		"help": helpAction,
	}
}

// HandleRequest pass the request to an external module
func HandleRequest(w http.ResponseWriter, r *http.Request) error {

	// remove /api/ from url and split
	parts := strings.Split(r.URL.Path[5:], "/")

	if len(parts) == 0 {
		return errors.New("No module called")
	}

	// Check if it is an internal action
	if action, ok := apiActions[parts[0]]; ok {
		defaultLogger.Printf("API builtin action called %q", parts[0])
		return action(w, r)
	}

	// Check if is an action related to a module
	if module, err := getModule(parts[0]); err == nil {
		defaultLogger.Printf("API  module %q called", parts[0])
		return module.HandleRequest(w, r)
	}

	return errors.New("Invalid operation")
}
