package battle

import "testing"

// 同じseedからは同じ列が出る。再現性の土台になる性質。
func TestRandRepeatsSequenceForSameSeed(t *testing.T) {
	t.Parallel()

	const seed = 20260918

	first := NewRand(seed)
	second := NewRand(seed)

	for i := 0; i < 100; i++ {
		a, b := first.Uint64(), second.Uint64()
		if a != b {
			t.Fatalf("draw %d: %d != %d", i, a, b)
		}
	}

	// IntN経由でも同じ列になる。
	first, second = NewRand(seed), NewRand(seed)
	for i := 0; i < 100; i++ {
		a, b := first.IntN(256), second.IntN(256)
		if a != b {
			t.Fatalf("IntN draw %d: %d != %d", i, a, b)
		}
	}
}

// 乱数列そのものを固定する。実装を変えたときに、これまでの対戦が再現できなく
// なったことへ気付けるようにする。
//
// seed=1の値はsplitmix64の原典が公表しているものと同じなので、
// このtestは「自分の実装が原典どおりか」の確認にもなっている。
func TestRandGoldenSequence(t *testing.T) {
	t.Parallel()

	wantUint64 := []uint64{
		0x910a2dec89025cc1,
		0xbeeb8da1658eec67,
		0xf893a2eefb32555e,
		0x71c18690ee42c90b,
		0x71bb54d8d101b5b9,
	}
	r := NewRand(1)
	for i, want := range wantUint64 {
		if got := r.Uint64(); got != want {
			t.Errorf("Uint64() draw %d = %#016x, want %#016x", i, got, want)
		}
	}

	// Generation Iの乱数は1 byte単位で使うことが多いため、その範囲でも固定する。
	wantIntN := []int{128, 84, 170, 44, 177, 95, 185, 135}
	r = NewRand(155)
	for i, want := range wantIntN {
		if got := r.IntN(256); got != want {
			t.Errorf("IntN(256) draw %d = %d, want %d", i, got, want)
		}
	}
}

// seedが違えば列も変わる。同じ値が並ぶようでは再現性の意味が無い。
func TestRandDiffersBetweenSeeds(t *testing.T) {
	t.Parallel()

	first := NewRand(1)
	second := NewRand(2)

	same := 0
	const draws = 100
	for i := 0; i < draws; i++ {
		if first.Uint64() == second.Uint64() {
			same++
		}
	}
	if same > 1 {
		t.Errorf("sequences matched %d/%d times, want at most 1", same, draws)
	}
}

// IntNは常に0以上n未満を返す。
func TestIntNStaysInRange(t *testing.T) {
	t.Parallel()

	for _, n := range []int{1, 2, 3, 6, 100, 255, 256, 1000} {
		r := NewRand(uint64(n))
		for i := 0; i < 1000; i++ {
			v := r.IntN(n)
			if v < 0 || v >= n {
				t.Fatalf("IntN(%d) = %d, want [0,%d)", n, v, n)
			}
		}
	}
}

// 取り得る値が1つだけなら常にその値になる。
func TestIntNWithOneAlwaysReturnsZero(t *testing.T) {
	t.Parallel()

	r := NewRand(7)
	for i := 0; i < 50; i++ {
		if v := r.IntN(1); v != 0 {
			t.Fatalf("IntN(1) = %d, want 0", v)
		}
	}
}

// 取り得る値はすべて出る。特定の値へ寄っていないことの最低限の確認。
func TestIntNCoversEveryValue(t *testing.T) {
	t.Parallel()

	const n = 6
	seen := make(map[int]int, n)

	r := NewRand(42)
	for i := 0; i < 600; i++ {
		seen[r.IntN(n)]++
	}

	for v := 0; v < n; v++ {
		if seen[v] == 0 {
			t.Errorf("IntN(%d) never returned %d", n, v)
		}
	}
}

// 取り得る値が無い呼び出しはpanicにする。握りつぶすと原因が分からなくなる。
func TestIntNPanicsWithoutRange(t *testing.T) {
	t.Parallel()

	for _, n := range []int{0, -1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("IntN(%d) did not panic", n)
				}
			}()
			NewRand(1).IntN(n)
		}()
	}
}

// RNGとして差し替えられる。testから決まった値を返す実装を渡せることの確認。
func TestRNGIsInjectable(t *testing.T) {
	t.Parallel()

	var rng RNG = fixedRNG{value: 3}
	if got := rng.IntN(256); got != 3 {
		t.Errorf("IntN() = %d, want 3", got)
	}

	rng = NewRand(1)
	if got := rng.IntN(1); got != 0 {
		t.Errorf("IntN(1) = %d, want 0", got)
	}
}

// fixedRNG は常に同じ値を返すRNG。
type fixedRNG struct {
	value int
}

func (f fixedRNG) IntN(n int) int {
	if f.value >= n {
		return n - 1
	}
	return f.value
}
