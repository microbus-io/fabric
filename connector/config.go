package connector

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// Config returns the value of the config as a string
func (c *Connector) Config(name string) (value string, ok bool) {
	v, ok := c.configs[name]
	return v, ok
}

// ConfigInt returns the value of the config as an integer
func (c *Connector) ConfigInt(name string) (value int, ok bool) {
	v, ok := c.configs[name]
	if !ok {
		return 0, false
	}
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, false
	}
	return int(i), true
}

// ConfigBool returns the value of the config as a boolean
func (c *Connector) ConfigBool(name string) (value bool, ok bool) {
	v, ok := c.configs[name]
	if !ok {
		return false, false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, false
	}
	return b, true
}

// ConfigDuration returns the value of the config as a duration
func (c *Connector) ConfigDuration(name string) (value time.Duration, ok bool) {
	v, ok := c.configs[name]
	if !ok {
		return 0, false
	}
	dur, err := time.ParseDuration(v)
	if err != nil {
		return 0, false
	}
	return dur, true
}

func (c *Connector) loadConfigs() error {
	// Scan the directory hierarchy for env.yaml files
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	for wd != "/" {
		envFileData, err := ioutil.ReadFile(wd + "/env.yaml")
		if err == nil {
			err = readEnvYamlFile(c.hostName, envFileData, c.configs)
			if err != nil {
				return err
			}
		}
		wd = filepath.Dir(wd) // Get the parent path
	}

	// Scan envars
	err = readEnvars(c.hostName, os.Environ(), c.configs)
	if err != nil {
		return err
	}

	return nil
}

func readEnvars(hostName string, environ []string, configs map[string]string) error {
	envarsMap := map[string]string{}
	for _, kv := range environ {
		p := strings.Index(kv, "=")
		if p > 0 {
			envarsMap[strings.ToUpper(kv[:p])] = kv[p+1:]
		}
	}

	// Look for an envar for each suffix of the host name.
	// For example, if the host name is www.example.com this will scan for envars named
	// MICROBUS_WWWEXAMPLECOM_*, MICROBUS_EXAMPLECOM_* and MICROBUS_COM_*.
	segments := strings.Split(hostName, ".")
	for i := len(segments) - 1; i >= 0; i-- {
		h := strings.ToUpper(strings.Join(segments[i:], ""))
		for k, v := range envarsMap {
			if strings.HasPrefix(k, "MICROBUS_"+h+"_") {
				n := k[len("MICROBUS_"+h+"_"):]
				configs[strings.ToLower(n)] = v
			}
		}
	}
	return nil
}

func readEnvYamlFile(hostName string, envFileData []byte, configs map[string]string) error {
	var envFileMap map[string]map[string]string
	err := yaml.Unmarshal(envFileData, &envFileMap)
	if err != nil {
		return err
	}

	// Look for a property map for each suffix of the host name.
	// For example, if the host name is www.example.com this will scan for config maps for
	// www.example.com, example.com and com in this order
	segments := strings.Split(hostName, ".")
	for i := len(segments) - 1; i >= 0; i-- {
		h := strings.Join(segments[i:], ".")
		for n, v := range envFileMap[h] {
			configs[strings.ToLower(n)] = v
		}
	}
	return nil
}
