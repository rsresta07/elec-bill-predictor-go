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

// CalculateLinearRegression predicts the next value based on a trend line
func CalculateLinearRegression(data []float64) float64 {
	n := float64(len(data))
	if n < 2 {
		if n == 1 {
			return data[0]
		}
		return 0
	}

	var sumX, sumY, sumXY, sumXX float64
	for i, y := range data {
		x := float64(i + 1) // x is the time step (Month 1, 2, 3...)
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	// Calculate Slope (m)
	m := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)

	// Calculate Intercept (c)
	c := (sumY - m*sumX) / n

	// Predict for the next time step (n + 1)
	nextX := n + 1
	return m*nextX + c
}
