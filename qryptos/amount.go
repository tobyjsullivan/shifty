package qryptos

const (
	AmountRatio = 100000000
	MinimalUnit = Amount(1)
)

type Amount int

func (ca Amount) ToDecimal() float64 {
	return float64(ca) / float64(AmountRatio)
}

func (ca *Amount) FromDecimal(dec float64)  {
	*ca = Amount(dec * AmountRatio)
}
