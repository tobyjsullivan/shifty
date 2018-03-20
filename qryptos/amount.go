package qryptos

import "math"

const (
	AmountRatio = 100000000
	MinimalUnit = Amount(1)
	AmountZero = Amount(0)
)

type Amount int

func (ca Amount) ToDecimal() float64 {
	return float64(ca) / float64(AmountRatio)
}

func (ca *Amount) FromDecimal(dec float64)  {
	*ca = Amount(dec * AmountRatio)
}

func (ca Amount) Multiply(o Amount) Amount {
	return (ca * o) / AmountRatio
}

func (ca Amount) Divide(o Amount) Amount {
	f := (float64(ca) / float64(o)) * AmountRatio
	var rounded float64
	if f - math.Floor(f) < 0.5 {
		rounded = math.Floor(f)
	} else {
		rounded = math.Ceil(f)
	}

	return Amount(rounded)
}
