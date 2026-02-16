package common

import "fmt"

const (
	MIN_PRICE = 1
	MAX_PRICE = 999999
)

func ValidatePrice(fieldName string, price float64, required bool) error {
	if !required && price == 0 {
		return nil
	}

	if price < MIN_PRICE {
		return fmt.Errorf("%s price must be at least %d", fieldName, MIN_PRICE)
	}
	if price > MAX_PRICE {
		return fmt.Errorf("%s price must not exceed %d", fieldName, MAX_PRICE)
	}
	return nil
}
