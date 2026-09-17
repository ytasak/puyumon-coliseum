package battle

import "testing"

// 能力変化stageの倍率。Generation Iでは命中・回避のstageもこの表を使う。
func TestStageMultiplierTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		stage int
		want  StageRatio
	}{
		{-6, StageRatio{25, 100}},
		{-5, StageRatio{28, 100}},
		{-4, StageRatio{33, 100}},
		{-3, StageRatio{40, 100}},
		{-2, StageRatio{50, 100}},
		{-1, StageRatio{66, 100}},
		{0, StageRatio{1, 1}},
		{1, StageRatio{15, 10}},
		{2, StageRatio{2, 1}},
		{3, StageRatio{25, 10}},
		{4, StageRatio{3, 1}},
		{5, StageRatio{35, 10}},
		{6, StageRatio{4, 1}},
	}

	if len(tests) != StageMax-StageMin+1 {
		t.Fatalf("table covers %d stages, want %d", len(tests), StageMax-StageMin+1)
	}
	for _, tt := range tests {
		if got := StageMultiplier(tt.stage); got != tt.want {
			t.Errorf("StageMultiplier(%d) = %+v, want %+v", tt.stage, got, tt.want)
		}
	}
}

// 上下限を超えるstageは状態として成立しないのでpanicにする。
func TestStageMultiplierPanicsOutOfRange(t *testing.T) {
	t.Parallel()

	for _, stage := range []int{StageMin - 1, StageMax + 1, -100, 100} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("StageMultiplier(%d) did not panic", stage)
				}
			}()
			StageMultiplier(stage)
		}()
	}
}

// 代表的な能力値に対する各stageの結果。境界の-6と+6を含む。
func TestApplyStageAcrossEveryStage(t *testing.T) {
	t.Parallel()

	// 100を基準にすると倍率がそのまま百分率で読める。
	want := map[int]int{
		-6: 25,
		-5: 28,
		-4: 33,
		-3: 40,
		-2: 50,
		-1: 66,
		0:  100,
		1:  150,
		2:  200,
		3:  250,
		4:  300,
		5:  350,
		6:  400,
	}

	for stage := StageMin; stage <= StageMax; stage++ {
		if got := ApplyStage(100, stage); got != want[stage] {
			t.Errorf("ApplyStage(100, %d) = %d, want %d", stage, got, want[stage])
		}
	}
}

// 端数はGeneration Iと同じく切り捨てる。小数で持つと結果が変わる箇所。
func TestApplyStageTruncates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value int
		stage int
		want  int
	}{
		{3, -1, 1},     // 3 * 66 / 100 = 1.98 -> 1
		{7, 1, 10},     // 7 * 15 / 10 = 10.5 -> 10
		{1, -6, 0},     // 1 * 25 / 100 = 0.25 -> 0（下限の適用は能力値計算側の責務）
		{255, -1, 168}, // 255 * 66 / 100 = 168.3 -> 168
		{255, 6, 1020}, // 上限で頭打ちにしないこともここで確かめる
		{0, 6, 0},
	}

	for _, tt := range tests {
		if got := ApplyStage(tt.value, tt.stage); got != tt.want {
			t.Errorf("ApplyStage(%d, %d) = %d, want %d", tt.value, tt.stage, got, tt.want)
		}
	}
}
