package switchdrv

import "testing"

const sampleOperXML = `<rpc-reply><data><interfaces xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-interfaces-oper">
<interface><name>GigabitEthernet1/0/1</name><oper-status>if-oper-state-ready</oper-status></interface>
<interface><name>GigabitEthernet1/0/2</name><oper-status>if-oper-state-down</oper-status></interface>
</interfaces></data></rpc-reply>`

func TestParseOperStates(t *testing.T) {
	states := parseOperStates(sampleOperXML)
	if states["Gi1/0/1"] != "up" {
		t.Fatalf("Gi1/0/1: got %q", states["Gi1/0/1"])
	}
	if states["Gi1/0/2"] != "down" {
		t.Fatalf("Gi1/0/2: got %q", states["Gi1/0/2"])
	}
}
