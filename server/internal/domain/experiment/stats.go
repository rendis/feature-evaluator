package experiment

import "math"

// WilsonScore computes the Wilson score confidence interval for a proportion.
// z is the z-score (1.96 for 95% CI).
func WilsonScore(conversions, exposures int64, z float64) (low, high float64) {
	if exposures == 0 {
		return 0, 0
	}
	n := float64(exposures)
	p := float64(conversions) / n

	denominator := 1 + z*z/n
	center := p + z*z/(2*n)
	spread := z * math.Sqrt((p*(1-p)+z*z/(4*n))/n)

	low = (center - spread) / denominator
	high = (center + spread) / denominator

	if low < 0 {
		low = 0
	}
	if high > 1 {
		high = 1
	}
	return low, high
}
