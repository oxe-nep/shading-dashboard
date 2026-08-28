package groups

import (
	"fmt"
	"strconv"

	"github.com/oxe-nep/shading-dashboard/internal/model"
)

func ValidatePortGroups(groups []model.PortGroupConfig) error {
	seen := make(map[string]string)
	for _, g := range groups {
		if g.ID == "" {
			return fmt.Errorf("port group must have an id")
		}
		if g.Name == "" {
			return fmt.Errorf("port group %q must have a name", g.ID)
		}
		for _, m := range g.Members {
			key := m.SwitchID + "|" + m.Port
			if owner, ok := seen[key]; ok {
				return fmt.Errorf("port %s on switch %s is already in group %q", m.Port, m.SwitchID, owner)
			}
			seen[key] = g.ID
		}
	}
	return nil
}

func BuildRuntimeGroups(cfg []model.PortGroupConfig, switches []model.SwitchRuntimeState) []model.PortGroupRuntime {
	type portInfo struct {
		vlan  *int
		label string
	}
	portIndex := make(map[string]map[string]portInfo)
	for _, sw := range switches {
		portIndex[sw.ID] = make(map[string]portInfo)
		for _, p := range sw.Ports {
			portIndex[sw.ID][p.Name] = portInfo{
				vlan:  p.AccessVLAN,
				label: p.AccessVlanLabel,
			}
		}
	}

	out := make([]model.PortGroupRuntime, 0, len(cfg))
	for _, g := range cfg {
		rt := model.PortGroupRuntime{
			ID:      g.ID,
			Name:    g.Name,
			Members: append([]model.PortGroupMember(nil), g.Members...),
		}

		var vlanIDs []int
		var labels []string
		seenID := make(map[int]struct{})
		seenLabel := make(map[string]struct{})

		for _, m := range g.Members {
			if bySwitch, ok := portIndex[m.SwitchID]; ok {
				if info, ok := bySwitch[m.Port]; ok && info.vlan != nil {
					if _, dup := seenID[*info.vlan]; !dup {
						seenID[*info.vlan] = struct{}{}
						vlanIDs = append(vlanIDs, *info.vlan)
					}
					label := info.label
					if label == "" {
						label = strconv.Itoa(*info.vlan)
					}
					if _, dup := seenLabel[label]; !dup {
						seenLabel[label] = struct{}{}
						labels = append(labels, label)
					}
				}
			}
		}

		switch len(labels) {
		case 0:
			rt.VLANLabel = "—"
		case 1:
			rt.CurrentVLAN = &vlanIDs[0]
			rt.VLANLabel = labels[0]
		default:
			rt.Mixed = true
			rt.VLANLabel = "Mixed"
		}

		out = append(out, rt)
	}
	return out
}
