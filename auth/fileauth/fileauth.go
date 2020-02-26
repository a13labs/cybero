package main

import (
	"cybero/types"
	"log"
)

type fileauthProvider struct {
	types.RestAPIAuthProvider
}

func (provider fileauthProvider) Initialize(logger *log.Logger, config map[string]interface{}) error {
	return nil
}

func (provider fileauthProvider) Name() string {
	return "File based authentication"
}

func (provider fileauthProvider) Version() string {
	return "0.0.1"
}

func (provider fileauthProvider) Authenticate(credential *types.RestAPICredentials) bool {
	return true
}

func main() {
	// Nothing here, we are a module
}

// CyberoAuthProvider the exported plugin
var CyberoAuthProvider fileauthProvider
