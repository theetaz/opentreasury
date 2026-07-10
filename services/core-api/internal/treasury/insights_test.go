package treasury

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestForecastFlowsProjectsAMovingAverageWithBounds(t *testing.T) {
	history := []MonthlyFlow{
		{Period: "2026-01", TotalMinor: 100000},
		{Period: "2026-02", TotalMinor: 120000},
		{Period: "2026-03", TotalMinor: 110000},
		{Period: "2026-04", TotalMinor: 130000},
	}

	forecast := ForecastFlows(history, 3)

	require.Len(t, forecast, 3)
	// Projection is the mean of the last three observed months.
	expected := int64((120000 + 110000 + 130000) / 3)
	require.Equal(t, expected, forecast[0].ProjectedMinor)
	// Periods continue from the last observed month, across the year boundary
	// when needed.
	require.Equal(t, "2026-05", forecast[0].Period)
	require.Equal(t, "2026-06", forecast[1].Period)
	require.Equal(t, "2026-07", forecast[2].Period)
	// The uncertainty band widens with distance and brackets the projection.
	require.LessOrEqual(t, forecast[0].LowMinor, forecast[0].ProjectedMinor)
	require.GreaterOrEqual(t, forecast[0].HighMinor, forecast[0].ProjectedMinor)
	require.GreaterOrEqual(t,
		forecast[2].HighMinor-forecast[2].LowMinor,
		forecast[0].HighMinor-forecast[0].LowMinor,
	)
}

func TestForecastFlowsCrossesYearBoundary(t *testing.T) {
	history := []MonthlyFlow{{Period: "2026-11", TotalMinor: 5000}, {Period: "2026-12", TotalMinor: 7000}}
	forecast := ForecastFlows(history, 2)
	require.Equal(t, "2027-01", forecast[0].Period)
	require.Equal(t, "2027-02", forecast[1].Period)
}

func TestForecastFlowsWithNoHistoryIsEmpty(t *testing.T) {
	require.Empty(t, ForecastFlows(nil, 3))
	require.Empty(t, ForecastFlows([]MonthlyFlow{{Period: "2026-01", TotalMinor: 1}}, 0))
}

func TestScoreAnomaliesFlagsRobustOutliersPerAccount(t *testing.T) {
	lines := []FlowObservation{
		{EntryID: "je-1", AccountCode: "22", AmountMinor: 10000, EffectiveDate: "2026-07-01", InstitutionID: "minfin"},
		{EntryID: "je-2", AccountCode: "22", AmountMinor: 10500, EffectiveDate: "2026-07-02", InstitutionID: "minfin"},
		{EntryID: "je-3", AccountCode: "22", AmountMinor: 9800, EffectiveDate: "2026-07-03", InstitutionID: "minfin"},
		{EntryID: "je-4", AccountCode: "22", AmountMinor: 10200, EffectiveDate: "2026-07-04", InstitutionID: "minfin"},
		{EntryID: "je-5", AccountCode: "22", AmountMinor: 9900, EffectiveDate: "2026-07-05", InstitutionID: "minfin"},
		// The outlier: fifty times the typical payment on this account.
		{EntryID: "je-6", AccountCode: "22", AmountMinor: 500000, EffectiveDate: "2026-07-06", InstitutionID: "minfin"},
		// A different account with consistent amounts must not be flagged.
		{EntryID: "je-7", AccountCode: "31", AmountMinor: 700, EffectiveDate: "2026-07-01", InstitutionID: "minfin"},
		{EntryID: "je-8", AccountCode: "31", AmountMinor: 720, EffectiveDate: "2026-07-02", InstitutionID: "minfin"},
		{EntryID: "je-9", AccountCode: "31", AmountMinor: 690, EffectiveDate: "2026-07-03", InstitutionID: "minfin"},
	}

	anomalies := ScoreAnomalies(lines)

	require.Len(t, anomalies, 1)
	require.Equal(t, "je-6", anomalies[0].EntryID)
	require.Greater(t, anomalies[0].Score, 3.5)
	require.Equal(t, int64(10100), anomalies[0].TypicalMinor, "median of the account is the reference point")
}

func TestScoreAnomaliesNeedsEnoughHistoryPerAccount(t *testing.T) {
	// Four observations or fewer: not enough signal, nothing is flagged even
	// when values differ wildly.
	lines := []FlowObservation{
		{EntryID: "a", AccountCode: "22", AmountMinor: 1},
		{EntryID: "b", AccountCode: "22", AmountMinor: 1000000},
	}
	require.Empty(t, ScoreAnomalies(lines))
}

func TestScoreAnomaliesSortsByScoreDescending(t *testing.T) {
	lines := []FlowObservation{
		{EntryID: "n1", AccountCode: "22", AmountMinor: 100},
		{EntryID: "n2", AccountCode: "22", AmountMinor: 105},
		{EntryID: "n3", AccountCode: "22", AmountMinor: 98},
		{EntryID: "n4", AccountCode: "22", AmountMinor: 102},
		{EntryID: "n5", AccountCode: "22", AmountMinor: 101},
		{EntryID: "big", AccountCode: "22", AmountMinor: 10000},
		{EntryID: "bigger", AccountCode: "22", AmountMinor: 100000},
	}
	anomalies := ScoreAnomalies(lines)
	require.GreaterOrEqual(t, len(anomalies), 2)
	require.Equal(t, "bigger", anomalies[0].EntryID)
	require.GreaterOrEqual(t, anomalies[0].Score, anomalies[1].Score)
}
