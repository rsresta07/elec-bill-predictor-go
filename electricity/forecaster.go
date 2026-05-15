package electricity

// CalculateSMA predicts the next value by averaging the last 'window' readings
func CalculateSMA(readings []float64) float64 {
	if len(readings) == 0 {
		return 0
	}
	var sum float64
	for _, r := range readings {
		sum += r
	}
	return sum / float64(len(readings))
}

// CalculateWMA predicts by giving more weight to recent data
// Example weights for 3 months: [0.2, 0.3, 0.5] where 0.5 is the most recent
func CalculateWMA(readings []float64, weights []float64) float64 {
	if len(readings) != len(weights) || len(readings) == 0 {
		return 0
	}
	var wma float64
	for i := 0; i < len(readings); i++ {
		wma += readings[i] * weights[i]
	}
	return wma
}
