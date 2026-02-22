package service

import "strconv"

func ValidateOrderNumber(number string) error {
	sum := 0
	isSecond := false

	for i := len(number) - 1; i >= 0; i-- {
		digit, err := strconv.Atoi(string(number[i]))
		if err != nil {
			return ErrInvalidOrderNumber
		}

		if isSecond {
			digit = digit * 2
			if digit > 9 {
				digit = digit - 9
			}
		}

		sum += digit
		isSecond = !isSecond
	}

	if sum%10 != 0 {
		return ErrInvalidOrderNumber
	}

	return nil
}
