package cmd

import "fmt"

// formatPrice renders a float pointer as a price string with 3 decimal places,
// or "-" when the value is nil.
func formatPrice(f *float64) string {
	if f == nil {
		return "-"
	}
	return fmt.Sprintf("%.3f", *f)
}

// formatLargeNum renders a float pointer in human-readable form (亿/万 suffixes),
// or "-" when the value is nil.
func formatLargeNum(f *float64) string {
	if f == nil {
		return "-"
	}
	v := *f
	switch {
	case v >= 1e8:
		return fmt.Sprintf("%.2f亿", v/1e8)
	case v >= 1e4:
		return fmt.Sprintf("%.2f万", v/1e4)
	default:
		return fmt.Sprintf("%.0f", v)
	}
}
