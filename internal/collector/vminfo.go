package collector

import (
	"fmt"
	"log"

	"github.com/eco-digit/leaf/internal/infrastructure"
	prommodel "github.com/prometheus/common/model"
)

type VMInfo struct {
	VMID        string
	ProjectID   string
	ProjectName string
	FlavourName string
	VCPUs       float64
	MemoryGB    float64
}

// collectVMInfo extracts VM information per Project
func collectVMInfo(q Querier, src *infrastructure.VMInfoSourceDef, r RawMetrics) {
	if src == nil {
		return
	}

	val, err := q.QueryMetric(src.Metric)
	if err != nil {
		warn := fmt.Sprintf("s% query failed: %v", src.Metric, err)
		log.Printf("collector warning: %s", warn)
		r.Warnings = append(r.Warnings, warn)
	}

	vec, _ := val.(prommodel.Vector)
	for _, s := range vec {
		uuid := string(s.Metric[prommodel.LabelName(src.UUIDLabel)])
		if uuid == "" {
			continue
		}
		flavor := string(s.Metric[prommodel.LabelName(src.FlavorLabel)])
		// TODO parse Flavor
		//  vcpus, memGB:= ParseSCSFlavor()
		r.VMInfos = append(r.VMInfos, VMInfo{
			VMID:        uuid,
			ProjectID:   string(s.Metric[prommodel.LabelName(src.ProjectIDLabel)]),
			ProjectName: string(s.Metric[prommodel.LabelName(src.ProjectNameLabel)]),
			FlavourName: flavor,
			// vcpus
			// vmem
		})

	}
}

func ParseSCSFlavor(name string) (vcpus float64, ramGB float64) {
	return
}
