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
