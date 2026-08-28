package switchdrv

import (
	"encoding/xml"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/oxe-nep/shading-dashboard/internal/model"
)

func parseVLANNames(raw string) map[int]string {
	list := parseVLANList(raw)
	out := make(map[int]string, len(list))
	for _, v := range list {
		out[v.ID] = v.Name
	}
	return out
}

func parseVLANList(raw string) []model.VLANInfo {
	var data rpcData
	if err := xml.Unmarshal([]byte(raw), &data); err != nil {
		return nil
	}
	out := make([]model.VLANInfo, 0, len(data.Native.VLAN.VLANList))
	for _, v := range data.Native.VLAN.VLANList {
		if v.ID > 0 {
			out = append(out, buildVLANInfo(v.ID, strings.TrimSpace(v.Name)))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func buildVLANInfo(id int, name string) model.VLANInfo {
	return model.VLANInfo{
		ID:    id,
		Name:  name,
		Label: vlanDisplayLabel(id, name),
		Title: vlanDisplayTitle(id, name),
	}
}

func mergeDiscoveredVLANs(states []model.SwitchRuntimeState) []model.VLANInfo {
	byID := make(map[int]string)
	for _, st := range states {
		for _, v := range st.Vlans {
			if existing, ok := byID[v.ID]; !ok || (existing == "" && v.Name != "") {
				byID[v.ID] = v.Name
			}
		}
	}
	out := make([]model.VLANInfo, 0, len(byID))
	for id, name := range byID {
		out = append(out, buildVLANInfo(id, name))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func filterSelectableVLANs(discovered []model.VLANInfo, allowed []int) []model.VLANInfo {
	if len(allowed) == 0 {
		out := make([]model.VLANInfo, len(discovered))
		copy(out, discovered)
		return out
	}

	byID := make(map[int]string, len(discovered))
	for _, v := range discovered {
		byID[v.ID] = v.Name
	}

	out := make([]model.VLANInfo, 0, len(allowed))
	for _, id := range allowed {
		if id <= 0 {
			continue
		}
		out = append(out, buildVLANInfo(id, byID[id]))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func vlanNamesMap(vlans []model.VLANInfo) map[int]string {
	out := make(map[int]string, len(vlans))
	for _, v := range vlans {
		out[v.ID] = v.Name
	}
	return out
}

func vlanDisplayLabel(id int, name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return strconv.Itoa(id)
	}
	return name
}

func vlanDisplayTitle(id int, name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Sprintf("VLAN %d", id)
	}
	return fmt.Sprintf("%s (VLAN %d)", name, id)
}

func applyVLANLabels(ports []model.PortState, names map[int]string) {
	if names == nil {
		return
	}
	for i := range ports {
		if ports[i].AccessVLAN == nil {
			continue
		}
		id := *ports[i].AccessVLAN
		name := names[id]
		label := vlanDisplayLabel(id, name)
		title := vlanDisplayTitle(id, name)
		ports[i].AccessVlanLabel = label
		ports[i].AccessVlanTitle = title
	}
}
