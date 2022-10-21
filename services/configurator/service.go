package configurator

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/pub"
	"gopkg.in/yaml.v2"
)

// Service is a configurator  microservice
type Service struct {
	*connector.Connector
	repo     *repository
	repoLock sync.Mutex

	mockedConfigYAML       string
	mockedConfigImportYAML string
	mockedURLs             map[string]string
}

// NewService creates a new configurator microservice
func NewService() *Service {
	s := &Service{
		Connector: connector.New("configurator.sys"),
	}
	s.SetDescription("The Configurator is a system microservice that centralizes the dissemination of configuration values to other microservices.")
	s.SetOnStartup(s.OnStartup)
	s.Subscribe("/values", s.Values)
	s.StartTicker("FetchValues", 20*time.Minute, s.fetchValues)
	// Must not define configs of its own
	return s
}

// OnStartup reads the config values from file.
func (s *Service) OnStartup(ctx context.Context) error {
	err := s.fetchValues(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// ForceRefresh tells all microservices to contact the configurator and refresh their configs.
// An error is returned if any of the values sent to the microservices fails validation.
func (s *Service) ForceRefresh(w http.ResponseWriter, r *http.Request) error {
	err := s.forceRefresh(r.Context())
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
	return nil
}

// forceRefresh tells all microservices to contact the configurator and refresh their configs.
// An error is returned if any of the values sent to the microservices fails validation.
func (s *Service) forceRefresh(ctx context.Context) error {
	var lastErr error
	ch := s.Publish(ctx, pub.GET("https://all:888/config/refresh"))
	for i := range ch {
		_, err := i.Get()
		if err != nil && err.Error() != "ack timeout" {
			lastErr = errors.Trace(err)
			s.LogError(ctx, "Updating config", log.Error(lastErr))
		}
	}
	return lastErr
}

// Values returns the values associated with the specified config property names
// for the caller microservice.
func (s *Service) Values(w http.ResponseWriter, r *http.Request) error {
	host := frame.Of(r).FromHost()

	// Read input: list of names
	var req struct {
		Names []string `json:"names"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return errors.Trace(err)
	}

	// Prepare output: map of values
	var res struct {
		Values map[string]string `json:"values"`
	}
	res.Values = map[string]string{}

	// Load the values of the requested properties from the repo
	s.repoLock.Lock()
	if s.repo != nil {
		for _, name := range req.Names {
			val, ok := s.repo.Value(host, name)
			if ok {
				res.Values[name] = val
			}
		}
	}
	s.repoLock.Unlock()

	// Write the response
	body, err := json.Marshal(res)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
	return nil
}

// fetchValues loads the values from the local and remote config.yaml files
// then informs all microservices to refresh their config.
func (s *Service) fetchValues(ctx context.Context) error {
	exists := func(fileName string) bool {
		_, err := os.Stat(fileName)
		return err == nil
	}

	var localRepo repository

	// Download remote config.yaml files
	var err error
	var y []byte
	if s.mockedConfigImportYAML != "" {
		y = []byte(s.mockedConfigImportYAML)
	} else if exists("configimport.yaml") {
		y, err = os.ReadFile("configimport.yaml")
		if err != nil {
			return errors.Trace(err)
		}
	}
	if len(y) > 0 {
		var remotes []*struct {
			From   string `json:"from"`
			Import string `json:"import"`
		}
		err = yaml.Unmarshal(y, &remotes)
		if err != nil {
			return errors.Trace(err)
		}
		for _, remote := range remotes {
			var y []byte
			if s.mockedURLs != nil && s.mockedURLs[remote.From] != "" {
				y = []byte(s.mockedURLs[remote.From])
			} else {
				client := http.Client{
					Timeout: 4 * time.Second,
				}
				response, err := client.Get(remote.From)
				if err != nil {
					return errors.Trace(err)
				}
				y, err = io.ReadAll(response.Body)
				if err != nil {
					return errors.Trace(err)
				}
			}
			scopes := strings.Split(remote.Import, ",")
			for _, scope := range scopes {
				err = localRepo.LoadYAML(y, strings.TrimSpace(scope))
				if err != nil {
					return errors.Trace(err)
				}
			}
		}
	}

	// Look for explicit values in local config.yaml file
	y = nil
	if s.mockedConfigYAML != "" {
		y = []byte(s.mockedConfigYAML)
	} else if exists("config.yaml") {
		y, err = os.ReadFile("config.yaml")
		if err != nil {
			return errors.Trace(err)
		}
	}
	if len(y) > 0 {
		err = localRepo.LoadYAML(y, "")
		if err != nil {
			return errors.Trace(err)
		}
	}

	// Replace the old repo with the new repo
	s.repoLock.Lock()
	s.repo = &localRepo
	s.repoLock.Unlock()

	// Tell all microservices to refresh their configs even if there was no change in values
	err = s.forceRefresh(ctx)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// mockConfigYAML sets a mock value for the config.yaml file. For testing only.
func (s *Service) mockConfigYAML(configYAML string) error {
	if s.Deployment() == connector.PROD {
		return errors.Newf("disallowed in %s deployment", connector.PROD)
	}
	s.mockedConfigYAML = configYAML
	return nil
}

// mockConfigImportYAML sets a mock value for the configimport.yaml file. For testing only.
func (s *Service) mockConfigImportYAML(configImportYAML string) error {
	if s.Deployment() == connector.PROD {
		return errors.Newf("disallowed in %s deployment", connector.PROD)
	}
	s.mockedConfigImportYAML = configImportYAML
	return nil
}

// mockRemoteConfigYAML sets a mock config.yaml for a URL. For testing only.
func (s *Service) mockRemoteConfigYAML(url string, remoteYAML string) error {
	if s.Deployment() == connector.PROD {
		return errors.Newf("disallowed in %s deployment", connector.PROD)
	}
	if s.mockedURLs == nil {
		s.mockedURLs = map[string]string{}
	}
	s.mockedURLs[url] = remoteYAML
	return nil
}
