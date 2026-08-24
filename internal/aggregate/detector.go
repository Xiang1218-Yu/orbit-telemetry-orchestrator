package aggregate

import (
	"fmt"
	"math"
	"time"

	"orbit-telemetry-orchestrator/internal/domain"
)

type Detection struct {
	RuleIndex int
	Severity  int
	Threshold string
}

func Detect(window domain.Window, rules []domain.Rule) []Detection {
	result := make([]Detection, 0)
	for index, rule := range rules {
		if rule.Signal != window.Signal {
			continue
		}
		triggered := false
		observed := window.LastValue
		switch rule.Comparator {
		case domain.ComparatorAbove:
			triggered = observed > rule.Upper || observed > rule.Lower
		case domain.ComparatorBelow:
			triggered = observed < rule.Lower || observed < rule.Upper
		case domain.ComparatorOutside:
			triggered = observed < rule.Lower || observed > rule.Upper
		case domain.ComparatorRate:
			if window.Count > 1 {
				rate := math.Abs(window.LastValue-window.Min) / float64(window.Count-1)
				triggered = rate > rule.Upper
			}
		}
		if triggered {
			result = append(result, Detection{
				RuleIndex: index, Severity: rule.Severity,
				Threshold: formatThreshold(rule, observed),
			})
		}
	}
	return result
}

func formatThreshold(rule domain.Rule, observed float64) string {
	switch rule.Comparator {
	case domain.ComparatorOutside:
		return fmt.Sprintf("outside[%g,%g], observed=%g", rule.Lower, rule.Upper, observed)
	case domain.ComparatorRate:
		return fmt.Sprintf("rate>%g, observed=%g", rule.Upper, observed)
	case domain.ComparatorAbove:
		return fmt.Sprintf("above>%g, observed=%g", max(rule.Lower, rule.Upper), observed)
	default:
		return fmt.Sprintf("below<%g, observed=%g", min(rule.Lower, rule.Upper), observed)
	}
}

func min(left, right float64) float64 {
	if left < right {
		return left
	}
	return right
}

func max(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func WindowBounds(now time.Time, width time.Duration) (time.Time, time.Time) {
	if width <= 0 {
		width = 5 * time.Minute
	}
	end := now.UTC()
	return end.Add(-width), end
}
