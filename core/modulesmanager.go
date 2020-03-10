package core

import (
	"cybero/types"
	"log"
	"os"
	"path"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
)

// ModulesManager holds and maintain the current modules stack
type ModulesManager struct {
	apiModules  map[string]interface{}
	authModules map[string]interface{}
}

var (
	modulesSync    sync.Once
	modulesManager *ModulesManager
)

//LoadModules load all modules
func (mod *ModulesManager) LoadModules() {

	cfg := GetConfigManager().GetConfig()
	logger := GetLogManager().GetLogger()

	filepath.Walk(cfg.Modules.Path, func(fPath string, info os.FileInfo, err error) error {

		if err != nil {
			logger.Printf("ModuleManager: Error accessing path %q: %v\n", fPath, err)
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// Extract module name from filename /xx/xx/module.xxx -> module
		name := strings.ReplaceAll(filepath.Base(info.Name()), filepath.Ext(info.Name()), "")

		// No module on the cache, try to load it from modules folder
		pluginFile := path.Join(cfg.Modules.Path, name+".so")
		module, err := plugin.Open(pluginFile)

		if err != nil {
			logger.Printf("API: Error processing module %q: %v\n", name, err)
			return nil
		}

		// Check if module is a Cybero handler
		logger.Printf("ModuleManager: Checking if %q is a Cybero handler\n", pluginFile)
		if symModule, err := module.Lookup("CyberoRestHandler"); err == nil {

			if moduleImpl, ok := symModule.(types.CyberoHandlerModule); ok {

				if config, ok := cfg.Modules.Configuration[name]; ok {

					if config.Enabled {

						// Initialize plugin with arguments
						if err = moduleImpl.Initialize(logger, config.Config); err != nil {
							logger.Printf("ModulesManager: Error initializing module %q: %v\n", name, err)
							return nil
						}

						logger.Printf("ModulesManager: Module loaded and initialized: %q, version: %q\n", moduleImpl.Name(), moduleImpl.Version())
						mod.apiModules[moduleImpl.Endpoint()] = moduleImpl

						return nil
					}

				}
			}

			logger.Printf("ModuleManager: Plugin %q not loaded. is it enabled?\n", name)
			return nil
		}

		// Check if module is a Authentication provider
		logger.Printf("ModuleManager: Checking if %q is a Auth provider\n", pluginFile)
		if symModule, err := module.Lookup("CyberoAuthProvider"); err == nil {

			if moduleImpl, ok := symModule.(types.CyberoAuthModule); ok {

				if config, ok := cfg.Modules.Configuration[name]; ok {

					if config.Enabled {

						// Initialize plugin with arguments
						if err = moduleImpl.Initialize(logger, config.Config); err != nil {
							logger.Printf("ModulesManager: Error initializing module %q: %v\n", name, err)
							return nil
						}

						logger.Printf("ModulesManager: Module loaded and initialized: %q, version: %q\n", moduleImpl.Name(), moduleImpl.Version())
						mod.authModules[name] = moduleImpl
						return nil
					}

				}
			}

			logger.Printf("ModuleManager: Plugin %q not loaded. is it enabled?\n", name)
			return nil
		}

		logger.Printf("ModuleManager: Error processing file %q: %v\n", name, err)
		return nil
	})
}

// GetAuthModules get an registered authentication modules
func (mod *ModulesManager) GetAuthModules() map[string]interface{} {
	return mod.authModules
}

// GetAPIModules get an registered rest api modules
func (mod *ModulesManager) GetAPIModules() map[string]interface{} {
	return mod.apiModules
}

// GetAuthModule get an registered authentication module
func (mod *ModulesManager) GetAuthModule(name string) (types.CyberoAuthModule, bool) {

	if module, ok := mod.authModules[name]; ok {
		return module.(types.CyberoAuthModule), ok
	}
	return nil, false
}

// GetAPIModule get an registered rest api module
func (mod *ModulesManager) GetAPIModule(name string) (types.CyberoHandlerModule, bool) {
	if module, ok := mod.apiModules[name]; ok {
		return module.(types.CyberoHandlerModule), ok
	}
	return nil, false
}

// GetModuleManager returns the current module manager
func GetModuleManager() *ModulesManager {

	modulesSync.Do(func() {

		// Get the current configuration
		logger := GetLogManager().GetLogger()

		// Instanciate the logManager
		modulesManager = &ModulesManager{}

		logger.Printf("ModuleManager: Initializing modules\n")

		// Initialize modules cache
		modulesManager.apiModules = make(map[string]interface{})
		modulesManager.authModules = make(map[string]interface{})
	})

	if configManager == nil {
		log.Println("ModulesManager: Something wen't wrong when creating modules manager!")
	}

	return modulesManager
}
