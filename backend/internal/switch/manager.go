package switchdrv

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/oxe-nep/shading-dashboard/internal/config"
	"github.com/oxe-nep/shading-dashboard/internal/groups"
	"github.com/oxe-nep/shading-dashboard/internal/model"
	"github.com/rs/zerolog/log"
)

const defaultOfflineAfter = 3

type Manager struct {
	store        *config.Store
	mu           sync.RWMutex
	switchMu     map[string]*sync.Mutex
	states       map[string]*model.SwitchRuntimeState
	clients      map[string]*Client
	failures     map[string]int
	offlineAfter int
	onUpdate     func(model.RuntimeSnapshot)
}

func NewManager(store *config.Store) *Manager {
	return &Manager{
		store:        store,
		switchMu:     make(map[string]*sync.Mutex),
		states:       make(map[string]*model.SwitchRuntimeState),
		clients:      make(map[string]*Client),
		failures:     make(map[string]int),
		offlineAfter: defaultOfflineAfter,
	}
}

func (m *Manager) SetUpdateHandler(fn func(model.RuntimeSnapshot)) {
	m.onUpdate = fn
}

func (m *Manager) lockSwitch(id string) {
	m.mu.Lock()
	mu, ok := m.switchMu[id]
	if !ok {
		mu = &sync.Mutex{}
		m.switchMu[id] = mu
	}
	m.mu.Unlock()
	mu.Lock()
}

func (m *Manager) unlockSwitch(id string) {
	m.mu.RLock()
	mu := m.switchMu[id]
	m.mu.RUnlock()
	if mu != nil {
		mu.Unlock()
	}
}

func (m *Manager) Snapshot() model.RuntimeSnapshot {
	cfg := m.store.Get()

	m.mu.RLock()
	defer m.mu.RUnlock()

	states := make([]model.SwitchRuntimeState, 0, len(cfg.Switches))
	for _, swCfg := range cfg.Switches {
		if st, ok := m.states[swCfg.ID]; ok {
			states = append(states, *st)
		}
	}

	discovered := mergeDiscoveredVLANs(states)
	return model.RuntimeSnapshot{
		Switches:        states,
		PortGroups:      groups.BuildRuntimeGroups(cfg.PortGroups, states),
		DiscoveredVlans: discovered,
		SelectableVlans: filterSelectableVLANs(discovered, cfg.AllowedVlans),
	}
}

func (m *Manager) GetStates() []model.SwitchRuntimeState {
	return m.Snapshot().Switches
}

func (m *Manager) SyncFromConfig() {
	cfg := m.store.Get()
	m.mu.Lock()
	defer m.mu.Unlock()

	known := make(map[string]struct{}, len(cfg.Switches))
	for _, sw := range cfg.Switches {
		known[sw.ID] = struct{}{}
		if _, ok := m.states[sw.ID]; !ok {
			m.states[sw.ID] = &model.SwitchRuntimeState{
				ID:    sw.ID,
				Name:  sw.Name,
				IP:    sw.IP,
				Ports: []model.PortState{},
			}
		} else {
			m.states[sw.ID].Name = sw.Name
			m.states[sw.ID].IP = sw.IP
		}
	}

	for id, client := range m.clients {
		if _, ok := known[id]; !ok {
			client.Close()
			delete(m.clients, id)
		}
	}

	for id := range m.states {
		if _, ok := known[id]; !ok {
			delete(m.states, id)
			delete(m.failures, id)
			delete(m.switchMu, id)
		}
	}
}

func (m *Manager) closeAllClients() {
	m.mu.Lock()
	for _, c := range m.clients {
		c.Close()
	}
	m.clients = make(map[string]*Client)
	m.mu.Unlock()
}

// RefreshAll re-polls every configured switch. Used at startup and after switch config changes.
func (m *Manager) RefreshAll() {
	log.Info().Msg("refreshing all switches")
	m.closeAllClients()
	m.PollAll()
}

func (m *Manager) PollSwitchByID(id string) {
	sw, ok := m.store.GetSwitch(id)
	if !ok {
		log.Warn().Str("switch", id).Msg("poll skipped: switch not found")
		return
	}
	log.Info().Str("switch", id).Msg("manual poll started")
	m.pollSwitch(sw)
}

