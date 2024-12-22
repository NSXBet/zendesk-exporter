package collector

import "strconv"

// isNumeric checks if a string represents a number
func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}
