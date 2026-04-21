package collector

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/eco-digit/leaf/internal/infrastructure"
	prommodel "github.com/prometheus/common/model"
)

type VMInfo struct {
	VMID        string
	ProjectID   string
	ProjectName string
	FlavorName  string
	VCPUs       int
	MemoryGB    int
}

// collectVMInfo extracts VM information per Project
func collectVMInfo(q Querier, src *infrastructure.VMInfoSourceDef, r *RawMetrics) {
	if src == nil {
		return
	}

	val, err := q.QueryMetric(src.Metric)
	if err != nil {
		warn := fmt.Sprintf("%s query failed: %v", src.Metric, err)
		log.Printf("collector warning: %s", warn)
		r.Warnings = append(r.Warnings, warn)
		return
	}

	vec, _ := val.(prommodel.Vector)
	for _, s := range vec {
		uuid := string(s.Metric[prommodel.LabelName(src.UUIDLabel)])
		if uuid == "" {
			continue
		}
		flavor := string(s.Metric[prommodel.LabelName(src.FlavorLabel)])
		vcpus, memGB := ParseSCSFlavor(flavor)
		r.VMInfos = append(r.VMInfos, VMInfo{
			VMID:        uuid,
			ProjectID:   string(s.Metric[prommodel.LabelName(src.ProjectIDLabel)]),
			ProjectName: string(s.Metric[prommodel.LabelName(src.ProjectNameLabel)]),
			FlavorName:  flavor,
			VCPUs:       vcpus,
			MemoryGB:    memGB,
			// vmem
		})

	}
}

// ParseSCSFlavor
// WIP in general it makes sense that leaf knows the difference between different
// SCS CPU suffixes or disk sizes and types
func ParseSCSFlavor(name string) (vcpus int, memGB int) {
	if !strings.HasPrefix(name, "SCS-") {
		return 0, 0
	}
	parts := strings.Split(name[4:], "-")
	if len(parts) < 2 {
		return 0, 0
	}
	return parseLeadingInt(parts[0]), parseLeadingInt(parts[1])
}

func parseLeadingInt(s string) int {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
	}
	i++
	if i == 0 {
		return 0
	}
	v, _ := strconv.Atoi(s[:i])
	return v
}
