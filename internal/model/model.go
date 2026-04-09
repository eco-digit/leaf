// Package model defines Leaf's output-format-agnostic internal result model.
package model

import "time"

// SubjectType identifies what an ImpactResult is about
type SubjectType string

const (
	SubjectProvider SubjectType = "provider"
	SubjectDevice   SubjectType = "device"
	SubjectTenant   SubjectType = "tenant"
)

// ImpactPhase classifies whether a result reflects operational or embodied impact
type ImpactPhase string

const (
	PhaseOperational ImpactPhase = "operational"
	PhaseEmbodied    ImpactPhase = "embodied"
	PhaseTotal       ImpactPhase = "total"
)

// Category is one of four environmental impact categories plus energy
type Category string

const (
	CategoryGWP    Category = "gwp"
	CategoryADP    Category = "adp"
	CategoryCED    Category = "ced"
	CategoryWater  Category = "water"
	CategoryEnergy Category = "energy"
)

type ImpactResult struct {
	Subject     SubjectType
	Provider    string
	Datacenter  string
	Component   string
	Device      string
	ProjectID   string
	ProjectName string
	ImpactPhase ImpactPhase
	Category    Category
	Value       float64
	Unit        string
	Timestamp   time.Time
	PeriodHours int
}

type ResultSet []ImpactResult

func (rs ResultSet) FilterBySubject(s SubjectType) ResultSet {
	return rs.filter(func(r ImpactResult) bool { return r.Subject == s })
}

func (rs ResultSet) filter(keep func(ImpactResult) bool) ResultSet {
	out := make(ResultSet, 0, len(rs))
	for _, r := range rs {
		if keep(r) {
			out = append(out, r)
		}
	}
	return out
}
