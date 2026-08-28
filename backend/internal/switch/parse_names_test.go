package switchdrv

import "testing"

const sampleFullIfNameXML = `<rpc-reply><data><native xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-native"><interface>
<GigabitEthernet><name>1/0/1</name><description>Short name port 1</description><switchport><access xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-switch"><vlan><vlan>100</vlan></vlan></access></switchport></GigabitEthernet>
<GigabitEthernet><name>GigabitEthernet1/0/12</name><description>Full name port 12</description><switchport><access xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-switch"><vlan><vlan>1012</vlan></vlan></access></switchport></GigabitEthernet>
<GigabitEthernet><name>GigabitEthernet1/0/15</name><description>Full name port 15</description><switchport><access xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-switch"><vlan><vlan>222</vlan></vlan></access></switchport></GigabitEthernet>
<GigabitEthernet><name>1/0/20</name></GigabitEthernet>
</interface></native></data></rpc-reply>`

func TestParsePortsFullInterfaceNames(t *testing.T) {
	ports := parsePortsFromGet(sampleFullIfNameXML)
	byName := map[string]string{}
	for _, p := range ports {
		byName[p.Name] = p.Description
	}

	if byName["Gi1/0/12"] != "Full name port 12" {
		t.Fatalf("Gi1/0/12 description = %q", byName["Gi1/0/12"])
	}
	if byName["Gi1/0/15"] != "Full name port 15" {
		t.Fatalf("Gi1/0/15 description = %q", byName["Gi1/0/15"])
	}

	for _, p := range ports {
		if p.Name == "Gi1/0/20" && p.AccessVLAN != nil {
			t.Fatalf("Gi1/0/20 should have no vlan, got %d", *p.AccessVLAN)
		}
		if p.Name == "Gi1/0/15" && (p.AccessVLAN == nil || *p.AccessVLAN != 222) {
			t.Fatalf("Gi1/0/15 vlan = %v, want 222", p.AccessVLAN)
		}
	}
}

func TestNativeIfName(t *testing.T) {
	cases := map[string]string{
		"1/0/1":                    "1/0/1",
		"GigabitEthernet1/0/12":    "1/0/12",
		"Gi1/0/20":                 "1/0/20",
		"gigabitethernet1/0/3":     "1/0/3",
	}
	for in, want := range cases {
		if got := nativeIfName(in); got != want {
			t.Fatalf("nativeIfName(%q) = %q, want %q", in, got, want)
		}
	}
}
