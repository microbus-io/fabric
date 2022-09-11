package connector

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var config map[string]interface{}

// Config returns the value if it exists and has the correct type.
//
// Example:
//
//	k=5
//	Config[int]("k") -> 5, true
//	Config[string]("k") -> "", false
func Config[V any](name string) (val V, ok bool) {
	if config == nil {
		// Detect configuration files
		configFiles, err := scanDirectory(".", false)
		if err != nil {

		}

		// Load from all discovered configuration files
		config, err = loadConfigFiles(configFiles...)
		if err != nil {
			//...
		}

		// Load environment config and overwrite existing values
		envCfg := loadEnvConfig()
		for k, v := range envCfg {
			config[k] = v
		}
	}

	if v, ok := config[name]; ok {
		val, ok = v.(V)
		if !ok {
			return val, false
		}
		return val, true
	}
	return val, false
}

// loadEnvConfig loads the configuration from environment variables.
// environment variables have to be prefixed with microbus_ and should
// not contain underscores (_). All variables are treated as lowercase
// while the values are case-sensitive.
func loadEnvConfig() (config map[string]interface{}) {
	vars := os.Environ()
	config = make(map[string]interface{})
	for _, envVar := range vars {
		val := os.Getenv(envVar)
		envVar = strings.ToLower(envVar)
		if !strings.HasPrefix(envVar, "microbus") {
			continue
		}

		idx := strings.Index(envVar, "_")
		key := envVar[idx:]
		config[key] = val
	}
	return config
}

func loadConfigFiles(filenames ...string) (config map[string]interface{}, err error) {
	config = make(map[string]interface{})
	for _, filename := range filenames {
		file, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = file.Close()
		}()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "[") {
				// TODO: Section handling
			} else {
				idx := strings.Index(line, "=")
				key := strings.ToLower(strings.TrimSpace(line[:idx]))
				value := strings.TrimSpace(line[idx+1:])
				config[key] = value
			}
		}
	}
	return config, nil
}

// scanDirectory scans a given directory for microbus.env files. The full paths to the files are returned as array
// in the order they are found. Directories are traversed in lexicographical order.
func scanDirectory(directory string, recursive bool) (filenames []string, err error) {
	filenames = make([]string, 0)
	queue := []string{directory}

	for len(queue) > 0 {
		err = filepath.Walk(queue[0], func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if recursive && info.IsDir() && !strings.HasPrefix(path, "../") {
				queue = append(queue, path)
			} else if !info.IsDir() && info.Name() == "microbus.env" {
				filenames = append(filenames, path)
			}
			return nil
		})
		queue = queue[1:]
	}
	return filenames, err
}
