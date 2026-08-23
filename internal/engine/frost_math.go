package engine

// applyFrostRelief applies the seasonal head threshold adjustment.
func applyFrostRelief(th HeadThresholds, relief float64) HeadThresholds {
	reduced := th.RestrictedDiffM + relief
	if reduced <= th.CriticalDiffM-0.5 {
		reduced = th.CriticalDiffM + 0.5
	}
	th.RestrictedDiffM = reduced
	return th
}
