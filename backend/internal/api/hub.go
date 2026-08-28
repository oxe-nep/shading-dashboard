package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/coder/websocket"
	"github.com/oxe-nep/shading-dashboard/internal/model"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]struct{}
	manager StateProvider
}

type StateProvider interface {
	Snapshot() model.RuntimeSnapshot
	SetPortVLAN(switchID, port string, vlan int) error
	SetGroupVLAN(groupID string, vlan int) ([]model.PortApplyResult, error)
	PollSwitchByID(id string)
}

func NewHub(manager StateProvider) *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]struct{}),
		manager: manager,
	}
}

func (h *Hub) Broadcast(snapshot model.RuntimeSnapshot) {
	msg := model.WSMessage{
		Type:            "state-update",
		Switches:        snapshot.Switches,
		PortGroups:      snapshot.PortGroups,
		DiscoveredVlans: snapshot.DiscoveredVlans,
		SelectableVlans: snapshot.SelectableVlans,
	}
	h.broadcast(msg)
}

func (h *Hub) NotifyPortVLANChanged(switchID, port string, vlan int, ok bool) {
	h.broadcast(model.WSMessage{
		Type:     "vlan-changed",
		SwitchID: switchID,
		Port:     port,
		VLAN:     vlan,
		OK:       ok,
	})
}

func (h *Hub) NotifyGroupVLANChanged(groupID string, vlan int, ok bool, results []model.PortApplyResult) {
	h.broadcast(model.WSMessage{
		Type:    "group-vlan-changed",
		GroupID: groupID,
		VLAN:    vlan,
		OK:      ok,
		Results: results,
	})
}

func (h *Hub) broadcast(msg model.WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	conns := make([]*websocket.Conn, 0, len(h.clients))
	for conn := range h.clients {
		conns = append(conns, conn)
	}
	h.mu.RUnlock()

	for _, conn := range conns {
		if err := conn.Write(context.Background(), websocket.MessageText, data); err != nil {
			_ = conn.Close(websocket.StatusInternalError, "write failed")
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
		}
	}
}

func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}

	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}()

	snapshot := h.manager.Snapshot()
	_ = h.write(conn, model.WSMessage{
		Type:            "state-update",
		Switches:        snapshot.Switches,
		PortGroups:      snapshot.PortGroups,
		DiscoveredVlans: snapshot.DiscoveredVlans,
		SelectableVlans: snapshot.SelectableVlans,
	})

	for {
		_, data, err := conn.Read(r.Context())
		if err != nil {
			return
		}
		h.handleMessage(conn, data)
	}
}

func (h *Hub) handleMessage(conn *websocket.Conn, data []byte) {
	var msg model.WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		_ = h.write(conn, model.WSMessage{Type: "error", Message: "invalid message"})
		return
	}

	switch msg.Type {
	case "set-vlan":
		if msg.SwitchID == "" || msg.Port == "" || msg.VLAN <= 0 {
			_ = h.write(conn, model.WSMessage{Type: "error", Message: "switchId, port and vlan required"})
			return
		}
		if err := h.manager.SetPortVLAN(msg.SwitchID, msg.Port, msg.VLAN); err != nil {
			_ = h.write(conn, model.WSMessage{Type: "error", Message: err.Error()})
			return
		}
		h.NotifyPortVLANChanged(msg.SwitchID, msg.Port, msg.VLAN, true)

	case "set-group-vlan":
		if msg.GroupID == "" || msg.VLAN <= 0 {
			_ = h.write(conn, model.WSMessage{Type: "error", Message: "groupId and vlan required"})
			return
		}
		results, err := h.manager.SetGroupVLAN(msg.GroupID, msg.VLAN)
		if err != nil {
			_ = h.write(conn, model.WSMessage{Type: "error", Message: err.Error()})
			return
		}
		ok := true
		for _, r := range results {
			if !r.OK {
				ok = false
				break
			}
		}
		h.NotifyGroupVLANChanged(msg.GroupID, msg.VLAN, ok, results)

	case "refresh-switch":
		if msg.SwitchID == "" {
			_ = h.write(conn, model.WSMessage{Type: "error", Message: "switchId required"})
			return
		}
		h.manager.PollSwitchByID(msg.SwitchID)

	default:
		_ = h.write(conn, model.WSMessage{Type: "error", Message: "unknown message type"})
	}
}

func (h *Hub) write(conn *websocket.Conn, msg model.WSMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return conn.Write(context.Background(), websocket.MessageText, data)
}
