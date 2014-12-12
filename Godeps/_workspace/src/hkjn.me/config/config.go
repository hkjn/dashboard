// Package config provides a wrapper around YAML configs.
//
// The default name is config.yaml, with an optional overrides.yaml
// file, for local overrides.
//
// To use, define a struct representing the parts of the YAML config
// you care about. If the importing package has a `config.yaml` with
// the following contents:
//   # This is a comment in config.yaml.
//   foo: 42
//   bar:
//     qux: marmalade
//     baz: 13
// This config is loaded as:
//   cfg := struct{
//     Foo int
//     Bar struct{
//       Qux string
//       Baz int
//     }
//   }{}
//   MustLoad(&cfg)
//   fmt.Println(cfg.Bar.Baz) // Outputs "13".
package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

var (
	defaultConfigName         = "config.yaml"    // default name of YAML config file
	defaultOverridesName      = "overrides.yaml" // default name of optional YAML config with local overrides
	BasePath                  = "."              // where to start looking for configs; relative to importing code
	MaxSteps             uint = 5                // maximum number of directories to step up while looking for configs

)

// Name applies the option to set a name for the YAML config file.
func Name(name string) option {
	return option{"configName", name}
}

// Overrides applies the option to set a name for the YAML overrides file.
func Overrides(name string) option {
	return option{"overridesName", name}
}

// MustLoad is like Load, but panics if the config can't be loaded or
// parsed.
func MustLoad(v interface{}, options ...option) {
	err := Load(v, options...)
	if err != nil {
		panic(fmt.Errorf("FATAL: %v\n", err))
	}
}

// Load parses the YAML-encoded config with specified options, and
// stores the result in the value pointed to by v.
func Load(v interface{}, options ...option) error {
	configName := defaultConfigName
	overridesName := defaultOverridesName
	for _, opt := range options {
		if opt.name == "configName" {
			configName = opt.value
		} else if opt.name == "overridesName" {
			overridesName = opt.value
		} else {
			return fmt.Errorf("internal: bad option name %q", opt.name)
		}
	}
	err := tryLoad(configName, v)
	if err != nil {
		return err
	}
	// Note: Since it's not required to have an overrides.yaml, we
	// treat a failed load as a non-error. It would be nice to log an
	// INFO message at this point to alert the caller that overrides
	// file is missing (to make the feature more discoverable), but we
	// can't use glog in case we're called from AppEngine.
	_ = tryLoad(overridesName, v)
	return nil
}

type option struct {
	name, value string
}

// tryLoad parses the YAML-encoded config in file name and stores the
// result in the value pointed to by v.
//
// tryLoad steps up one directory level at a time, at most MaxSteps
// number of times, until the named config file is found.
func tryLoad(name string, v interface{}) error {
	var err error
	tries := uint(0)
	path := filepath.Join(BasePath, name)
	for tries <= MaxSteps {
		err = loadPath(path, v)
		if err == nil {
			return nil
		} else if os.IsNotExist(err) {
			path = filepath.Join(BasePath, strings.Repeat("../", int(tries+1)), name)
			tries += 1
		} else {
			return err // not missing file; something else is wrong, so bail.
		}
	}
	return fmt.Errorf("failed to find a valid %s: %v", name, err)
}

// loadPath parses the YAML-encoded config at path and stores the
// result in the value pointed to by v.
func loadPath(path string, v interface{}) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return err
		}
		return fmt.Errorf("couldn't read config: %v", err)
	}

	err = yaml.Unmarshal(b, v)
	if err != nil {
		return fmt.Errorf("couldn't unmarshal config: %v", err)
	}
	return nil
}