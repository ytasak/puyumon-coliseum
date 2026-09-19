package balance

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
	"github.com/ytasak/puyumon-coliseum/internal/simulation"
)

// testConfig はtestで回すbatchの設定。
//
// 20×20のmatchupを2 seedずつなので800対戦になる。1対戦が0.1msほどなので、
// 1本のtestは0.1秒ほどで終わる。
func testConfig() Config {
	return Config{FirstSeed: 1, Trials: 2, MaxTurns: 200}
}

// runForTest はtest用のbatchを1回実行する。
func runForTest(t *testing.T, cfg Config) Report {
	t.Helper()

	report, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run(%+v) error = %v", cfg, err)
	}
	return report
}

// TestRunCoversEveryOrderedMatchup は20×20の総当たりを、指定した試行数だけ回すことを確かめる。
func TestRunCoversEveryOrderedMatchup(t *testing.T) {
	cfg := testConfig()
	report := runForTest(t, cfg)

	combinations := roster.Combinations()
	if len(report.Combinations) != len(combinations) {
		t.Fatalf("combinationの数 = %d, want %d", len(report.Combinations), len(combinations))
	}
	if want := len(combinations) * len(combinations) * cfg.Trials; report.Battles != want {
		t.Errorf("Battles = %d, want %d", report.Battles, want)
	}

	if len(report.Matchups) != len(combinations) {
		t.Fatalf("matrixの行数 = %d, want %d", len(report.Matchups), len(combinations))
	}
	for i, row := range report.Matchups {
		if len(row) != len(combinations) {
			t.Fatalf("matrix[%d]の列数 = %d, want %d", i, len(row), len(combinations))
		}
		for j, matchup := range row {
			if matchup.First != i || matchup.Second != j {
				t.Errorf("matrix[%d][%d] = (%d,%d), want (%d,%d)", i, j, matchup.First, matchup.Second, i, j)
			}
			if matchup.Outcomes.Battles != cfg.Trials {
				t.Errorf("matrix[%d][%d] の対戦数 = %d, want %d", i, j, matchup.Outcomes.Battles, cfg.Trials)
			}
		}
	}

	for i, summary := range report.Combinations {
		if summary.Team != combinations[i] {
			t.Errorf("Combinations[%d].Team = %v, want %v", i, summary.Team, combinations[i])
		}
	}
}

// addOutcomes は2つの内訳を足す。
func addOutcomes(a, b Outcomes) Outcomes {
	return Outcomes{
		Battles:    a.Battles + b.Battles,
		Wins:       a.Wins + b.Wins,
		Losses:     a.Losses + b.Losses,
		Draws:      a.Draws + b.Draws,
		Unresolved: a.Unresolved + b.Unresolved,
	}
}

// flipOutcomes は反対側から見た内訳を返す。
func flipOutcomes(o Outcomes) Outcomes {
	o.Wins, o.Losses = o.Losses, o.Wins
	return o
}

// TestRunSummariesMatchTheMatchupMatrix は、side別・combination別・species別の集計が
// matchup matrixから足し直した数と一致することを確かめる。
//
// matrixは対戦結果そのものなので、集計側が数え落としや二重計上をしていれば食い違う。
func TestRunSummariesMatchTheMatchupMatrix(t *testing.T) {
	report := runForTest(t, testConfig())

	var asFirst Outcomes
	perCombination := make([][2]Outcomes, len(report.Combinations))
	perSpecies := make([][2]Outcomes, len(report.Species)) // [0]=含むteam側 [1]=含まないteam側

	for i, row := range report.Matchups {
		for j, matchup := range row {
			first := matchup.Outcomes
			second := flipOutcomes(first)

			asFirst = addOutcomes(asFirst, first)
			perCombination[i][battle.Player1] = addOutcomes(perCombination[i][battle.Player1], first)
			perCombination[j][battle.Player2] = addOutcomes(perCombination[j][battle.Player2], second)

			for s := range report.Species {
				species := report.Species[s].Species
				for _, side := range []struct {
					team     [battle.TeamSize]battle.SpeciesID
					outcomes Outcomes
				}{
					{report.Combinations[i].Team, first},
					{report.Combinations[j].Team, second},
				} {
					index := 1
					for _, member := range side.team {
						if member == species {
							index = 0
							break
						}
					}
					perSpecies[s][index] = addOutcomes(perSpecies[s][index], side.outcomes)
				}
			}
		}
	}

	if report.Sides[battle.Player1] != asFirst {
		t.Errorf("Sides[player1] = %+v, want %+v", report.Sides[battle.Player1], asFirst)
	}
	if want := flipOutcomes(asFirst); report.Sides[battle.Player2] != want {
		t.Errorf("Sides[player2] = %+v, want %+v", report.Sides[battle.Player2], want)
	}

	for i, summary := range report.Combinations {
		for _, side := range []battle.Side{battle.Player1, battle.Player2} {
			if summary.Sides[side] != perCombination[i][side] {
				t.Errorf("Combinations[%d].Sides[%v] = %+v, want %+v",
					i, side, summary.Sides[side], perCombination[i][side])
			}
		}
		want := addOutcomes(perCombination[i][battle.Player1], perCombination[i][battle.Player2])
		if summary.Total != want {
			t.Errorf("Combinations[%d].Total = %+v, want %+v", i, summary.Total, want)
		}
	}

	for s, summary := range report.Species {
		if summary.With != perSpecies[s][0] {
			t.Errorf("Species[%v].With = %+v, want %+v", summary.Species, summary.With, perSpecies[s][0])
		}
		if summary.Without != perSpecies[s][1] {
			t.Errorf("Species[%v].Without = %+v, want %+v", summary.Species, summary.Without, perSpecies[s][1])
		}
	}
}