func (m *Manager) PollAll() {
	cfg := m.store.Get()
	log.Info().Int("switches", len(cfg.Switches)).Msg("poll all started")
	var wg sync.WaitGroup
	for _, sw := range cfg.Switches {
		wg.Add(1)
		go func(sw model.SwitchConfig) {
			defer wg.Done()
			m.pollSwitch(sw)
		}(sw)
	}
	wg.Wait()
	m.emitUpdate()
	log.Info().Msg("poll all finished")
}

func (m *Manager) pollSwitch(sw model.SwitchConfig) {
	if sw.IP == "" || sw.Username == "" {
		m.markFailure(sw, "switch not configured")
		return
	}

	m.setPolling(sw.ID, true)
	m.emitUpdate()

	defer func() {
		m.setPolling(sw.ID, false)
		m.emitUpdate()
	}()

	waitStart := time.Now()
	m.lockSwitch(sw.ID)
	defer m.unlockSwitch(sw.ID)

	if waited := time.Since(waitStart); waited > 100*time.Millisecond {
		log.Info().Str("switch", sw.ID).Dur("wait", waited).Msg("poll waited for switch lock")
	}

	log.Info().Str("switch", sw.ID).Str("ip", sw.IP).Msg("polling switch")

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	client := m.getOrCreateClient(sw)
	ports, vlans, err := client.GetPorts(ctx)
	if err != nil {
		log.Warn().Err(err).Str("switch", sw.ID).Msg("poll failed")
		m.markFailure(sw, err.Error())
		return
	}

	now := time.Now().UTC()
	m.mu.Lock()
	st := m.states[sw.ID]
	st.Online = true
	st.LastPollAt = &now
	st.LastSuccessAt = &now
	st.Ports = ports
	st.Vlans = vlans
	st.LastError = ""
	m.failures[sw.ID] = 0
	m.mu.Unlock()

	log.Info().
		Str("switch", sw.ID).
		Int("ports", len(ports)).
		Int("vlans", len(vlans)).
		Msg("poll ok")

	m.emitUpdate()
}

func (m *Manager) verifyPortsLocked(sw model.SwitchConfig, expected map[string]int) error {
	if len(expected) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := m.getOrCreateClient(sw)

	m.mu.RLock()
	st := m.states[sw.ID]
	vlanNames := vlanNamesMap(st.Vlans)
	m.mu.RUnlock()

	ports := make([]model.PortState, 0, len(expected))
	for port, want := range expected {
		_, _, display := NormalizePortQuery(port)
		p, err := client.GetPort(ctx, display)
		if err != nil {
			return fmt.Errorf("verify %s: %w", display, err)
		}
		labeled := []model.PortState{p}
		applyVLANLabels(labeled, vlanNames)
		p = labeled[0]
		if p.AccessVLAN == nil || *p.AccessVLAN != want {
			got := "none"
			if p.AccessVLAN != nil {
				got = fmt.Sprintf("%d", *p.AccessVLAN)
			}
			return fmt.Errorf("%s VLAN is %s, expected %d", display, got, want)
		}
		ports = append(ports, p)
	}

	m.mu.Lock()
	st = m.states[sw.ID]
	for _, p := range ports {
		for i := range st.Ports {
			if st.Ports[i].Name == p.Name {
				if p.Description == "" {
					p.Description = st.Ports[i].Description
				}
				st.Ports[i] = p
				break
			}
		}
	}
	now := time.Now().UTC()
	st.LastPollAt = &now
	st.LastSuccessAt = &now
	st.LastError = ""
	m.mu.Unlock()

	m.emitUpdate()
	return nil
}

func (m *Manager) markFailure(sw model.SwitchConfig, msg string) {
	now := time.Now().UTC()
	m.mu.Lock()
	st := m.states[sw.ID]
	st.LastPollAt = &now
	st.LastError = msg
	m.failures[sw.ID]++
	if m.failures[sw.ID] >= m.offlineAfter {
		st.Online = false
	}
	m.mu.Unlock()
	m.emitUpdate()
}

