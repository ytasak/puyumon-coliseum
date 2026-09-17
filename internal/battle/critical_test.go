package battle

import "testing"

// 急所率はbase Speedで決まる。現代世代のような固定確率ではない。
//
// 期待値はpokeredのCriticalHitTestから起こした参照実装で別に求めたもの。
func TestCriticalThreshold(t *testing.T) {
	t.Parallel()

	tests := []struct {
		baseSpeed int
		highRatio bool
		want      int
	}{
		{0, false, 0},
		{1, false, 0},
		{5, false, 2},
		{50, false, 25},
		{55, false, 27}, // 端数はbyte単位のshiftで落ちる
		{100, false, 50},
		{110, false, 55},
		{130, false, 65},
		{140, false, 70},
		{255, false, 127},
		{5, true, 16},
		{50, true, 200},
		{55, true, 216},
		{100, true, 255}, // 8倍は255で頭打ち
		{110, true, 255},
		{255, true, 255},
	}

	for _, tt := range tests {
		if got := CriticalThreshold(tt.baseSpeed, tt.highRatio); got != tt.want {
			t.Errorf("CriticalThreshold(%d, %v) = %d, want %d",
				tt.baseSpeed, tt.highRatio, got, tt.want)
		}
	}
}

// base Speedが高いほど急所率も高い。速いポケモンほど急所を出しやすいのがGeneration I。
func TestCriticalThresholdRisesWithBaseSpeed(t *testing.T) {
	t.Parallel()

	previous := -1
	for _, baseSpeed := range []int{5, 30, 50, 70, 90, 110, 130, 150} {
		threshold := CriticalThreshold(baseSpeed, false)
		if threshold <= previous {
			t.Errorf("base speed %d: threshold %d did not rise above %d",
				baseSpeed, threshold, previous)
		}
		previous = threshold
	}

	// 高急所技は同じbase Speedでも急所率が上がる。
	if CriticalThreshold(55, true) <= CriticalThreshold(55, false) {
		t.Error("high critical ratio move did not raise the threshold")
	}
}

// 閾値未満の乱数で急所になる。境界の値を確かめる。
func TestIsCriticalAtThresholdBoundary(t *testing.T) {
	t.Parallel()

	const baseSpeed = 110 // 閾値は55
	threshold := CriticalThreshold(baseSpeed, false)

	if !IsCritical(fixedRNG{value: threshold - 1}, baseSpeed, false) {
		t.Errorf("roll %d was not a critical hit, want critical", threshold-1)
	}
	if IsCritical(fixedRNG{value: threshold}, baseSpeed, false) {
		t.Errorf("roll %d was a critical hit, want none", threshold)
	}
	if IsCritical(fixedRNG{value: 255}, baseSpeed, false) {
		t.Error("highest roll was a critical hit, want none")
	}
}

// 実際の発生率が閾値どおりになる。seedを固定しているので結果は毎回同じ。
func TestIsCriticalRateFollowsThreshold(t *testing.T) {
	t.Parallel()

	const (
		baseSpeed = 110 // 閾値55 → 55/256 ≈ 21.5%
		draws     = 20000
	)

	rng := NewRand(155)
	criticals := 0
	for i := 0; i < draws; i++ {
		if IsCritical(rng, baseSpeed, false) {
			criticals++
		}
	}

	want := float64(CriticalThreshold(baseSpeed, false)) / 256
	got := float64(criticals) / draws
	if diff := got - want; diff < -0.02 || diff > 0.02 {
		t.Errorf("critical rate = %.4f, want about %.4f", got, want)
	}
}