// TestRunSummarisesEverySpecies はキャラクター別の集計が6体分あり、
// 含むteamと含まないteamを1対戦につき両side数えていることを確かめる。
func TestRunSummarisesEverySpecies(t *testing.T) {
	report := runForTest(t, testConfig())

	characters := roster.All()
	if len(report.Species) != len(characters) {
		t.Fatalf("speciesの数 = %d, want %d", len(report.Species), len(characters))
	}

	for s, summary := range report.Species {
		if summary.Species != characters[s].ID {
			t.Errorf("Species[%d] = %v, want %v（roster.All()と同じ並び）", s, summary.Species, characters[s].ID)
		}

		teams := 0
		for _, combination := range roster.Combinations() {
			for _, member := range combination {
				if member == summary.Species {
					teams++
					break
				}
			}
		}
		if summary.Teams != teams {
			t.Errorf("Species[%v].Teams = %d, want %d", summary.Species, summary.Teams, teams)
		}

		// 1対戦につき両sideを1件ずつ数えるので、合計は対戦数の2倍になる。
		if got, want := summary.With.Battles+summary.Without.Battles, report.Battles*2; got != want {
			t.Errorf("Species[%v] の件数 = %d, want %d", summary.Species, got, want)
		}
	}
}

// TestRunReportsAssignedLevels は実際に配られたLevelをReportへ載せることを確かめる。
//
// 勝率をLevel 55の割り当てと結び付けて読めるようにするため。
func TestRunReportsAssignedLevels(t *testing.T) {
	report := runForTest(t, testConfig())

	for i, summary := range report.Combinations {
		want, err := roster.AssignLevels(summary.Team)
		if err != nil {
			t.Fatalf("roster.AssignLevels(%v) error = %v", summary.Team, err)
		}
		if summary.Levels != want {
			t.Errorf("Combinations[%d].Levels = %v, want %v", i, summary.Levels, want)
		}

		total := 0
		for _, level := range summary.Levels {
			total += level
		}
		if total != roster.LevelTotal {
			t.Errorf("Combinations[%d] のLevel合計 = %d, want %d", i, total, roster.LevelTotal)
		}
	}
}

// TestRunReportsTurnDistribution はturn分布が対戦数と噛み合っていることを確かめる。
func TestRunReportsTurnDistribution(t *testing.T) {
	cfg := testConfig()
	report := runForTest(t, cfg)

	// 決着した対戦だけを数える。未決着はUnresolvedで別に数える。
	if want := report.Battles - report.Sides[battle.Player1].Unresolved; report.Turns.Battles != want {
		t.Errorf("Turns.Battles = %d, want %d", report.Turns.Battles, want)
	}
	if report.Turns.Battles == 0 {
		t.Fatal("決着した対戦が1つも無い")
	}
	if report.Turns.Min < 1 || report.Turns.Max > cfg.MaxTurns {
		t.Errorf("turn数の範囲 = %d..%d, want 1..%d", report.Turns.Min, report.Turns.Max, cfg.MaxTurns)
	}
	if mean := report.Turns.Mean(); mean < float64(report.Turns.Min) || mean > float64(report.Turns.Max) {
		t.Errorf("Mean() = %v, want %d..%d の範囲", mean, report.Turns.Min, report.Turns.Max)
	}

	counted := 0
	previous := 0
	for i, bucket := range report.Turns.Buckets {
		if bucket.Min != previous+1 {
			t.Errorf("Buckets[%d].Min = %d, want %d（区間が連続していない）", i, bucket.Min, previous+1)
		}
		previous = bucket.Max
		counted += bucket.Battles
	}
	if previous != cfg.MaxTurns {
		t.Errorf("最後の区間の上端 = %d, want %d", previous, cfg.MaxTurns)
	}
	if counted != report.Turns.Battles {
		t.Errorf("区間の合計 = %d, want %d", counted, report.Turns.Battles)
	}
}

