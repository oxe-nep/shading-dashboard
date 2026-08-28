package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/oxe-nep/shading-dashboard/internal/model"
)

type fakeManager struct {
	snapshot model.RuntimeSnapshot
}

func (f *fakeManager) Snapshot() model.RuntimeSnapshot { return f.snapshot }
func (f *fakeManager) SetPortVLAN(string, string, int) error { return nil }
func (f *fakeManager) SetGroupVLAN(string, int) ([]model.PortApplyResult, error) {
	return nil, nil
}
func (f *fakeManager) PollSwitchByID(string) {}

func TestHubBroadcastsToAllClients(t *testing.T) {
	hub := NewHub(&fakeManager{snapshot: model.RuntimeSnapshot{}})
	server := httptest.NewServer(http.HandlerFunc(hub.HandleWS))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dial := func() *websocket.Conn {
		t.Helper()
		wsURL := "ws" + server.URL[4:] // http -> ws
		conn, _, err := websocket.Dial(ctx, wsURL, nil)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		return conn
	}

	conn1 := dial()
	defer conn1.Close(websocket.StatusNormalClosure, "")
	conn2 := dial()
	defer conn2.Close(websocket.StatusNormalClosure, "")

	// Drain initial state-update on both connections.
	readType := func(conn *websocket.Conn) string {
		t.Helper()
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var msg model.WSMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		return msg.Type
	}

	if got := readType(conn1); got != "state-update" {
		t.Fatalf("conn1 first msg = %q", got)
	}
	if got := readType(conn2); got != "state-update" {
		t.Fatalf("conn2 first msg = %q", got)
	}

	hub.NotifyPortVLANChanged("sw-3", "Gi1/0/5", 2022, true)

	var wg sync.WaitGroup
	wg.Add(2)
	for i, conn := range []*websocket.Conn{conn1, conn2} {
		conn := conn
		i := i
		go func() {
			defer wg.Done()
			_, data, err := conn.Read(ctx)
			if err != nil {
				t.Errorf("conn%d read: %v", i+1, err)
				return
			}
			var msg model.WSMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				t.Errorf("conn%d unmarshal: %v", i+1, err)
				return
			}
			if msg.Type != "vlan-changed" {
				t.Errorf("conn%d type = %q, want vlan-changed", i+1, msg.Type)
				return
			}
			if msg.SwitchID != "sw-3" || msg.Port != "Gi1/0/5" || msg.VLAN != 2022 {
				t.Errorf("conn%d payload = %+v", i+1, msg)
			}
		}()
	}
	wg.Wait()
}
