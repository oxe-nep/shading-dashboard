package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/coder/websocket"
	"github.com/oxe-nep/shading-dashboard/internal/model"
)

type wsClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (c *wsClient) writeJSON(msg model.WSMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.Write(context.Background(), websocket.MessageText, data)
}

func (c *wsClient) close() {
	_ = c.conn.Close(websocket.StatusNormalClosure, "")
}

type Hub struct {
	mu      sync.RWMutex
	clients map[*wsClient]struct{}
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
		clients: make(map[*wsClient]struct{}),
		manager: manager,
	}
}

func (h *Hub) Broadcast(snapshot model.RuntimeSnapshot) {
	h.broadcast(model.WSMessage{
		Type:            "state-update",
		Switches:        snapshot.Switches,
		PortGroups:      snapshot.PortGroups,
		DiscoveredVlans: snapshot.DiscoveredVlans,
		SelectableVlans: snapshot.SelectableVlans,
	})
}

func (h *Hub) NotifyVlanApplying(switchID, port string, vlan int) {
	h.broadcast(model.WSMessage{
		Type:     "vlan-applying",
		SwitchID: switchID,
		Port:     port,
		VLAN:     vlan,
	})
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

func (h *Hub) NotifyGroupVlanApplying(groupID string, vlan int) {
	h.broadcast(model.WSMessage{
		Type:    "group-vlan-applying",
		GroupID: groupID,
		VLAN:    vlan,
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
	clients := make([]*wsClient, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		client.mu.Lock()
		err := client.conn.Write(context.Background(), websocket.MessageText, data)
		client.mu.Unlock()
		if err != nil {
			client.close()
			h.mu.Lock()
			delete(h.clients, client)
			h.mu.Unlock()
		}
	}
}

func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}

	client := &wsClient{conn: conn}
	h.mu.Lock()
	h.clients[client] = struct{}{}
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, client)
		h.mu.Unlock()
		client.close()
	}()

	snapshot := h.manager.Snapshot()
	_ = client.writeJSON(model.WSMessage{
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
		h.handleMessage(client, data)
	}
}

func (h *Hub) handleMessage(client *wsClient, data []byte) {
	var msg model.WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		_ = client.writeJSON(model.WSMessage{Type: "error", Message: "invalid message"})
		return
	}

	switch msg.Type {
	case "set-vlan":
		if msg.SwitchID == "" || msg.Port == "" || msg.VLAN <= 0 {
			_ = client.writeJSON(model.WSMessage{Type: "error", Message: "switchId, port and vlan required"})
			return
		}
		switchID, port, vlan := msg.SwitchID, msg.Port, msg.VLAN
		h.NotifyVlanApplying(switchID, port, vlan)
		go func() {
			if err := h.manager.SetPortVLAN(switchID, port, vlan); err != nil {
				_ = client.writeJSON(model.WSMessage{Type: "error", Message: err.Error()})
				h.NotifyPortVLANChanged(switchID, port, vlan, false)
				return
			}
			h.NotifyPortVLANChanged(switchID, port, vlan, true)
		}()

	case "set-group-vlan":
		if msg.GroupID == "" || msg.VLAN <= 0 {
			_ = client.writeJSON(model.WSMessage{Type: "error", Message: "groupId and vlan required"})
			return
		}
		groupID, vlan := msg.GroupID, msg.VLAN
		h.NotifyGroupVlanApplying(groupID, vlan)
		go func() {
			results, err := h.manager.SetGroupVLAN(groupID, vlan)
			if err != nil {
				_ = client.writeJSON(model.WSMessage{Type: "error", Message: err.Error()})
				h.NotifyGroupVLANChanged(groupID, vlan, false, nil)
				return
			}
			ok := true
			for _, r := range results {
				if !r.OK {
					ok = false
					break
				}
			}
			h.NotifyGroupVLANChanged(groupID, vlan, ok, results)
		}()

	case "refresh-switch":
		if msg.SwitchID == "" {
			_ = client.writeJSON(model.WSMessage{Type: "error", Message: "switchId required"})
			return
		}
		switchID := msg.SwitchID
		go h.manager.PollSwitchByID(switchID)

	default:
		_ = client.writeJSON(model.WSMessage{Type: "error", Message: "unknown message type"})
	}
}
