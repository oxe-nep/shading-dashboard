package switchdrv

import (
	"encoding/xml"
	"net/url"
	"strings"

	"github.com/oxe-nep/shading-dashboard/internal/model"
)

type rpcData struct {
	Native nativeData `xml:"data>native"`
}

type nativeData struct {
	Interface interfaceData `xml:"interface"`
	VLAN      vlanData      `xml:"vlan"`
}

type interfaceData struct {
	GigabitEthernet    []physicalInterface `xml:"GigabitEthernet"`
	TenGigabitEthernet []physicalInterface `xml:"TenGigabitEthernet"`
}

type physicalInterface struct {
	Name        string     `xml:"name"`
	Shutdown    *struct{}  `xml:"shutdown"`
	Description string     `xml:"description"`
	Switchport  switchport `xml:"switchport"`
}

type switchport struct {
	Access accessBlock `xml:"access"`
}

type accessBlock struct {
	VLAN nestedVLAN `xml:"vlan"`
}

type nestedVLAN struct {
	VLAN int `xml:"vlan"`
}

type vlanData struct {
	VLANList []vlanEntry `xml:"vlan-list"`
}

type vlanEntry struct {
	ID   int    `xml:"id"`
	Name string `xml:"name"`
}

func parsePortsFromGet(raw string) []model.PortState {
	return finalizeMonitoredPorts(parseRawPorts(raw))
}

func parseRawPorts(raw string) []model.PortState {
	var data rpcData
	if err := xml.Unmarshal([]byte(raw), &data); err != nil {
		return nil
	}

	var ports []model.PortState
	ports = append(ports, mapPhysicalPorts("GigabitEthernet", data.Native.Interface.GigabitEthernet)...)
	return ports
}

func mapPhysicalPorts(ifType string, ifs []physicalInterface) []model.PortState {
	out := make([]model.PortState, 0, len(ifs))
	for _, iface := range ifs {
		adminDown := iface.Shutdown != nil
		oper := "unknown"
		if adminDown {
			oper = "down"
		}

		var vlanPtr *int
		if iface.Switchport.Access.VLAN.VLAN > 0 {
			v := iface.Switchport.Access.VLAN.VLAN
			vlanPtr = &v
		}

		out = append(out, model.PortState{
			Name:        displayPortName(ifType, iface.Name),
			OperState:   oper,
			AccessVLAN:  vlanPtr,
			AdminDown:   adminDown,
			Description: strings.TrimSpace(iface.Description),
		})
	}
	return out
}

func parseVLANIDs(raw string) map[int]struct{} {
	var data rpcData
	if err := xml.Unmarshal([]byte(raw), &data); err != nil {
		return nil
	}
	out := make(map[int]struct{})
	for _, v := range data.Native.VLAN.VLANList {
		if v.ID > 0 {
			out[v.ID] = struct{}{}
		}
	}
	return out
}

func NormalizePortQuery(port string) (ifType, ifName, display string) {
	if decoded, err := url.PathUnescape(strings.TrimSpace(port)); err == nil {
		port = decoded
	}
	ifType, ifName = parseInterfaceName(port)
	display = displayPortName(ifType, ifName)
	return ifType, ifName, display
}
