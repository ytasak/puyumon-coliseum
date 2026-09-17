package battle

// MaxCriticalThreshold は急所判定の閾値の上限。乱数は0〜255を取る。
const MaxCriticalThreshold = 255

// CriticalThreshold は急所判定に使う閾値（0〜255）を返す。
//
// Generation Iの急所率は**base Speed**で決まる。現代世代のような固定確率ではない。
// 通常の技では floor(base Speed / 2)、高急所技ではその8倍で、255で頭打ちになる。
//
// base Speedは種族ごとの基礎値で、stat stageやまひを反映したeffective Speedとは別物。
// 値はspecies dataから渡す。
//
// Focus Energyの扱いはこのIssueの範囲外。実機のFocus Energyは急所率を4分の1にする
// 既知のバグがあり、再現するかはProduct Owner判断を要する。
func CriticalThreshold(baseSpeed int, highRatio bool) int {
	if baseSpeed < 0 {
		baseSpeed = 0
	}

	// 実機は byte 単位のshiftで計算する。/2してから×2するので、
	// 端数はここで落ちる。
	threshold := capThreshold(baseSpeed / 2 * 2)
	if highRatio {
		threshold = capThreshold(threshold * 2)
		threshold = capThreshold(threshold * 2)
	} else {
		threshold /= 2
	}
	return threshold
}

// capThreshold は閾値を255で頭打ちにする。
func capThreshold(threshold int) int {
	if threshold > MaxCriticalThreshold {
		return MaxCriticalThreshold
	}
	return threshold
}

// IsCritical は急所かどうかを判定する。
func IsCritical(rng RNG, baseSpeed int, highRatio bool) bool {
	return rng.IntN(MaxCriticalThreshold+1) < CriticalThreshold(baseSpeed, highRatio)
}
