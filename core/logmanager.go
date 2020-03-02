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
	"log"
	"os"
	"sync"
)

// LogManager the log manager
type LogManager struct {
	coreLogger *log.Logger
}

var (
	coreSync   sync.Once
	logManager *LogManager
)

// GetLogger get the curremt logger
func (log *LogManager) GetLogger() *log.Logger {
	return log.coreLogger
}

// GetLogManager get the current logger
func GetLogManager() *LogManager {

	coreSync.Do(func() {

		// Get the current configuration
		config := GetConfigManager().GetConfig()

		// Instanciate the log manager
		logManager = &LogManager{}

		// Setup the log file
		logFile, err := os.OpenFile(config.LogFile,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

		// Initialize logging to a file
		if err != nil {
			log.Println(err)
			return
		}

		logManager.coreLogger = log.New(logFile, "", log.LstdFlags)
	})

	if configManager == nil {
		log.Println("LogManager: Something wen't wrong when creating config manager!")
	}

	return logManager
}
