// Copyright 2014 Daniel Kertesz <daniel@spatof.org>
// All rights reserved. This program comes with ABSOLUTELY NO WARRANTY.
// See the file LICENSE for details.

package spock

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type Configuration struct {
	SecretKey string `json:"secret_key"`
}

func NewConfiguration(filename string) (*Configuration, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	var config Configuration
	if err = json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func (cfg *Configuration) Validate() bool {
	if cfg.SecretKey == "" {
		return false
	}

	return true
}
