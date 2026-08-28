package switchdrv

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/oxe-nep/shading-dashboard/internal/model"
)

const (
	monitoredPortMin = 1
	monitoredPortMax = 48
)

var monitoredPortRe = regexp.MustCompile(`(?i)^Gi1/0/(\d+)$`)

func MonitoredPortName(index int) string {
	return fmt.Sprintf("Gi1/0/%d", index)
}

func IsMonitoredPort(name string) bool {
	m := monitoredPortRe.FindStringSubmatch(name)
	if m == nil {
		return false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return false
	}
	return n >= monitoredPortMin && n <= monitoredPortMax
}

func finalizeMonitoredPorts(ports []model.PortState) []model.PortState {
	byName := make(map[string]model.PortState, monitoredPortMax)
	for _, p := range ports {
		if IsMonitoredPort(p.Name) {
			byName[p.Name] = p
		}
	}

	out := make([]model.PortState, 0, monitoredPortMax)
	for i := monitoredPortMin; i <= monitoredPortMax; i++ {
		name := MonitoredPortName(i)
		if p, ok := byName[name]; ok {
			out = append(out, p)
			continue
		}
		out = append(out, model.PortState{
			Name:      name,
			OperState: "unknown",
		})
	}
	return out
}
