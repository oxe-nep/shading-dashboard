package switchdrv

import (
	"testing"

	"github.com/oxe-nep/shading-dashboard/internal/model"
)

const samplePortsXML = `<rpc-reply xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><data><native xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-native"><interface><GigabitEthernet><name>1/0/1</name><description>Camera A</description><switchport><access xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-switch"><vlan><vlan>2022</vlan></vlan></access></switchport></GigabitEthernet><GigabitEthernet><name>1/0/2</name><switchport><access xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-switch"><vlan><vlan>2023</vlan></vlan></access></switchport></GigabitEthernet></interface></native></data></rpc-reply>`

func TestParsePortsDescription(t *testing.T) {
	ports := parsePortsFromGet(samplePortsXML)
	byName := make(map[string]model.PortState, len(ports))
	for _, p := range ports {
		byName[p.Name] = p
	}

	p1 := byName["Gi1/0/1"]
	if p1.Description != "Camera A" {
		t.Fatalf("Gi1/0/1 description = %q, want Camera A", p1.Description)
	}
	if p1.AccessVLAN == nil || *p1.AccessVLAN != 2022 {
		t.Fatalf("Gi1/0/1 vlan = %v, want 2022", p1.AccessVLAN)
	}

	p2 := byName["Gi1/0/2"]
	if p2.Description != "" {
		t.Fatalf("Gi1/0/2 description = %q, want empty", p2.Description)
	}
	if p2.AccessVLAN == nil || *p2.AccessVLAN != 2023 {
		t.Fatalf("Gi1/0/2 vlan = %v, want 2023", p2.AccessVLAN)
	}
}
