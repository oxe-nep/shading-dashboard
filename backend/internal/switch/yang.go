package switchdrv

import (
	"fmt"
	"strings"
)

const nativeNS = "http://cisco.com/ns/yang/Cisco-IOS-XE-native"

func interfaceFilter() string {
	return fmt.Sprintf(`<native xmlns="%s">
    <interface>
      <GigabitEthernet/>
    </interface>
  </native>`, nativeNS)
}

func vlanListFilter() string {
	return fmt.Sprintf(`<native xmlns="%s">
    <vlan/>
  </native>`, nativeNS)
}

func editAccessVLANConfig(ifType, ifName string, vlan int) string {
	tag := xmlInterfaceTag(ifType)
	switchNS := "http://cisco.com/ns/yang/Cisco-IOS-XE-switch"
	return fmt.Sprintf(`<config>
  <native xmlns="%s">
    <interface>
      <%s>
        <name>%s</name>
        <switchport>
          <access xmlns="%s">
            <vlan>
              <vlan>%d</vlan>
            </vlan>
          </access>
          <mode xmlns="%s">
            <access/>
          </mode>
        </switchport>
      </%s>
    </interface>
  </native>
</config>`, nativeNS, tag, ifName, switchNS, vlan, switchNS, tag)
}

func interfaceFilterFor(ifType, ifName string) string {
	tag := xmlInterfaceTag(ifType)
	return fmt.Sprintf(`<native xmlns="%s">
    <interface>
      <%s>
        <name>%s</name>
      </%s>
    </interface>
  </native>`, nativeNS, tag, ifName, tag)
}

func xmlInterfaceTag(ifType string) string {
	switch strings.ToLower(ifType) {
	case "tengigabitethernet", "te":
		return "TenGigabitEthernet"
	default:
		return "GigabitEthernet"
	}
}

func parseInterfaceName(port string) (ifType, ifName string) {
	port = strings.TrimSpace(port)
	lower := strings.ToLower(port)

	switch {
	case strings.HasPrefix(lower, "tengigabitethernet"):
		return "TenGigabitEthernet", strings.TrimPrefix(port, "TenGigabitEthernet")
	case strings.HasPrefix(lower, "te"):
		rest := port[2:]
		if strings.HasPrefix(rest, "ngabitEthernet") {
			rest = strings.TrimPrefix(rest, "ngabitEthernet")
		}
		return "TenGigabitEthernet", rest
	case strings.HasPrefix(lower, "gigabitethernet"):
		return "GigabitEthernet", strings.TrimPrefix(port, "GigabitEthernet")
	case strings.HasPrefix(lower, "gi"):
		rest := port[2:]
		if strings.HasPrefix(rest, "gabitEthernet") {
			rest = strings.TrimPrefix(rest, "gabitEthernet")
		}
		return "GigabitEthernet", rest
	default:
		return "GigabitEthernet", port
	}
}

func displayPortName(ifType, ifName string) string {
	prefix := "Gi"
	if strings.EqualFold(ifType, "TenGigabitEthernet") {
		prefix = "Te"
	}
	ifName = strings.TrimSpace(ifName)
	if strings.HasPrefix(ifName, "1/") || strings.HasPrefix(ifName, "0/") {
		return prefix + ifName
	}
	return prefix + ifName
}
