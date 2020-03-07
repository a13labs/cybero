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

package main

import (
	"cybero/modules/nodemanager"
	"cybero/types"
	cyberotypes "cybero/types"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type nodeManagerHandlerModule struct {
	cyberotypes.CyberoHandlerModule
	moduleActions map[string]types.RestAPIHandler
	logger        *log.Logger
}

func (mod nodeManagerHandlerModule) Initialize(defaultLogger *log.Logger, config map[string]interface{}) error {

	mod.logger = defaultLogger

	// Setup API actions callbacks
	mod.moduleActions = map[string]types.RestAPIHandler{}

	mod.logger.Printf("NodeManager: Initializing module\n")
	return nil
}

func (mod nodeManagerHandlerModule) Name() string {
	return "Node Module"
}

func (mod nodeManagerHandlerModule) Version() string {
	return "0.0.1"
}

func (mod nodeManagerHandlerModule) Info() string {
	return "Manage nodes"
}

func (mod nodeManagerHandlerModule) Endpoint() string {
	return "nodes"
}

func (mod nodeManagerHandlerModule) Help(action string) string {

	switch action {
	default:
		return "Not Implemented yet"
	}
}

func (mod nodeManagerHandlerModule) HandleRequest(w http.ResponseWriter, r *http.Request) error {

	// our endpoint is "/api/nodes"
	endpoint := "/api/" + mod.Endpoint()

	// we should at least recieve /api/nodes/
	if len(r.URL.Path) < len(endpoint)+1 {
		mod.logger.Printf("NodeManager: No action called %q\n", r.URL.Path)
		return errors.New("No module called")
	}

	// remove /api/nodes/ from url and split
	parts := strings.Split(r.URL.Path[len(endpoint)+1:], "/")

	if len(parts) == 0 {
		mod.logger.Printf("NodeManager: No action called %q\n", r.URL.Path[len(endpoint)+1:])
		return errors.New("No action called")
	}

	// Check if it is an handled action
	if action, ok := mod.moduleActions[parts[0]]; ok {
		mod.logger.Printf("NodeManager: builtin action called %q", parts[0])
		return action(w, r)
	}

	return errors.New("Invalid operation")
}

/* RestAPI Handlers */

func (mod nodeManagerHandlerModule) createAction(w http.ResponseWriter, r *http.Request) error {

	if r.Method != "POST" {
		mod.logger.Printf("NodeManager: Method not supported, action: create, method: %q\n", r.Method)
		return errors.New("Action not supported")
	}

	// We try to read the body of the request
	var node nodemanager.Node
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&node)
	if err != nil {
		fmt.Printf("NodeManager: Error reading body, action: create, err: %v\n", err)
		return err
	}

	// TODO: validate incoming node data pherarps???

	// If we receive a valid node we try to create it
	err = nodemanager.GetNodeManager(mod.logger).NodeCreate(&node)
	if err != nil {
		fmt.Printf("NodeManager: Error creating node, node: %q, err: %v\n", node.Name, err)
		return err
	}

	return nil
}

func destroyAction(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func signalAction(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func execAction(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func main() {
	// Nothing here, we are a module
}

// CyberoRestHandler the exported plugin
var CyberoRestHandler nodeManagerHandlerModule
