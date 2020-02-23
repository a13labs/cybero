package core

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"plugin"
)

// ModulesLocation location to find modukes
var ModulesLocation string

// ModulesLogger logger for modules
var ModulesLogger *log.Logger

var builtinActions = ServiceAction{
	"list": listModules,
	"info": moduleInfo,
	"help": moduleHelp,
}

func loadModule(pluginPath string) (ServiceModule, error) {

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

	moduleImpl, ok := symModule.(ServiceModule)
	if !ok {
		ModulesLogger.Printf("Error processing file %q: %v\n", pluginPath, err)
		return nil, errors.New("Invalid plugin file")
	}

	return moduleImpl, nil
}

// ModuleHandle pass the request to an external module
func ModuleHandle(request *ServiceRequest) (int, map[string]interface{}) {

	if request.Module == "builtin" {
		action, ok := builtinActions[request.Action]

		if ok {
			return action(request)
		}

		return -1, map[string]interface{}{"Error": fmt.Sprintf("Error, action %q does not exits\n", request.Action)}
	}

	if module, err := loadModule(path.Join(ModulesLocation, request.Module+".so")); err == nil {
		return module.Handle(request)
	}

	return -1, map[string]interface{}{"Error": fmt.Sprintf("Error opening module %q\n", request.Module)}
}

func listModules(request *ServiceRequest) (int, map[string]interface{}) {

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

	return 0, map[string]interface{}{"modules": modules}
}

func moduleInfo(request *ServiceRequest) (int, map[string]interface{}) {

	if request.Module == "builtin" {
		return 0, map[string]interface{}{"Info": "Built-in module"}
	}

	if module, err := loadModule(path.Join(ModulesLocation, request.Module+".so")); err == nil {
		return 0, map[string]interface{}{"Info": module.Info()}
	}

	return -1, map[string]interface{}{"Error": fmt.Sprintf("Error opening module %q\n", request.Module)}
}

func moduleHelp(request *ServiceRequest) (int, map[string]interface{}) {

	if _, ok := request.Parameters["action"]; !ok {
		return 0, map[string]interface{}{"Help": "Returns help about a specific action"}
	}

	action := request.Parameters["action"].(string)

	// We handle builtin modules
	if request.Module == "builtin" {

		if action == "list" {
			return 0, map[string]interface{}{"Help": "Returns a list of available modules"}
		}

		if action == "info" {
			return 0, map[string]interface{}{"Help": "Returns information about a specific module"}
		}

		return -1, map[string]interface{}{"Error": fmt.Sprintf("Action not available: %q", action)}
	}

	// If is not a builtin load the module
	if module, err := loadModule(path.Join(ModulesLocation, request.Module+".so")); err == nil {
		return 0, map[string]interface{}{"Help": module.Help(action)}
	}

	return -1, map[string]interface{}{"Error": fmt.Sprintf("Module does not exists: %q\n", request.Module)}
}
