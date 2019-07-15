// Copyright 2019 Adrien Waksberg
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gpm

import (
  "encoding/json"
  "fmt"
  "io/ioutil"
  "os"
  "os/user"
  "runtime"
)

// Config struct contain the config
type Config struct {
  WalletDir     string            `json:"wallet_dir"`
  WalletDefault string            `json:"wallet_default"`
}

// Init the configuration
func (c *Config) Init() error {
  user, err := user.Current()
  if err != nil {
    return err
  }

  if runtime.GOOS == "darwin" {
    c.WalletDir = fmt.Sprintf("%s/Library/Preferences/mpw", user.HomeDir)
  } else if runtime.GOOS == "windows" {
    c.WalletDir = fmt.Sprintf("%s/AppData/Local/mpw", user.HomeDir)
  } else {
    c.WalletDir = fmt.Sprintf("%s/.config/mpw", user.HomeDir)
  }
  c.WalletDefault = "default"

  return nil
}

// Load the configuration from a file
func (c *Config) Load(path string) error {
  _, err := os.Stat(path)
  if err != nil {
    err = c.Init()
    if err != nil {
      return err
    }
  }

  data, err := ioutil.ReadFile(path)
  if err != nil {
    return err
  }

  err = json.Unmarshal(data, &c)
  if err != nil {
    return err
  }

  return nil
}

// Save the configuration
func (c *Config) Save(path string) error {
  data, err := json.Marshal(&c)
  if err != nil {
    return err
  }

  err = ioutil.WriteFile(path, []byte(data), 0644)
  if err != nil {
    return err
  }

  return nil
}
