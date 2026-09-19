package balance

import "testing"

// TestOutcomesRates は割合の分母の取り方を固定する。
func TestOutcomesRates(t *testing.T) {
	outcomes := Outcomes{Battles: 10, Wins: 4, Losses: 3, Draws: 1, Unresolved: 2}

	if got, want := outcomes.WinRate(), 0.4; got != want {
		t.Errorf("WinRate() = %v, want %v", got, want)
	}
	// 未決着の2件だけを分母から除く。引き分けは決着なので残る。
	if got, want := outcomes.ResolvedWinRate(), 0.5; got != want {
		t.Errorf("ResolvedWinRate() = %v, want %v", got, want)
	}
	if got, want := outcomes.UnresolvedRate(), 0.2; got != want {
		t.Errorf("UnresolvedRate() = %v, want %v", got, want)
	}
}

// TestOutcomesRatesWithoutBattles は1件も数えていないときに0除算しないことを確かめる。
func TestOutcomesRatesWithoutBattles(t *testing.T) {
	var outcomes Outcomes

	for _, check := range []struct {
		name string
		got  float64
	}{
		{"WinRate", outcomes.WinRate()},
		{"ResolvedWinRate", outcomes.ResolvedWinRate()},
		{"UnresolvedRate", outcomes.UnresolvedRate()},
	} {
		if check.got != 0 {
			t.Errorf("%s() = %v, want 0", check.name, check.got)
		}
	}
}

// TestOutcomesAdd は結果の内訳の数え方を確かめる。
func TestOutcomesAdd(t *testing.T) {
	var outcomes Outcomes
	for _, result := range []outcome{win, win, loss, draw, unresolved} {
		outcomes.add(result)
	}

	want := Outcomes{Battles: 5, Wins: 2, Losses: 1, Draws: 1, Unresolved: 1}
	if outcomes != want {
		t.Errorf("add後のOutcomes = %+v, want %+v", outcomes, want)
	}
}

// TestOutcomeFlip は反対側から見た結果を確かめる。
//
// 引き分けと未決着は両者に共通なので裏返らない。
func TestOutcomeFlip(t *testing.T) {
	for _, tt := range []struct {
		from outcome
		want outcome
	}{
		{win, loss},
		{loss, win},
		{draw, draw},
		{unresolved, unresolved},
	} {
		if got := tt.from.flip(); got != tt.want {
			t.Errorf("outcome(%d).flip() = %d, want %d", tt.from, got, tt.want)
		}
	}
}

// TestNewTurnStatsBuckets は区間の切り方を固定する。
//
// 上限が幅で割り切れない場合、最後の区間だけ短くなる。
func TestNewTurnStatsBuckets(t *testing.T) {
	stats := newTurnStats(25)

	want := []TurnBucket{{Min: 1, Max: 10}, {Min: 11, Max: 20}, {Min: 21, Max: 25}}
	if len(stats.Buckets) != len(want) {
		t.Fatalf("区間の数 = %d, want %d", len(stats.Buckets), len(want))
	}
	for i, bucket := range stats.Buckets {
		if bucket != want[i] {
			t.Errorf("Buckets[%d] = %+v, want %+v", i, bucket, want[i])
		}
	}
}

// TestTurnStatsAdd はturn数を区間へ振り分け、最小・最大・平均を保つことを確かめる。
func TestTurnStatsAdd(t *testing.T) {
	stats := newTurnStats(25)
	for _, turns := range []int{10, 1, 11, 25} {
		stats.add(turns)
	}

	if stats.Battles != 4 {
		t.Errorf("Battles = %d, want 4", stats.Battles)
	}
	if stats.Min != 1 || stats.Max != 25 {
		t.Errorf("Min/Max = %d/%d, want 1/25", stats.Min, stats.Max)
	}
	if got, want := stats.Mean(), 47.0/4; got != want {
		t.Errorf("Mean() = %v, want %v", got, want)
	}

	want := []int{2, 1, 1}
	for i, bucket := range stats.Buckets {
		if bucket.Battles != want[i] {
			t.Errorf("Buckets[%d].Battles = %d, want %d", i, bucket.Battles, want[i])
		}
	}
}

// TestTurnStatsMeanWithoutBattles は1件も数えていないときに0除算しないことを確かめる。
func TestTurnStatsMeanWithoutBattles(t *testing.T) {
	stats := newTurnStats(25)

	if got := stats.Mean(); got != 0 {
		t.Errorf("Mean() = %v, want 0", got)
	}
	if stats.Min != 0 || stats.Max != 0 {
		t.Errorf("Min/Max = %d/%d, want 0/0", stats.Min, stats.Max)
	}
}

// TestTurnStatsAddClampsOutOfRange は区間の外のturn数でも取りこぼさないことを確かめる。
//
// 上限を超えるturn数はsimulationが返さないが、区間の外を黙って捨てると
// 合計が合わなくなる。両端の区間へ寄せる。
func TestTurnStatsAddClampsOutOfRange(t *testing.T) {
	stats := newTurnStats(25)
	stats.add(30)

	if got := stats.Buckets[len(stats.Buckets)-1].Battles; got != 1 {
		t.Errorf("最後の区間 = %d, want 1", got)
	}
	if stats.Battles != 1 {
		t.Errorf("Battles = %d, want 1", stats.Battles)
	}
}
