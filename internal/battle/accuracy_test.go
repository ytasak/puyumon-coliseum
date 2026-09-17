package battle

import "testing"

// 百分率からGeneration Iの0〜255表現への変換。実機のmove dataと同じ値になる。
func TestAccuracyPercent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		percent int
		want    int
	}{
		{100, 255},
		{95, 242},
		{90, 229},
		{85, 216},
		{80, 204},
		{75, 191},
		{70, 178},
		{55, 140},
		{50, 127},
		{30, 76},
		{0, 0},
		{-10, 0},
		{200, 255},
	}

	for _, tt := range tests {
		if got := AccuracyPercent(tt.percent); got != tt.want {
			t.Errorf("AccuracyPercent(%d) = %d, want %d", tt.percent, got, tt.want)
		}
	}
}

// 命中stageと回避stageを反映した命中率。
//
// 期待値はpokeredのCalcHitChanceから起こした参照実装で別に求めたもの。
func TestHitChance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		baseAccuracy int
		accuracy     int
		evasion      int
		want         int
	}{
		{"no stages", 255, 0, 0, 255},
		{"accuracy down one", 255, -1, 0, 168},
		{"evasion up one", 255, 0, 1, 168},
		{"accuracy floor", 255, StageMin, 0, 63},
		{"evasion ceiling", 255, 0, StageMax, 63},
		{"accuracy up does not exceed the maximum", 255, StageMax, 0, 255},
		{"evasion down does not exceed the maximum", 255, 0, StageMin, 255},
		{"both extremes against the attacker", 255, StageMin, StageMax, 15},
		{"both extremes for the attacker", 255, StageMax, StageMin, 255},
		{"imperfect accuracy", 242, 0, 0, 242},
		{"three quarters", 191, 0, 0, 191},
		{"low accuracy", 76, 0, 0, 76},
		{"never drops below one", 1, StageMin, StageMax, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := HitChance(tt.baseAccuracy, tt.accuracy, tt.evasion)
			if got != tt.want {
				t.Errorf("HitChance(%d, %d, %d) = %d, want %d",
					tt.baseAccuracy, tt.accuracy, tt.evasion, got, tt.want)
			}
		})
	}
}

// 命中判定の境界。乱数が命中率未満なら当たる。
func TestHitsAtBoundary(t *testing.T) {
	t.Parallel()

	const accuracy = 191

	if !Hits(fixedRNG{value: accuracy - 1}, accuracy, 0, 0) {
		t.Errorf("roll %d missed, want hit", accuracy-1)
	}
	if Hits(fixedRNG{value: accuracy}, accuracy, 0, 0) {
		t.Errorf("roll %d hit, want miss", accuracy)
	}
}

// 命中率が最大なら必ず当たる。実機の1/256 missは再現しない（Issueの指示）。
func TestHitsNeverMissesAtMaximumAccuracy(t *testing.T) {
	t.Parallel()

	// 実機で外れるのは乱数が255のとき。その値でも当たることを確かめる。
	if !Hits(fixedRNG{value: 255}, MaxAccuracy, 0, 0) {
		t.Error("maximum accuracy missed on the highest roll, want hit")
	}

	rng := NewRand(155)
	for i := 0; i < 5000; i++ {
		if !Hits(rng, MaxAccuracy, 0, 0) {
			t.Fatalf("draw %d missed at maximum accuracy, want hit", i)
		}
	}
}

// 命中判定は結果によらず乱数をちょうど1つ消費する。
// 消費数が結果で変わると、後続の乱数がずれて再現性を追いにくくなる。
func TestHitsConsumesOneRandomValue(t *testing.T) {
	t.Parallel()

	for _, accuracy := range []int{MaxAccuracy, 191, 1} {
		used := NewRand(7)
		Hits(used, accuracy, 0, 0)
		got := used.IntN(256)

		reference := NewRand(7)
		reference.IntN(256)
		want := reference.IntN(256)

		if got != want {
			t.Errorf("accuracy %d: next value = %d, want %d (one draw consumed)",
				accuracy, got, want)
		}
	}
}

// 回避を上げると命中率が下がり、命中を上げると戻る。
func TestHitsRateFollowsStages(t *testing.T) {
	t.Parallel()

	const draws = 20000

	rate := func(accuracyStage, evasionStage int) float64 {
		rng := NewRand(20260918)
		hits := 0
		for i := 0; i < draws; i++ {
			if Hits(rng, AccuracyPercent(100), accuracyStage, evasionStage) {
				hits++
			}
		}
		return float64(hits) / draws
	}

	if got := rate(0, 0); got != 1 {
		t.Errorf("hit rate without stages = %.4f, want 1", got)
	}

	// 回避+6なら命中率は63/255。
	want := float64(HitChance(AccuracyPercent(100), 0, StageMax)) / 256
	if got := rate(0, StageMax); got < want-0.02 || got > want+0.02 {
		t.Errorf("hit rate against maximum evasion = %.4f, want about %.4f", got, want)
	}
}
