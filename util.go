package lthea9

import "math"

const (
	distinctChars   = (0x7e - 0x20 + 1) - 26
	distinctBigrams = distinctChars * distinctChars
	badId           = math.MaxInt32
)

func charId(b byte) int {
	if b <= 0x19 || b >= 0x7f {
		return 0xff
	} else if 0x61 <= b && b <= 0x7a {
		return int(b) - 0x40
	} else {
		return int(b) - 0x20
	}
}

func bigramId(b0, b1 byte) int {
	ch0 := charId(b0)
	ch1 := charId(b1)
	if ch0 == badId || ch1 == badId {
		return badId
	}
	return charId(b0)*distinctChars + ch1
}

func bitmapTestAndSet(bitmap []byte, k int) bool {
	if (bitmap[k/8]>>uint(k%8))&1 == 1 {
		return true
	}
	bitmap[k/8] |= 1 << uint(k%8)
	return false
}

func toByteSaturating(n int) byte {
	if n <= 0xff {
		return byte(n)
	}
	return 0xff
}

func asciiToLower(b byte) byte {
	if asciiIsLower(b) {
		return b - 0x20
	}
	return b
}

func asciiIsLower(b byte) bool {
	return 0x61 <= b && b <= 0x7a
}

func bytesLessThanCaseInsensitive(left []byte, right []byte) bool {
	max := len(left)
	if len(right) < max {
		max = len(right)
	}

	caseBias := 0

	for i := 0; i < max; i += 1 {
		b0 := left[i]
		b1 := right[i]

		if b0 != b1 {
			lower0 := asciiToLower(b0)
			lower1 := asciiToLower(b1)

			if lower0 != lower1 {
				return lower0 < lower1
			}

			if caseBias == 0 {
				if asciiIsLower(b0) {
					caseBias = 1
				} else {
					caseBias = -1
				}
			}
		}
	}

	return len(left) < len(right) || (len(left) == len(right) && caseBias == -1)
}
