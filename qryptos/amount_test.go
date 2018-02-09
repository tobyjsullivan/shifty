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

func TestAmount_Multiply(t *testing.T) {
	am1 := Amount(297349782)
	am2 := Amount(874301822)

	product := am1.Multiply(am2)

	expected := Amount(2599734561)
	if product != expected {
		t.Errorf("The product did not match expected value. Expected: %d; Actual: %d.", expected, product)
	}
}