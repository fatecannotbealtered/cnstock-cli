package output

import "testing"

func TestFilterFieldsObjectOrdered(t *testing.T) {
	in := []byte(`{"symbol":"sh600519","price":1800,"change_pct":0.87,"name":"x"}`)
	// Requested order differs from source order; output must follow requested order.
	got := string(filterFields(in, []string{"price", "symbol"}))
	want := `{"price":1800,"symbol":"sh600519"}`
	if got != want {
		t.Errorf("filterFields = %s, want %s", got, want)
	}
}

func TestFilterFieldsArray(t *testing.T) {
	in := []byte(`[{"a":1,"b":2},{"a":3,"b":4}]`)
	got := string(filterFields(in, []string{"b"}))
	want := `[{"b":2},{"b":4}]`
	if got != want {
		t.Errorf("filterFields = %s, want %s", got, want)
	}
}

func TestFilterFieldsMissingKeySkipped(t *testing.T) {
	in := []byte(`{"a":1}`)
	got := string(filterFields(in, []string{"a", "missing"}))
	want := `{"a":1}`
	if got != want {
		t.Errorf("filterFields = %s, want %s", got, want)
	}
}

func TestFilterFieldsNonObjectUnchanged(t *testing.T) {
	in := []byte(`"just a string"`)
	got := string(filterFields(in, []string{"a"}))
	if got != `"just a string"` {
		t.Errorf("filterFields mangled non-object: %s", got)
	}
}
