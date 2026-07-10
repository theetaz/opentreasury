package treasury

import (
	"fmt"
	"math"
	"sort"
)

// The insights layer offers two deliberately transparent baseline methods:
// a moving-average flow forecast and robust (median/MAD) anomaly scoring.
// Both are explainable to an auditor in one sentence — a property this
// domain values over model sophistication. Fancier models can replace them
// behind the same API once a deployment has real history to validate on.

// MonthlyFlow is the posted debit total for one calendar month.
type MonthlyFlow struct {
	Period     string // YYYY-MM
	TotalMinor int64
}

// ForecastPoint projects one future month with an uncertainty band.
type ForecastPoint struct {
	Period         string
	ProjectedMinor int64
	LowMinor       int64
	HighMinor      int64
}

const forecastWindow = 3

// ForecastFlows projects the moving average of the last three observed
// months, with a band of ±1 mean absolute deviation that widens with the
// square root of the forecast distance.
func ForecastFlows(history []MonthlyFlow, horizon int) []ForecastPoint {
	if len(history) == 0 || horizon <= 0 {
		return nil
	}

	window := history
	if len(window) > forecastWindow {
		window = window[len(window)-forecastWindow:]
	}

	var sum int64
	for _, month := range window {
		sum += month.TotalMinor
	}
	mean := sum / int64(len(window))

	var deviation float64
	for _, month := range window {
		deviation += math.Abs(float64(month.TotalMinor - mean))
	}
	deviation /= float64(len(window))

	period := history[len(history)-1].Period
	forecast := make([]ForecastPoint, 0, horizon)
	for step := 1; step <= horizon; step++ {
		period = nextPeriod(period)
		band := int64(deviation * math.Sqrt(float64(step)))
		low := mean - band
		if low < 0 {
			low = 0
		}
		forecast = append(forecast, ForecastPoint{
			Period:         period,
			ProjectedMinor: mean,
			LowMinor:       low,
			HighMinor:      mean + band,
		})
	}
	return forecast
}

func nextPeriod(period string) string {
	var year, month int
	if _, err := fmt.Sscanf(period, "%d-%d", &year, &month); err != nil {
		return period
	}
	month++
	if month > 12 {
		month = 1
		year++
	}
	return fmt.Sprintf("%04d-%02d", year, month)
}

// FlowObservation is one posted debit line, the unit anomaly scoring works on.
type FlowObservation struct {
	EntryID       string
	InstitutionID string
	AccountCode   string
	AmountMinor   int64
	EffectiveDate string
}

// Anomaly is an observation whose amount is far outside what is typical for
// its account, with the evidence attached.
type Anomaly struct {
	EntryID       string
	InstitutionID string
	AccountCode   string
	AmountMinor   int64
	EffectiveDate string
	TypicalMinor  int64   // the account's median amount
	Score         float64 // robust z-score (modified, via MAD)
}

const (
	anomalyThreshold   = 3.5
	minObservations    = 5
	madConsistencyCoef = 1.4826 // scales MAD to a standard-deviation estimate
)

// ScoreAnomalies flags observations whose modified z-score
// |x − median| / (1.4826 × MAD) exceeds 3.5 within their account, the
// standard robust-outlier rule. Accounts with fewer than five observations
// are skipped — too little history to call anything unusual. Results are
// sorted by score, highest first.
func ScoreAnomalies(observations []FlowObservation) []Anomaly {
	byAccount := map[string][]FlowObservation{}
	for _, observation := range observations {
		byAccount[observation.AccountCode] = append(byAccount[observation.AccountCode], observation)
	}

	anomalies := []Anomaly{}
	for _, group := range byAccount {
		if len(group) < minObservations {
			continue
		}

		amounts := make([]float64, len(group))
		for i, observation := range group {
			amounts[i] = float64(observation.AmountMinor)
		}
		med := median(amounts)

		deviations := make([]float64, len(amounts))
		for i, amount := range amounts {
			deviations[i] = math.Abs(amount - med)
		}
		mad := median(deviations)
		if mad == 0 {
			// All-identical amounts: any different value is infinitely
			// surprising, but with zero spread the score is undefined —
			// fall back to a tiny spread so genuine jumps still flag.
			mad = 1
		}

		for _, observation := range group {
			score := math.Abs(float64(observation.AmountMinor)-med) / (madConsistencyCoef * mad)
			if score > anomalyThreshold {
				anomalies = append(anomalies, Anomaly{
					EntryID:       observation.EntryID,
					InstitutionID: observation.InstitutionID,
					AccountCode:   observation.AccountCode,
					AmountMinor:   observation.AmountMinor,
					EffectiveDate: observation.EffectiveDate,
					TypicalMinor:  int64(med),
					Score:         score,
				})
			}
		}
	}

	sort.Slice(anomalies, func(i, j int) bool {
		if anomalies[i].Score != anomalies[j].Score {
			return anomalies[i].Score > anomalies[j].Score
		}
		return anomalies[i].EntryID < anomalies[j].EntryID
	})
	return anomalies
}

func median(values []float64) float64 {
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	middle := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[middle]
	}
	return (sorted[middle-1] + sorted[middle]) / 2
}