func (m *Manager) setPolling(id string, polling bool) {
	m.mu.Lock()
	if st, ok := m.states[id]; ok {
		st.Polling = polling
	}
	m.mu.Unlock()
}

func (m *Manager) getOrCreateClient(sw model.SwitchConfig) *Client {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.clients[sw.ID]; ok {
		return c
	}
	c := NewClient(sw)
	m.clients[sw.ID] = c
	return c
}

func (m *Manager) SetPortVLAN(switchID, port string, vlan int) error {
	sw, ok := m.store.GetSwitch(switchID)
	if !ok {
		return ErrSwitchNotFound
	}

	_, _, display := NormalizePortQuery(port)

	m.lockSwitch(sw.ID)
	defer m.unlockSwitch(sw.ID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := m.getOrCreateClient(sw)
	if err := client.SetAccessVLAN(ctx, display, vlan); err != nil {
		return err
	}

	return m.verifyPortsLocked(sw, map[string]int{display: vlan})
}

type groupMember struct {
	switchID string
	port     string
}

func (m *Manager) SetGroupVLAN(groupID string, vlan int) ([]model.PortApplyResult, error) {
	cfg := m.store.Get()
	var group *model.PortGroupConfig
	for i := range cfg.PortGroups {
		if cfg.PortGroups[i].ID == groupID {
			group = &cfg.PortGroups[i]
			break
		}
	}
	if group == nil {
		return nil, ErrGroupNotFound
	}

	bySwitch := make(map[string][]groupMember)
	for _, mem := range group.Members {
		bySwitch[mem.SwitchID] = append(bySwitch[mem.SwitchID], groupMember{
			switchID: mem.SwitchID,
			port:     mem.Port,
		})
	}

	resultsCh := make(chan []model.PortApplyResult, len(bySwitch))
	var wg sync.WaitGroup

	for switchID, members := range bySwitch {
		wg.Add(1)
		go func(switchID string, members []groupMember) {
			defer wg.Done()
			resultsCh <- m.applyVLANsOnSwitch(switchID, members, vlan)
		}(switchID, members)
	}

	wg.Wait()
	close(resultsCh)

	var all []model.PortApplyResult
	for batch := range resultsCh {
		all = append(all, batch...)
	}

	return all, nil
}

func (m *Manager) applyVLANsOnSwitch(switchID string, members []groupMember, vlan int) []model.PortApplyResult {
	sw, ok := m.store.GetSwitch(switchID)
	if !ok {
		out := make([]model.PortApplyResult, 0, len(members))
		for _, mem := range members {
			out = append(out, model.PortApplyResult{
				SwitchID: switchID,
				Port:     mem.port,
				OK:       false,
				Error:    "switch not found",
			})
		}
		return out
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	m.lockSwitch(switchID)
	defer m.unlockSwitch(switchID)

	client := m.getOrCreateClient(sw)
	out := make([]model.PortApplyResult, 0, len(members))
	expected := make(map[string]int, len(members))

	for _, mem := range members {
		res := model.PortApplyResult{SwitchID: switchID, Port: mem.port}
		_, _, display := NormalizePortQuery(mem.port)
		if err := client.SetAccessVLAN(ctx, display, vlan); err != nil {
			res.OK = false
			res.Error = err.Error()
		} else {
			res.OK = true
			expected[display] = vlan
		}
		out = append(out, res)
	}

	if len(expected) > 0 {
		if err := m.verifyPortsLocked(sw, expected); err != nil {
			for i := range out {
				if out[i].OK {
					out[i].OK = false
					out[i].Error = err.Error()
				}
			}
		}
	}

	return out
}

func (m *Manager) emitUpdate() {
	if m.onUpdate == nil {
		return
	}
	m.onUpdate(m.Snapshot())
}

var (
	ErrSwitchNotFound = errSentinel("switch not found")
	ErrGroupNotFound  = errSentinel("port group not found")
)

type errSentinel string

func (e errSentinel) Error() string { return string(e) }

func (m *Manager) Start() {
	m.SyncFromConfig()
	go m.PollAll()
}
