package switchdrv

import "testing"

const samplePortsXML = `<rpc-reply xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><data><native xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-native"><interface><GigabitEthernet><name>1/0/1</name><description>Camera A</description><switchport><access xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-switch"><vlan><vlan>2022</vlan></vlan></access></switchport></GigabitEthernet><GigabitEthernet><name>1/0/2</name><switchport><access xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-switch"><vlan><vlan>2023</vlan></vlan></access></switchport></GigabitEthernet></interface></native></data></rpc-reply>`

func TestParsePortsDescription(t *testing.T) {
	ports := parsePortsFromGet(samplePortsXML)
	byName := make(map[string]string, len(ports))
	for _, p := range ports {
		byName[p.Name] = p.Description
	}

	if got := byName["Gi1/0/1"]; got != "Camera A" {
		t.Fatalf("Gi1/0/1 description = %q, want Camera A", got)
	}
	if got := byName["Gi1/0/2"]; got != "" {
		t.Fatalf("Gi1/0/2 description = %q, want empty", got)
	}
}
