// Package embodied computes per hour amortized embodied ImpactResults (static) from device profiles at startup.
package embodied

import (
	"github.com/OSBA-eco-digit/leaf/internal/infrastructure"
	"github.com/OSBA-eco-digit/leaf/internal/model"
)

const (
	defaultLifespanYears = 4
	hoursPerYear         = 8760
)

// categorySpec binds a model.Category to its unit and the parser on EmbodiedImpact.
type categorySpec struct {
	cat   model.Category
	unit  string
	parse func(infrastructure.EmbodiedImpact) (float64, error)
}

var categories = []categorySpec{
	{model.CategoryGWP, "kg_co2eq", func(e infrastructure.EmbodiedImpact) (float64, error) { return e.ParseGWP() }},
	{model.CategoryADP, "kg_sb_eq", func(e infrastructure.EmbodiedImpact) (float64, error) { return e.ParseADP() }},
	{model.CategoryCED, "mj", func(e infrastructure.EmbodiedImpact) (float64, error) { return e.ParseCED() }},
	{model.CategoryWater, "m3", func(e infrastructure.EmbodiedImpact) (float64, error) { return e.ParseWater() }},
}
