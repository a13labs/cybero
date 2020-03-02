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

func (mod *ModulesManager) loadModulesFromFolder(modulesPath string) {

	cfg := GetConfigManager().GetConfig()
	logger := GetLogManager().GetLogger()

	filepath.Walk(modulesPath, func(fPath string, info os.FileInfo, err error) error {
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
		module, err := plugin.Open(path.Join(modulesPath, name+".so"))

		if err != nil {
			logger.Printf("API: Error processing module %q: %v\n", name, err)
			return nil
		}

		if symModule, err := module.Lookup("CyberoModule"); err != nil {

			logger.Println(name)

			if moduleImpl, ok := symModule.(types.RestAPIModule); ok {

				if config, ok := cfg.Modules.Configuration[name]; ok {

					if !config.Enabled {
						// Plugin disabled on configuration skipping
						return nil
					}

					// Initialize plugin with arguments
					if err = moduleImpl.Initialize(logger, config.Config); err != nil {
						logger.Printf("ModulesManager: Error initializing module %q: %v\n", name, err)
						return nil
					}

					logger.Printf("ModulesManager: Module loaded and initialized: %q, version: %q\n", moduleImpl.Name(), moduleImpl.Version())
					mod.apiModules[name] = moduleImpl
				} else {
					// No configuration found, default behaviour is to disable the plugin
					return nil
				}

			} else {
				logger.Printf("ModuleManager: Error processing file %q: %v\n", name, err)
				return nil
			}

		}

		if symModule, err := module.Lookup("CyberoAuthProvider"); err != nil {

			if moduleImpl, ok := symModule.(types.RestAPIAuthProvider); ok {

				if config, ok := cfg.Modules.Configuration[name]; ok {

					if !config.Enabled {
						// Plugin disabled on configuration skipping
						return nil
					}

					// Initialize plugin with arguments
					if err = moduleImpl.Initialize(logger, config.Config); err != nil {
						logger.Printf("ModulesManager: Error initializing module %q: %v\n", name, err)
						return nil
					}

					logger.Printf("ModulesManager: Module loaded and initialized: %q, version: %q\n", moduleImpl.Name(), moduleImpl.Version())
					mod.authModules[name] = moduleImpl
				} else {
					// No configuration found, default behaviour is to disable the plugin
					return nil
				}

			} else {
				logger.Printf("ModuleManager: Error processing module %q: %v\n", name, err)
				return nil
			}

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
func (mod *ModulesManager) GetAuthModule(name string) (types.RestAPIAuthProvider, bool) {

	if module, ok := mod.authModules[name]; ok {
		return module.(types.RestAPIAuthProvider), ok
	}
	return nil, false
}

// GetAPIModule get an registered rest api module
func (mod *ModulesManager) GetAPIModule(name string) (types.RestAPIModule, bool) {
	if module, ok := mod.apiModules[name]; ok {
		return module.(types.RestAPIModule), ok
	}
	return nil, false
}

// GetModuleManager returns the current module manager
func GetModuleManager() *ModulesManager {

	modulesSync.Do(func() {

		// Get the current configuration
		cfg := GetConfigManager().GetConfig()
		logger := GetLogManager().GetLogger()

		// Instanciate the logManager
		modulesManager = &ModulesManager{}

		logger.Printf("ModuleManager: Initializing modules\n")

		// Initialize modules cache
		modulesManager.apiModules = make(map[string]interface{})
		modulesManager.authModules = make(map[string]interface{})

		// try to load modules from the configured folder
		modulesManager.loadModulesFromFolder(cfg.Modules.Path)
	})

	if configManager == nil {
		log.Println("ModulesManager: Something wen't wrong when creating modules manager!")
	}

	return modulesManager
}
