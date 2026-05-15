package electricity

import (
	"math"
)

// Slab represents a row from your TariffSlabs table
type Slab struct {
	Start       float64
	End         float64
	MinFee      float64
	Rate        float64
	SpecialRate float64 // Used for the 5A "Rs. 3" rule
}

type BillReport struct {
	TotalUnits   float64
	EnergyCharge float64
	MinimumFee   float64
	TotalAmount  float64
}

// CalculateExactBill implements the cumulative slab logic
func CalculateExactBill(amps int, totalUnits float64, slabs []Slab) BillReport {
	var energyCharge float64
	var finalMinFee float64

	for _, s := range slabs {
		// 1. Check if the user's consumption even reaches this slab
		if totalUnits > s.Start {

			// Determine units belonging to THIS slab
			upperLimit := s.End
			if s.End == 0 { // 0 represents '251+' or infinity
				upperLimit = totalUnits
			}

			unitsInThisSlab := math.Min(totalUnits, upperLimit) - s.Start

			// 2. Apply special 5A logic for the first slab (0-20)
			currentRate := s.Rate
			if amps == 5 && s.Start == 0 {
				if totalUnits > 20 {
					currentRate = 3.0 // The "Important Note" rule
				} else {
					currentRate = 0.0
				}
			}

			energyCharge += unitsInThisSlab * currentRate

			// 3. Update Min Fee
			// The rule is: your Min Fee is dictated by the highest slab you touch.
			finalMinFee = s.MinFee
		}
	}

	return BillReport{
		TotalUnits:   totalUnits,
		EnergyCharge: energyCharge,
		MinimumFee:   finalMinFee,
		TotalAmount:  energyCharge + finalMinFee,
	}
}
