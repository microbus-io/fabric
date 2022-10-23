package configurator

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/sub"
)

// Service is a configurator  microservice
type Service struct {
	*connector.Connector
	repo          *repository
	repoTimestamp time.Time
	lock          sync.RWMutex
}

// NewService creates a new configurator microservice
func NewService() *Service {
	s := &Service{
		Connector: connector.New("configurator.sys"),
		repo:      &repository{},
	}
	s.SetDescription("The Configurator is a system microservice that centralizes the dissemination of configuration values to other microservices.")
	s.SetOnStartup(s.OnStartup)
	s.Subscribe("/values", s.Values)
	s.Subscribe("/refresh", s.Refresh)
	s.Subscribe("/sync", s.Sync, sub.NoQueue())
	s.StartTicker("PublishRefresh", 20*time.Minute, s.PublishRefresh)
	// Must not define configs of its own
	return s
}

// OnStartup reads the config values from file.
func (s *Service) OnStartup(ctx context.Context) error {
	// Load values from config.yaml if present in current working directory
	exists := func(fileName string) bool {
		_, err := os.Stat(fileName)
		return err == nil
	}
	if exists("config.yaml") {
		y, err := os.ReadFile("config.yaml")
		if err != nil {
			return errors.Trace(err)
		}
		s.lock.Lock()
		err = s.repo.LoadYAML(y)
		s.repoTimestamp = time.Now()
		s.lock.Unlock()
		if err != nil {
			return errors.Trace(err)
		}
	}

	// Sync the current repo to peers before microservices pull the new config
	err := s.publishSync(ctx)
	if err != nil {
		return errors.Trace(err)
	}

	// Tell all microservices to refresh their config
	err = s.PublishRefresh(ctx)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// Refresh tells all microservices to contact the configurator and refresh their configs.
// An error is returned if any of the values sent to the microservices fails validation.
func (s *Service) Refresh(w http.ResponseWriter, r *http.Request) error {
	err := s.PublishRefresh(r.Context())
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
	return nil
}

// PublishRefresh tells all microservices to contact the configurator and refresh their configs.
// An error is returned if any of the values sent to the microservices fails validation.
func (s *Service) PublishRefresh(ctx context.Context) error {
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

// Sync is used to synchronize values among peers.
func (s *Service) Sync(w http.ResponseWriter, r *http.Request) error {
	// Only respond to peers, and not to self
	if frame.Of(r).FromHost() != s.HostName() || frame.Of(r).FromID() == s.ID() {
		return nil
	}

	// Read input: list of names
	var req struct {
		Timestamp time.Time                    `json:"timestamp"`
		Values    map[string]map[string]string `json:"values"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return errors.Trace(err)
	}

	// Compare incoming and current repos
	localRepo := &repository{
		values: req.Values,
	}
	s.lock.RLock()
	same := localRepo.Equals(s.repo)
	newness := s.repoTimestamp.Sub(req.Timestamp)
	s.lock.RUnlock()

	// If repos are the same, do nothing
	if same {
		return nil
	}

	// If incoming repo is newer, override the current one
	if newness <= 0 {
		s.lock.Lock()
		s.repo = localRepo
		s.repoTimestamp = req.Timestamp
		s.lock.Unlock()
		return nil
	}

	// Sync the current repo to peers
	err = s.publishSync(r.Context())
	if err != nil {
		return errors.Trace(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
	return nil
}

// publishSync syncs the current repo with peers.
func (s *Service) publishSync(ctx context.Context) error {
	// Prep the payload
	s.lock.RLock()
	var req struct {
		Timestamp time.Time                    `json:"timestamp"`
		Values    map[string]map[string]string `json:"values"`
	}
	req.Timestamp = s.repoTimestamp
	req.Values = s.repo.values
	body, err := json.Marshal(req)
	s.lock.RUnlock()
	if err != nil {
		return errors.Trace(err)
	}

	// Broadcast to peers
	ch := s.Publish(
		ctx,
		pub.POST("https://"+s.HostName()+"/sync"),
		pub.Header("Content-Type", "application/json"),
		pub.Body(body))
	for range ch {
		// Ignore results
	}
	return nil
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
	s.lock.RLock()
	for _, name := range req.Names {
		val, ok := s.repo.Value(host, name)
		if ok {
			res.Values[name] = val
		}
	}
	s.lock.RUnlock()

	// Write the response
	body, err := json.Marshal(res)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
	return nil
}

// loadYAML loads a config.yaml into the repo. For testing only.
func (s *Service) loadYAML(configYAML string) error {
	if s.Deployment() == connector.PROD {
		return errors.Newf("disallowed in %s deployment", connector.PROD)
	}
	s.lock.Lock()
	s.repo.LoadYAML([]byte(configYAML))
	s.repoTimestamp = time.Now()
	s.lock.Unlock()
	return nil
}
