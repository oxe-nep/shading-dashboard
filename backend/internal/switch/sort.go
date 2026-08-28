package switchdrv

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/oxe-nep/shading-dashboard/internal/model"
)

var portNumberParts = regexp.MustCompile(`(\d+)`)

// ComparePortNames sorts Gi1/0/2 before Gi1/0/10 (natural order).
func ComparePortNames(a, b string) bool {
	aParts := portNumberParts.FindAllString(a, -1)
	bParts := portNumberParts.FindAllString(b, -1)

	max := len(aParts)
	if len(bParts) > max {
		max = len(bParts)
	}
	for i := 0; i < max; i++ {
		var av, bv int
		if i < len(aParts) {
			av, _ = strconv.Atoi(aParts[i])
		}
		if i < len(bParts) {
			bv, _ = strconv.Atoi(bParts[i])
		}
		if av != bv {
			return av < bv
		}
	}

	pa := strings.ToLower(a)
	pb := strings.ToLower(b)
	if pa != pb {
		return pa < pb
	}
	return a < b
}

func sortPorts(ports []model.PortState) {
	sort.Slice(ports, func(i, j int) bool {
		return ComparePortNames(ports[i].Name, ports[j].Name)
	})
}