// TestRunCountsUnresolvedSeparately はturn上限に達した対戦を、
// 勝敗にも引き分けにもせず、turn分布からも外すことを確かめる。
func TestRunCountsUnresolvedSeparately(t *testing.T) {
	cfg := testConfig()
	cfg.MaxTurns = 2 // 3体を倒しきれない短さ
	report := runForTest(t, cfg)

	unresolved := report.Sides[battle.Player1].Unresolved
	if unresolved == 0 {
		t.Fatal("上限2 turnでも未決着が1つも無い")
	}
	if got := report.Sides[battle.Player2].Unresolved; got != unresolved {
		t.Errorf("player2側の未決着 = %d, want %d（両者に共通のはず）", got, unresolved)
	}
	if want := report.Battles - unresolved; report.Turns.Battles != want {
		t.Errorf("Turns.Battles = %d, want %d（未決着を含めない）", report.Turns.Battles, want)
	}
	if got, want := report.Sides[battle.Player1].UnresolvedRate(), float64(unresolved)/float64(report.Battles); got != want {
		t.Errorf("UnresolvedRate() = %v, want %v", got, want)
	}
}

// TestRunUsesEverySeedInTheRange は、Trials回の試行がFirstSeedから1ずつ増やしたseedで
// 回っていることを確かめる。
//
// 試行ごとにseedを進めていないと、同じ対戦をTrials回繰り返すだけになる。
// それでも決定性のtestは通ってしまうので、seedの配り方そのものを固定する。
func TestRunUsesEverySeedInTheRange(t *testing.T) {
	cfg := testConfig()
	cfg.Trials = 2

	both := runForTest(t, cfg)
	first := runForTest(t, Config{FirstSeed: cfg.FirstSeed, Trials: 1, MaxTurns: cfg.MaxTurns})
	second := runForTest(t, Config{FirstSeed: cfg.FirstSeed + 1, Trials: 1, MaxTurns: cfg.MaxTurns})

	for i, row := range both.Matchups {
		for j, matchup := range row {
			want := addOutcomes(first.Matchups[i][j].Outcomes, second.Matchups[i][j].Outcomes)
			if matchup.Outcomes != want {
				t.Errorf("matrix[%d][%d] = %+v, want %+v（seed %d と %d の合計）",
					i, j, matchup.Outcomes, want, cfg.FirstSeed, cfg.FirstSeed+1)
			}
		}
	}
}

// TestRunIsDeterministic は同じConfigから同じReportが再生成できることを確かめる。
func TestRunIsDeterministic(t *testing.T) {
	cfg := testConfig()

	first := runForTest(t, cfg)
	second := runForTest(t, cfg)

	if !reflect.DeepEqual(first, second) {
		t.Error("同じConfigの2回目のReportが一致しない")
	}
}

// TestRunSeedChangesTheReport はseedを変えれば結果が変わることを確かめる。
//
// seedを無視して同じ対戦を繰り返していないことの確認。
func TestRunSeedChangesTheReport(t *testing.T) {
	cfg := testConfig()
	other := cfg
	other.FirstSeed = cfg.FirstSeed + uint64(cfg.Trials)

	if reflect.DeepEqual(runForTest(t, cfg), runForTest(t, other)) {
		t.Error("seedを変えてもReportが変わらない")
	}
}

// TestRunNormalisesConfig は省略した値を埋めたConfigをReportへ載せることを確かめる。
//
// Reportに載っている値をそのまま渡せば再現できる状態にしておく。
func TestRunNormalisesConfig(t *testing.T) {
	cfg := Config{FirstSeed: 7, Trials: 1}
	report := runForTest(t, cfg)

	want := Config{FirstSeed: 7, Trials: 1, Policy: DefaultPolicy, MaxTurns: simulation.DefaultMaxTurns}
	if report.Config != want {
		t.Errorf("Report.Config = %+v, want %+v", report.Config, want)
	}
	if got := runForTest(t, report.Config); !reflect.DeepEqual(got, report) {
		t.Error("Reportに載っているConfigで再実行したら結果が変わった")
	}
}

