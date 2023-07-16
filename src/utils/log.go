package utils

import (
	"fmt"

	"github.com/rs/zerolog"
)

func ToZeroLogArray[T fmt.Stringer](arr []T) (ret *zerolog.Array) {
	ret = zerolog.Arr()

	for _, elem := range arr {
		ret = ret.Str(elem.String())
	}

	return ret
}
