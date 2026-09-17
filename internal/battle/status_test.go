package battle

import "testing"

// 状態異常を持たせたPokemonを返す。
func statusPokemon(status MajorStatus) Pokemon {
	p := Pokemon{
		Species:   "test",
		Level:     55,
		CurrentHP: 200,
		Stats:     Stats{HP: 200, Attack: 180, Defense: 150, Speed: 130, Special: 120},
		Status:    status,
	}
	return p
}

// ねむりの継続turnは1〜7。
func TestSleepDurationRange(t *testing.T) {
	t.Parallel()

	seen := make(map[int]bool, MaxSleepTurns)
	rng := NewRand(155)
	for i := 0; i < 2000; i++ {
		turns := SleepDuration(rng)
		if turns < MinSleepTurns || turns > MaxSleepTurns {
			t.Fatalf("SleepDuration() = %d, want [%d,%d]", turns, MinSleepTurns, MaxSleepTurns)
		}
		seen[turns] = true
	}
	for turns := MinSleepTurns; turns <= MaxSleepTurns; turns++ {
		if !seen[turns] {
			t.Errorf("SleepDuration() never returned %d", turns)
		}
	}
}

// 乱数の値と継続turnの対応。
func TestSleepDurationFromRandomValue(t *testing.T) {
	t.Parallel()

	for value := 0; value < MaxSleepTurns; value++ {
		want := value + 1
		if got := SleepDuration(fixedRNG{value: value}); got != want {
			t.Errorf("SleepDuration() with roll %d = %d, want %d", value, got, want)
		}
	}
}

// ねむりはcounterが0になるまで行動できず、起床したturnも行動できない。
func TestCheckStatusSleepCountdown(t *testing.T) {
	t.Parallel()

	p := statusPokemon(Sleep)
	p.SleepTurns = 3

	tests := []struct {
		wantBlock  StatusBlock
		wantTurns  int
		wantStatus MajorStatus
	}{
		{BlockedBySleep, 2, Sleep},
		{BlockedBySleep, 1, Sleep},
		{BlockedByWakeUp, 0, NoStatus}, // 起床したturnも動けない
	}

	rng := NewRand(1)
	for i, tt := range tests {
		block := CheckStatus(rng, &p)
		if block != tt.wantBlock {
			t.Errorf("turn %d: block = %v, want %v", i+1, block, tt.wantBlock)
		}
		if p.SleepTurns != tt.wantTurns {
			t.Errorf("turn %d: sleep turns = %d, want %d", i+1, p.SleepTurns, tt.wantTurns)
		}
		if p.Status != tt.wantStatus {
			t.Errorf("turn %d: status = %v, want %v", i+1, p.Status, tt.wantStatus)
		}
	}

	// 起床後は妨げられない。
	if block := CheckStatus(rng, &p); block != NotBlocked {
		t.Errorf("after waking: block = %v, want %v", block, NotBlocked)
	}
}

// 継続turnがNなら、Nターン行動できない。
func TestSleepBlocksForItsWholeDuration(t *testing.T) {
	t.Parallel()

	for duration := MinSleepTurns; duration <= MaxSleepTurns; duration++ {
		p := statusPokemon(Sleep)
		p.SleepTurns = duration

		rng := NewRand(1)
		blocked := 0
		for turn := 0; turn < duration+3; turn++ {
			if CheckStatus(rng, &p) != NotBlocked {
				blocked++
			}
		}
		if blocked != duration {
			t.Errorf("duration %d: blocked for %d turns, want %d", duration, blocked, duration)
		}
	}
}

// こおりは自然に解けない。現代世代の自然解凍を持ち込まない。
func TestFreezeNeverThawsOnItsOwn(t *testing.T) {
	t.Parallel()

	p := statusPokemon(Freeze)
	rng := NewRand(155)

	for turn := 0; turn < 100; turn++ {
		if block := CheckStatus(rng, &p); block != BlockedByFreeze {
			t.Fatalf("turn %d: block = %v, want %v", turn, block, BlockedByFreeze)
		}
		if p.Status != Freeze {
			t.Fatalf("turn %d: status = %v, want %v", turn, p.Status, Freeze)
		}
	}

	// 解除は技の側から状態を消して行う。
	p.Status = NoStatus
	if block := CheckStatus(rng, &p); block != NotBlocked {
		t.Errorf("after thawing: block = %v, want %v", block, NotBlocked)
	}
}

