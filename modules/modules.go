package modules

import (
	"cybero/core"
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

// RestModule represents a module plugin
type RestModule interface {
	Name() string
	Version() string
	Info() string
	Help(action string) string
	HandleRequest(w http.ResponseWriter, r *http.Request) error
}

// ModulesLocation location to find modukes
var ModulesLocation string

// ModulesLogger logger for modules
var ModulesLogger *log.Logger

func loadModule(pluginPath string) (RestModule, error) {

	module, err := plugin.Open(pluginPath)

	if err != nil {
		ModulesLogger.Printf("Error processing file %q: %v\n", pluginPath, err)
		return nil, errors.New("Invalid plugin file")
	}

	_, err = module.Lookup("Name")

	if err != nil {
		ModulesLogger.Printf("Error processing file %q: %v\n", pluginPath, err)
		return nil, errors.New("Invalid plugin file")
	}

	_, err = module.Lookup("Version")

	if err != nil {
		ModulesLogger.Printf("Error processing file %q: %v\n", pluginPath, err)
		return nil, errors.New("Invalid plugin file")
	}

	symModule, err := module.Lookup("Module")

	if err != nil {
		ModulesLogger.Printf("Error processing file %q: %v\n", pluginPath, err)
		return nil, errors.New("Invalid plugin file")
	}

	moduleImpl, ok := symModule.(RestModule)
	if !ok {
		ModulesLogger.Printf("Error processing file %q: %v\n", pluginPath, err)
		return nil, errors.New("Invalid plugin file")
	}

	return moduleImpl, nil
}

// ModuleHandle pass the request to an external module
func ModuleHandle(w http.ResponseWriter, r *http.Request) error {

	parts := strings.Split(r.URL.Path[1:], "/")

	if len(parts) < 3 {
		return errors.New("Invalid operation")
	}

	// Module or action name
	module := parts[2]

	switch r.Method {
	case "GET":
		switch module {
		case "list":
			return listModules(w, r)
		case "info":
			return moduleInfo(w, r)
		case "help":
			return moduleHelp(w, r)
		default:
			if module, err := loadModule(path.Join(ModulesLocation, module+".so")); err == nil {
				return module.HandleRequest(w, r)
			}
			return errors.New("Invalid operation")
		}
	case "POST":
	default:
		if module, err := loadModule(path.Join(ModulesLocation, module+".so")); err == nil {
			return module.HandleRequest(w, r)
		}
		return errors.New("Invalid operation")
	}

	return errors.New("Invalid operation")
}

func listModules(w http.ResponseWriter, r *http.Request) error {

	modules := []map[string]interface{}{
		map[string]interface{}{"name": "builtin", "version": "-"},
	}

	filepath.Walk(ModulesLocation, func(fPath string, info os.FileInfo, err error) error {

		if err != nil {
			ModulesLogger.Printf("Error accessing path %q: %v\n", fPath, err)
			return nil
		}

		if info.IsDir() || filepath.Ext(info.Name()) != "so" {
			return nil
		}

		if module, err := loadModule(path.Join(fPath, info.Name())); err == nil {
			modules = append(modules, map[string]interface{}{"name": module.Name(), "version": module.Version()})
		}

		return nil
	})

	// If we arrive here means something went wrong
	encoder := json.NewEncoder(w)
	code, msg := -1, map[string]interface{}{"modules": modules}

	encoder.Encode(core.RestAPIResponse{
		"Status":   code,
		"Response": msg,
	})

	return nil
}

func moduleInfo(w http.ResponseWriter, r *http.Request) error {

	encoder := json.NewEncoder(w)

	parts := strings.Split(r.URL.Path[1:], "/")

	if len(parts) < 3 {
		return errors.New("Invalid operation")
	}

	module := r.URL.Query().Get("module")
	code, msg := -1, map[string]interface{}{"Error": fmt.Sprintf("Error module does not exits %q\n", module)}

	if module == "" {
		code, msg = 0, map[string]interface{}{"Info": fmt.Sprintf("Builtin module, contains builtin functions to handle modules.\n")}
	} else if module, err := loadModule(path.Join(ModulesLocation, module+".so")); err == nil {
		code, msg = 0, map[string]interface{}{"Info": module.Info()}
	}

	encoder.Encode(core.RestAPIResponse{
		"Status":   code,
		"Response": msg,
	})

	return nil
}

func moduleHelp(w http.ResponseWriter, r *http.Request) error {

	encoder := json.NewEncoder(w)

	parts := strings.Split(r.URL.Path[1:], "/")

	if len(parts) < 3 {
		return errors.New("Invalid operation")
	}

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

	} else if module, err := loadModule(path.Join(ModulesLocation, module+".so")); err == nil {
		code, msg = 0, map[string]interface{}{"Help": module.Help(action)}
	}

	encoder.Encode(core.RestAPIResponse{
		"Status":   code,
		"Response": msg,
	})

	return nil
}
