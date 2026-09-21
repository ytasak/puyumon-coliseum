package game

import "testing"

// 次の対戦のseedは固定の規則で導く。
//
// ここを変えると同じ起動seedから別の試合列になるので、値を固定して
// 不用意に変わらないようにする。
func TestNextSeedIsStable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		seed uint64
		want uint64
	}{
		{seed: 0, want: 16294208416658607535},
		{seed: 1, want: 10451216379200822465},
		{seed: 42, want: 13679457532755275413},
	}
	for _, tc := range cases {
		if got := nextSeed(tc.seed); got != tc.want {
			t.Errorf("nextSeed(%d) = %d, want %d", tc.seed, got, tc.want)
		}
	}
}

// 続けて導いても同じseedへ戻らない。
//
// 戻ると「もう一度」で同じ対戦が繰り返される。
func TestNextSeedDoesNotRepeatQuickly(t *testing.T) {
	t.Parallel()

	seen := map[uint64]bool{}
	seed := uint64(1)
	for i := 0; i < 100; i++ {
		if seen[seed] {
			t.Fatalf("%d 回目でseedが巡回した: %d", i, seed)
		}
		seen[seed] = true
		seed = nextSeed(seed)
	}
}
