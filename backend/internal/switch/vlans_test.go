package switchdrv

import (
	"testing"
)

const sampleVLANXML = `<rpc-reply xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><data><native xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-native"><vlan><vlan-list xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-vlan"><id>1029</id><name>VARTAN_CONTROL</name></vlan-list><vlan-list xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-vlan"><id>2022</id><name>RV22-C2IP</name></vlan-list><vlan-list xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-vlan"><id>2120</id></vlan-list></vlan></native></data></rpc-reply>`

func TestParseVLANList(t *testing.T) {
	list := parseVLANList(sampleVLANXML)
	if len(list) != 3 {
		t.Fatalf("expected 3 vlans, got %d", len(list))
	}
	if list[0].ID != 1029 || list[0].Name != "VARTAN_CONTROL" {
		t.Fatalf("unexpected first vlan: %+v", list[0])
	}
	if list[1].Label != "RV22-C2IP" {
		t.Fatalf("unexpected label: %q", list[1].Label)
	}
	if list[2].Name != "" || list[2].Label != "2120" {
		t.Fatalf("unexpected unnamed vlan: %+v", list[2])
	}
}
