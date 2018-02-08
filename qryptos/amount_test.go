package qryptos

import "testing"

func TestAmount_FromDecimal(t *testing.T) {
	dec := 0.347234

	var ca Amount
	ca.FromDecimal(dec)

	expected := 34723400
	if actual := int(ca); actual != expected {
		t.Errorf("Unexpected value. Expected: %d; Actual: %d.", expected, actual)
	}
}

func TestAmount_ToDecimal(t *testing.T) {
	ca := Amount(297349782)

	expected := 2.97349782
	if actual := ca.ToDecimal(); actual != expected {
		t.Errorf("Unexpected value. Expected: %.02f; Actual: %.02f.", expected, actual)
	}
}