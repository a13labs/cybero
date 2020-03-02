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
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
)

// ConfigManager represents a config manager struct
type ConfigManager struct {
	masterConfig *types.RestAPIConfig
	configFile   string
}

var (
	configSync    sync.Once
	configManager *ConfigManager
)

func (config *ConfigManager) loadFromFile(configFile string) error {

	fileDscr, err := os.Open(configFile)
	if err != nil {
		fmt.Printf("RestAPIServer: Error loading config file %q: %v\n", configFile, err)
		return err
	}
	defer fileDscr.Close()

	decoder := json.NewDecoder(fileDscr)

	err = decoder.Decode(config.masterConfig)
	if err != nil {
		fmt.Printf("RestAPIServer: Error loading config file %q: %v\n", configFile, err)
		return err
	}

	return nil
}

func (config *ConfigManager) loadFormArgs() {

	if config.masterConfig == nil {
		config.masterConfig = &types.RestAPIConfig{}
	}

	// Try to load configuration from arguments
	// TODO: this need to be revisited we want args to override
	// config, therefore we need to find a away to allow
	// the config file to be load side-by-side with arguments
	flag.StringVar(&config.configFile, "config", "/etc/cybero/config.json", "Service config file")
	flag.StringVar(&config.masterConfig.LogFile, "logfile", "/var/log/cybero.log", "Log file name")
	flag.StringVar(&config.masterConfig.Socket, "socket", "/var/run/cybero.sock", "Unix socket file")
	flag.BoolVar(&config.masterConfig.TLS, "tls", false, "Use TLS encryption")
	flag.StringVar(&config.masterConfig.CertPEM, "pem", "", "TLS PEM file")
	flag.StringVar(&config.masterConfig.CertKey, "key", "", "TLS key file")
	flag.StringVar(&config.masterConfig.Modules.Path, "modules", "/var/lib/modules", "Modiles location")
	flag.Parse()
}

// GetConfig access to the current config manager
func (config *ConfigManager) GetConfig() *types.RestAPIConfig {
	return config.masterConfig
}

// GetConfigManager access to the current config manager
func GetConfigManager() *ConfigManager {

	configSync.Do(func() {

		configManager = &ConfigManager{}

		// first we load configuration from arguments
		configManager.loadFormArgs()

		// load configuration file
		if configManager.configFile != "" {
			if err := configManager.loadFromFile(configManager.configFile); err != nil {
				return
			}
		}
	})

	if configManager == nil {
		log.Println("ConfigManager: Something wen't wrong when creating config manager!")
	}

	return configManager
}