// まひで行動できなくなる境界。乱数が63未満なら動けない。
func TestFullParalysisBoundary(t *testing.T) {
	t.Parallel()

	if !IsFullyParalyzed(fixedRNG{value: FullParalysisThreshold - 1}) {
		t.Errorf("roll %d was not fully paralyzed, want blocked", FullParalysisThreshold-1)
	}
	if IsFullyParalyzed(fixedRNG{value: FullParalysisThreshold}) {
		t.Errorf("roll %d was fully paralyzed, want able to act", FullParalysisThreshold)
	}
	if IsFullyParalyzed(fixedRNG{value: 255}) {
		t.Error("highest roll was fully paralyzed, want able to act")
	}
}

// まひの行動不能はおよそ25%。seedを固定しているので結果は毎回同じ。
func TestFullParalysisRate(t *testing.T) {
	t.Parallel()

	const draws = 20000

	rng := NewRand(155)
	blocked := 0
	for i := 0; i < draws; i++ {
		if IsFullyParalyzed(rng) {
			blocked++
		}
	}

	want := float64(FullParalysisThreshold) / randomByteValues
	got := float64(blocked) / draws
	if diff := got - want; diff < -0.02 || diff > 0.02 {
		t.Errorf("full paralysis rate = %.4f, want about %.4f", got, want)
	}
}

// まひはturn開始の判定にも効く。
func TestCheckStatusParalysis(t *testing.T) {
	t.Parallel()

	p := statusPokemon(Paralysis)

	if block := CheckStatus(fixedRNG{value: FullParalysisThreshold - 1}, &p); block != BlockedByParalysis {
		t.Errorf("block = %v, want %v", block, BlockedByParalysis)
	}
	if block := CheckStatus(fixedRNG{value: FullParalysisThreshold}, &p); block != NotBlocked {
		t.Errorf("block = %v, want %v", block, NotBlocked)
	}
	// まひは解けない。
	if p.Status != Paralysis {
		t.Errorf("status = %v, want %v", p.Status, Paralysis)
	}
}

// 状態異常が無ければ妨げられない。
func TestCheckStatusWithoutStatus(t *testing.T) {
	t.Parallel()

	p := statusPokemon(NoStatus)
	if block := CheckStatus(NewRand(1), &p); block != NotBlocked {
		t.Errorf("block = %v, want %v", block, NotBlocked)
	}
}

// まひはSpeedを4分の1、やけどはAttackを半分にする。
func TestStatusStatModifiers(t *testing.T) {
	t.Parallel()

	speeds := []struct{ speed, want int }{
		{130, 32}, {100, 25}, {4, 1}, {3, 0}, {0, 0},
	}
	for _, tt := range speeds {
		if got := ParalyzedSpeed(tt.speed); got != tt.want {
			t.Errorf("ParalyzedSpeed(%d) = %d, want %d", tt.speed, got, tt.want)
		}
	}

	attacks := []struct{ attack, want int }{
		{180, 90}, {3, 1}, {1, 0}, {0, 0},
	}
	for _, tt := range attacks {
		if got := BurnedAttack(tt.attack); got != tt.want {
			t.Errorf("BurnedAttack(%d) = %d, want %d", tt.attack, got, tt.want)
		}
	}
}

// 毒・やけどの継続ダメージは最大HPの16分の1。切り捨てて0でも1は受ける。
func TestResidualDamage(t *testing.T) {
	t.Parallel()

	tests := []struct{ maxHP, want int }{
		{200, 12},
		{160, 10},
		{16, 1},
		{15, 1}, // 下限
		{1, 1},
		{999, 62},
	}

	for _, tt := range tests {
		if got := ResidualDamage(tt.maxHP); got != tt.want {
			t.Errorf("ResidualDamage(%d) = %d, want %d", tt.maxHP, got, tt.want)
		}
	}
}

