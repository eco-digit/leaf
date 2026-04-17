package intensity

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// MixFactors holds the per-kWh environmental intensity factors a country's electricity mix.
type MixFactors struct {
	ADP float64
	CED float64
	GWP float64
	WUE float64
}

// MixReader loads electricity mix factors from the CSV.
// Satisfies StaticFetcher: swap this for an API client when the data
// moves to an external data source/ endpoint.
type MixReader struct {
	data map[string]MixFactors
}

// LoadMixData parses the CSV.
// Expected columns: name,adpe,pe,gwp,wue — country code is a 3-letter ISO trigram ("DEU")
// Runtime info on ISO code can be set in the config.yaml.
func LoadMixData(path string) (*MixReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("intensity: open mix data %s: %w", path, err)
	}
	defer f.Close()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("intensity: parse mix data %s: %w", path, err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("intensity: mix data %s has no data rows", path)
	}

	data := make(map[string]MixFactors, len(rows)-1)
	for i, row := range rows[1:] {
		if len(row) < 5 {
			return nil, fmt.Errorf("intensity: mix data row %d has %d columns, want 5", i+2, len(row))
		}
		code := strings.TrimSpace(row[0])
		adp, err := strconv.ParseFloat(strings.TrimSpace(row[1]), 64)
		if err != nil {
			return nil, fmt.Errorf("intensity: mix data row %d adpe: %w", i+2, err)
		}
		pe, err := strconv.ParseFloat(strings.TrimSpace(row[2]), 64)
		if err != nil {
			return nil, fmt.Errorf("intensity: mix data row %d pe: %w", i+2, err)
		}
		gwp, err := strconv.ParseFloat(strings.TrimSpace(row[3]), 64)
		if err != nil {
			return nil, fmt.Errorf("intensity: mix data row %d gwp: %w", i+2, err)
		}
		wue, err := strconv.ParseFloat(strings.TrimSpace(row[4]), 64)
		if err != nil {
			return nil, fmt.Errorf("intensity: mix data row %d wue: %w", i+2, err)
		}
		data[code] = MixFactors{ADP: adp, CED: pe, GWP: gwp, WUE: wue}
	}

	return &MixReader{data: data}, nil
}

// Lookup returns electricity mix factors by countryCode.
func (r *MixReader) Lookup(countryCode string) (MixFactors, error) {
	f, ok := r.data[strings.ToUpper(countryCode)]
	if !ok {
		return MixFactors{}, fmt.Errorf("intensity: no mix data for country %q", countryCode)
	}
	return f, nil
}

func (r *MixReader) FetchGWP(countryCode string) (float64, error) {
	f, err := r.Lookup(countryCode)
	return f.GWP, err
}

func (r *MixReader) FetchADP(countryCode string) (float64, error) {
	f, err := r.Lookup(countryCode)
	return f.ADP, err
}

func (r *MixReader) FetchCED(countryCode string) (float64, error) {
	f, err := r.Lookup(countryCode)
	return f.CED, err
}

func (r *MixReader) FetchWUE(countryCode string) (float64, error) {
	f, err := r.Lookup(countryCode)
	return f.WUE, err
}
