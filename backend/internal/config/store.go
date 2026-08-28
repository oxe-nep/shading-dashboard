package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/oxe-nep/shading-dashboard/internal/model"
)

type Store struct {
	path string
	mu   sync.RWMutex
	cfg  model.AppConfig
}

func NewStore(path string) (*Store, error) {
	s := &Store{path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		s.cfg = defaultConfig()
		return s.persistLocked()
	}
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	var cfg model.AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	if len(cfg.Switches) == 0 {
		cfg = defaultConfig()
	}
	if cfg.PortGroups == nil {
		cfg.PortGroups = []model.PortGroupConfig{}
	}
	if cfg.AllowedVlans == nil {
		cfg.AllowedVlans = []int{}
	}
	s.cfg = cfg
	return nil
}

func defaultConfig() model.AppConfig {
	switches := make([]model.SwitchConfig, 4)
	for i := range switches {
		n := i + 1
		switches[i] = model.SwitchConfig{
			ID:   fmt.Sprintf("sw-%d", n),
			Name: fmt.Sprintf("Switch %d", n),
			Port: 830,
		}
	}
	return model.AppConfig{
		Switches:     switches,
		PortGroups:   []model.PortGroupConfig{},
		AllowedVlans: []int{},
	}
}

func (s *Store) Get() model.AppConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneConfig(s.cfg)
}

func (s *Store) Update(cfg model.AppConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
	if s.cfg.PortGroups == nil {
		s.cfg.PortGroups = []model.PortGroupConfig{}
	}
	if s.cfg.AllowedVlans == nil {
		s.cfg.AllowedVlans = []int{}
	}
	return s.persistLocked()
}

func (s *Store) GetSwitch(id string) (model.SwitchConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sw := range s.cfg.Switches {
		if sw.ID == id {
			return sw, true
		}
	}
	return model.SwitchConfig{}, false
}

func (s *Store) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("mkdir config dir: %w", err)
	}
	data, err := json.MarshalIndent(s.cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func cloneConfig(cfg model.AppConfig) model.AppConfig {
	out := cfg
	out.Switches = append([]model.SwitchConfig(nil), cfg.Switches...)
	out.PortGroups = append([]model.PortGroupConfig{}, cfg.PortGroups...)
	for i := range out.PortGroups {
		out.PortGroups[i].Members = append([]model.PortGroupMember(nil), cfg.PortGroups[i].Members...)
	}
	out.AllowedVlans = append([]int(nil), cfg.AllowedVlans...)
	return out
}
