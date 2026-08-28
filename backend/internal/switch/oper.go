package switchdrv

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/oxe-nep/shading-dashboard/internal/model"
)

const operNS = "http://cisco.com/ns/yang/Cisco-IOS-XE-interfaces-oper"

type operRPCData struct {
	Interfaces operInterfaceList `xml:"data>interfaces"`
}

type operInterfaceList struct {
	Interface []operInterface `xml:"interface"`
}

type operInterface struct {
	Name       string `xml:"name"`
	OperStatus string `xml:"oper-status"`
}

func operInterfaceFilter() string {
	return fmt.Sprintf(`<interfaces xmlns="%s"><interface/></interfaces>`, operNS)
}

func operInterfaceFilterFor(ifType, ifName string) string {
	fullName := operInterfaceName(ifType, ifName)
	return fmt.Sprintf(`<interfaces xmlns="%s"><interface><name>%s</name></interface></interfaces>`, operNS, fullName)
}

func operInterfaceName(ifType, ifName string) string {
	tag := xmlInterfaceTag(ifType)
	ifName = strings.TrimPrefix(strings.TrimSpace(ifName), "/")
	return tag + ifName
}

func parseOperStates(raw string) map[string]string {
	var data operRPCData
	if err := xml.Unmarshal([]byte(raw), &data); err != nil {
		return parseOperStatesFallback(raw)
	}

	out := make(map[string]string, len(data.Interfaces.Interface))
	for _, iface := range data.Interfaces.Interface {
		ifType, ifName := parseInterfaceName(iface.Name)
		display := displayPortName(ifType, ifName)
		out[display] = normalizeOperStatus(iface.OperStatus)
	}
	if len(out) == 0 {
		return parseOperStatesFallback(raw)
	}
	return out
}

func parseOperStatesFallback(raw string) map[string]string {
	type block struct {
		Name string
		Oper string
	}
	var blocks []block
	parts := strings.Split(raw, "<interface>")
	for _, part := range parts[1:] {
		end := strings.Index(part, "</interface>")
		if end < 0 {
			continue
		}
		chunk := part[:end]
		name := extractXMLValue(chunk, "name")
		if name == "" {
			continue
		}
		oper := extractXMLValue(chunk, "oper-status")
		if oper == "" {
			oper = extractXMLValue(chunk, "phy-status")
		}
		ifType, ifName := parseInterfaceName(name)
		display := displayPortName(ifType, ifName)
		blocks = append(blocks, block{Name: display, Oper: normalizeOperStatus(oper)})
	}
	out := make(map[string]string, len(blocks))
	for _, b := range blocks {
		out[b.Name] = b.Oper
	}
	return out
}

func extractXMLValue(chunk, tag string) string {
	open := "<" + tag + ">"
	close := "</" + tag + ">"
	start := strings.Index(chunk, open)
	if start < 0 {
		// handle xmlns on tag: <oper-status ...>
		marker := "<" + tag
		start = strings.Index(chunk, marker)
		if start < 0 {
			return ""
		}
		start = strings.Index(chunk[start:], ">")
		if start < 0 {
			return ""
		}
		start += strings.Index(chunk, marker) + start + 1
	} else {
		start += len(open)
	}
	end := strings.Index(chunk[start:], close)
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(chunk[start : start+end])
}

func normalizeOperStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	switch {
	case strings.Contains(status, "ready"),
		strings.Contains(status, "up"),
		status == "if-oper-up":
		return "up"
	case strings.Contains(status, "down"),
		strings.Contains(status, "not-present"),
		strings.Contains(status, "lower-layer"):
		return "down"
	default:
		return "unknown"
	}
}

func applyOperStates(ports []model.PortState, oper map[string]string) {
	if len(oper) == 0 {
		return
	}
	for i := range ports {
		if ports[i].AdminDown {
			ports[i].OperState = "down"
			continue
		}
		if state, ok := oper[ports[i].Name]; ok {
			ports[i].OperState = state
		}
	}
}
