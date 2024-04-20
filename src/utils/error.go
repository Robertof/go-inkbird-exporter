package utils

import "errors"

func ErrorIsAnyOf(err error, targets... error) bool {
	for _, target := range targets {
		if errors.Is(err, target) {
			return true
		}
	}

	return false
}
