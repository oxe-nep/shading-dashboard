package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/oxe-nep/shading-dashboard/internal/config"
	"github.com/oxe-nep/shading-dashboard/internal/groups"
	"github.com/oxe-nep/shading-dashboard/internal/model"
	switchdrv "github.com/oxe-nep/shading-dashboard/internal/switch"
)

type Server struct {
	store   *config.Store
	manager *switchdrv.Manager
	hub     *Hub
}

func NewServer(store *config.Store, manager *switchdrv.Manager, hub *Hub) *Server {
	return &Server{store: store, manager: manager, hub: hub}
}

func (s *Server) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/health", s.handleHealth)
	r.Get("/ws", s.hub.HandleWS)

	r.Route("/api", func(r chi.Router) {
		r.Get("/config", s.handleGetConfig)
		r.Put("/config", s.handlePutConfig)
		r.Put("/port-groups", s.handlePutPortGroups)
		r.Put("/port-groups/{id}/vlan", s.handleSetGroupVLAN)
		r.Put("/switches/{id}/ports/{port}/vlan", s.handleSetPortVLAN)
	})

	return r
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"switches": len(s.manager.GetStates()),
	})
}

func (s *Server) handleGetConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Get())
}

func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var cfg model.AppConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid config body")
		return
	}
	for _, sw := range cfg.Switches {
		if sw.Name == "" {
			writeError(w, http.StatusBadRequest, "each switch must have a name")
			return
		}
	}
	if err := groups.ValidatePortGroups(cfg.PortGroups); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	oldCfg := s.store.Get()
	if err := s.store.Update(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.applyConfigUpdate(switchesConfigChanged(oldCfg, cfg))
	writeJSON(w, http.StatusOK, s.store.Get())
}

func (s *Server) handlePutPortGroups(w http.ResponseWriter, r *http.Request) {
	var portGroups []model.PortGroupConfig
	if err := json.NewDecoder(r.Body).Decode(&portGroups); err != nil {
		writeError(w, http.StatusBadRequest, "invalid port groups body")
		return
	}
	if err := groups.ValidatePortGroups(portGroups); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if portGroups == nil {
		portGroups = []model.PortGroupConfig{}
	}
	cfg := s.store.Get()
	cfg.PortGroups = portGroups
	if err := s.store.Update(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.applyConfigUpdate(false)
	writeJSON(w, http.StatusOK, s.store.Get())
}

func (s *Server) applyConfigUpdate(refreshSwitches bool) {
	s.manager.SyncFromConfig()
	s.hub.Broadcast(s.manager.Snapshot())
	if refreshSwitches {
		go s.manager.RefreshAll()
	}
}

func switchesConfigChanged(before, after model.AppConfig) bool {
	if len(before.Switches) != len(after.Switches) {
		return true
	}
	for i := range before.Switches {
		a, b := before.Switches[i], after.Switches[i]
		if a.ID != b.ID || a.Name != b.Name || a.IP != b.IP ||
			a.Username != b.Username || a.Password != b.Password || a.Port != b.Port {
			return true
		}
	}
	return false
}

func (s *Server) handleSetPortVLAN(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	port := chi.URLParam(r, "port")
	var body model.SetVLANRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.VLAN <= 0 {
		writeError(w, http.StatusBadRequest, "vlan required")
		return
	}
	if err := s.manager.SetPortVLAN(id, port, body.VLAN); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.hub.NotifyPortVLANChanged(id, port, body.VLAN, true)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleSetGroupVLAN(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body model.SetVLANRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.VLAN <= 0 {
		writeError(w, http.StatusBadRequest, "vlan required")
		return
	}
	results, err := s.manager.SetGroupVLAN(id, body.VLAN)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ok := true
	for _, r := range results {
		if !r.OK {
			ok = false
			break
		}
	}
	s.hub.NotifyGroupVLANChanged(id, body.VLAN, ok, results)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "results": results})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
