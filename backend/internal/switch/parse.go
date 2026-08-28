package switchdrv

import (
	"encoding/xml"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/oxe-nep/shading-dashboard/internal/model"
)

var giBlockRe = regexp.MustCompile(`(?s)<GigabitEthernet[^>]*>(.*?)</GigabitEthernet>`)
var accessVlanRe = regexp.MustCompile(`<vlan>(\d+)</vlan>`)

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
	ports := parsePhysicalPortsFromBlocks(raw, "GigabitEthernet")
	if len(ports) > 0 {
		return ports
	}
	return parseRawPortsXML(raw)
}

func parseRawPortsXML(raw string) []model.PortState {
	var data rpcData
	if err := xml.Unmarshal([]byte(raw), &data); err != nil {
		return nil
	}

	var ports []model.PortState
	ports = append(ports, mapPhysicalPorts("GigabitEthernet", data.Native.Interface.GigabitEthernet)...)
	return ports
}

func parsePhysicalPortsFromBlocks(raw, tag string) []model.PortState {
	matches := giBlockRe.FindAllStringSubmatch(raw, -1)
	if len(matches) == 0 {
		return nil
	}

	byName := make(map[string][]string)
	for _, m := range matches {
		block := m[1]
		name := nativeIfName(extractXMLValue(block, "name"))
		if name == "" {
			continue
		}
		byName[name] = append(byName[name], block)
	}

	ports := make([]model.PortState, 0, len(byName))
	for name, blocks := range byName {
		ports = append(ports, buildPortFromBlocks("GigabitEthernet", name, blocks))
	}
	return ports
}

func buildPortFromBlocks(ifType, ifName string, blocks []string) model.PortState {
	display := displayPortName(ifType, ifName)
	var desc string
	var vlan *int
	adminDown := false

	for _, block := range blocks {
		if d := extractXMLValue(block, "description"); d != "" {
			desc = d
		}
		if strings.Contains(block, "<shutdown") {
			adminDown = true
		}
		if v := parseAccessVLANFromBlock(block); v != nil {
			vlan = v
		}
	}

	oper := "unknown"
	if adminDown {
		oper = "down"
	}

	return model.PortState{
		Name:        display,
		OperState:   oper,
		AccessVLAN:  vlan,
		AdminDown:   adminDown,
		Description: desc,
	}
}

func parseAccessVLANFromBlock(block string) *int {
	swIdx := strings.Index(block, "<switchport>")
	if swIdx < 0 {
		return nil
	}
	section := block[swIdx:]
	if end := strings.Index(section, "</switchport>"); end >= 0 {
		section = section[:end]
	}
	matches := accessVlanRe.FindAllStringSubmatch(section, -1)
	if len(matches) == 0 {
		return nil
	}
	v, err := strconv.Atoi(matches[len(matches)-1][1])
	if err != nil || v <= 0 {
		return nil
	}
	return &v
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
			Name:        displayPortName(ifType, nativeIfName(iface.Name)),
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