// 継続ダメージは毒とやけどにだけ効く。
func TestApplyResidualDamage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status     MajorStatus
		currentHP  int
		wantDamage int
		wantHP     int
	}{
		{Poison, 200, 12, 188},
		{Burn, 200, 12, 188},
		{Paralysis, 200, 0, 200},
		{Sleep, 200, 0, 200},
		{Freeze, 200, 0, 200},
		{NoStatus, 200, 0, 200},
		{Poison, 5, 5, 0}, // HPより大きいダメージは残りHPまで
		{Poison, 0, 0, 0}, // 戦闘不能には効かない
	}

	for _, tt := range tests {
		p := statusPokemon(tt.status)
		p.CurrentHP = tt.currentHP

		damage := ApplyResidualDamage(&p)
		if damage != tt.wantDamage {
			t.Errorf("%v: damage = %d, want %d", tt.status, damage, tt.wantDamage)
		}
		if p.CurrentHP != tt.wantHP {
			t.Errorf("%v: current HP = %d, want %d", tt.status, p.CurrentHP, tt.wantHP)
		}
	}
}

// 状態異常は同時に1つだけ。すでに持っていれば上書きしない。
func TestApplyStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		current   MajorStatus
		fainted   bool
		apply     MajorStatus
		wantOK    bool
		wantAfter MajorStatus
	}{
		{"onto a healthy mon", NoStatus, false, Paralysis, true, Paralysis},
		{"already statused", Burn, false, Paralysis, false, Burn},
		{"same status twice", Sleep, false, Sleep, false, Sleep},
		{"fainted", NoStatus, true, Poison, false, NoStatus},
		{"no status to apply", NoStatus, false, NoStatus, false, NoStatus},
		{"unknown status", NoStatus, false, MajorStatus(99), false, NoStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := statusPokemon(tt.current)
			if tt.current == Sleep {
				p.SleepTurns = 2
			}
			if tt.fainted {
				p.CurrentHP = 0
			}

			got := ApplyStatus(NewRand(1), &p, tt.apply)
			if got != tt.wantOK {
				t.Errorf("ApplyStatus() = %v, want %v", got, tt.wantOK)
			}
			if p.Status != tt.wantAfter {
				t.Errorf("status = %v, want %v", p.Status, tt.wantAfter)
			}
		})
	}
}

// ねむりを付与すると継続turnも決まる。
func TestApplyStatusSetsSleepTurns(t *testing.T) {
	t.Parallel()

	rng := NewRand(155)
	for i := 0; i < 100; i++ {
		p := statusPokemon(NoStatus)
		if !ApplyStatus(rng, &p, Sleep) {
			t.Fatal("ApplyStatus() = false, want true")
		}
		if p.SleepTurns < MinSleepTurns || p.SleepTurns > MaxSleepTurns {
			t.Fatalf("sleep turns = %d, want [%d,%d]", p.SleepTurns, MinSleepTurns, MaxSleepTurns)
		}
	}

	// ねむり以外はcounterを持たない。
	p := statusPokemon(NoStatus)
	ApplyStatus(rng, &p, Burn)
	if p.SleepTurns != 0 {
		t.Errorf("sleep turns = %d after burn, want 0", p.SleepTurns)
	}
}

// 同じseedからは同じ結果になる。
func TestStatusRNGIsReproducible(t *testing.T) {
	t.Parallel()

	run := func() []int {
		rng := NewRand(20260918)
		results := make([]int, 0, 60)
		for i := 0; i < 30; i++ {
			results = append(results, SleepDuration(rng))
			blocked := 0
			if IsFullyParalyzed(rng) {
				blocked = 1
			}
			results = append(results, blocked)
		}
		return results
	}

	first, second := run(), run()
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("result %d: %d != %d", i, first[i], second[i])
		}
	}
}