// TestRunRecordsThePolicy は使った方針をReportへ残すことを確かめる。
//
// 方針が違えば分布も違う。何で測ったかがReportから分からないと、
// あとから数字を読み違える。
func TestRunRecordsThePolicy(t *testing.T) {
	cfg := testConfig()
	cfg.Policy = PolicyUniformUsable

	if got := runForTest(t, cfg).Config.Policy; got != PolicyUniformUsable {
		t.Errorf("Report.Config.Policy = %q, want %q", got, PolicyUniformUsable)
	}
}

// TestRunDefaultsToFirstUsable は方針を省略したときにFirstUsableで回ることを確かめる。
//
// YTA-23のbaselineは方針を指定せずに測ったので、省略時の意味が変わると再現できなくなる。
func TestRunDefaultsToFirstUsable(t *testing.T) {
	omitted := testConfig()
	omitted.Policy = ""
	explicit := testConfig()
	explicit.Policy = PolicyFirstUsable

	report := runForTest(t, omitted)
	if report.Config.Policy != PolicyFirstUsable {
		t.Errorf("Report.Config.Policy = %q, want %q", report.Config.Policy, PolicyFirstUsable)
	}
	if !reflect.DeepEqual(report, runForTest(t, explicit)) {
		t.Error("方針を省略した場合とFirstUsableを指定した場合で結果が違う")
	}
}

// TestRunPolicyChangesTheReport は方針を変えれば結果が変わることを確かめる。
func TestRunPolicyChangesTheReport(t *testing.T) {
	first := testConfig()
	uniform := testConfig()
	uniform.Policy = PolicyUniformUsable

	if reflect.DeepEqual(runForTest(t, first), runForTest(t, uniform)) {
		t.Error("方針を変えてもReportが変わらない")
	}
}

// TestRunUniformPolicyIsDeterministic はUniformUsableでも同じConfigから
// 同じReportが再生成できることを確かめる。
func TestRunUniformPolicyIsDeterministic(t *testing.T) {
	cfg := testConfig()
	cfg.Policy = PolicyUniformUsable

	if !reflect.DeepEqual(runForTest(t, cfg), runForTest(t, cfg)) {
		t.Error("同じConfigの2回目のReportが一致しない")
	}
}

// TestRunRejectsUnknownPolicy は定義にない方針をerrorにすることを確かめる。
//
// 黙ってFirstUsableで回すと、別の方針で測ったつもりの数字が出てくる。
func TestRunRejectsUnknownPolicy(t *testing.T) {
	cfg := testConfig()
	cfg.Policy = "greedy"

	if _, err := Run(cfg); !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("Run() error = %v, want %v", err, ErrInvalidConfig)
	}
}

// TestPolicySeedIsSeparatedFromTheBattleSeed は行動選択のseedが
// 対戦のseedとも、もう一方のsideとも重ならないことを確かめる。
func TestPolicySeedIsSeparatedFromTheBattleSeed(t *testing.T) {
	seen := map[uint64]string{}
	for battleSeed := uint64(0); battleSeed < 1000; battleSeed++ {
		for _, side := range sides {
			seed := policySeed(battleSeed, side)
			if seed == battleSeed {
				t.Fatalf("policySeed(%d, %v) が対戦のseedと同じ", battleSeed, side)
			}
			where := fmt.Sprintf("battleSeed %d の %v", battleSeed, side)
			if previous, ok := seen[seed]; ok {
				t.Fatalf("%s のseedが %s と重複している", where, previous)
			}
			seen[seed] = where
		}
	}
}

// TestRunRejectsInvalidConfig は試行数と上限の取り得ない値をerrorにすることを確かめる。
func TestRunRejectsInvalidConfig(t *testing.T) {
	for _, tt := range []struct {
		name string
		cfg  Config
	}{
		{"試行数が0", Config{}},
		{"試行数が負", Config{Trials: -1}},
		{"上限が負", Config{Trials: 1, MaxTurns: -1}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Run(tt.cfg); !errors.Is(err, ErrInvalidConfig) {
				t.Errorf("Run(%+v) error = %v, want %v", tt.cfg, err, ErrInvalidConfig)
			}
		})
	}
}
