package download_exp

import (
	"math"
	"os"
	"strconv"
)

// SecondsFromSize mirrors the legacy size-aware download link TTL (log-based formula).
func SecondsFromSize(fileSizeBytes int64, ratio float64) int {
	if ratio <= 0 {
		ratio = 4.25
	}
	const lowerThreshold = 100_000
	sz := fileSizeBytes
	if sz < lowerThreshold {
		sz = lowerThreshold
	}
	return int(math.Round(math.Pow(math.Log10(float64(sz)), ratio)))
}

// SecondsFromEnv uses DOWNLOAD_EXP_BYTES_SECS_RATIO when ratio is zero.
func SecondsFromEnv(fileSizeBytes int64, ratio float64) int {
	if ratio <= 0 {
		r, err := strconv.ParseFloat(os.Getenv("DOWNLOAD_EXP_BYTES_SECS_RATIO"), 64)
		if err == nil && r > 0 {
			ratio = r
		} else {
			ratio = 4.25
		}
	}
	return SecondsFromSize(fileSizeBytes, ratio)
}
