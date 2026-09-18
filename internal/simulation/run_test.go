package simulation

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
)

// goldenSeed は代表scenarioで使うseed。
//
// 急所・ねむり・こおりが1本の対戦へ収まり、11turnで決着するものを選んだ。
const goldenSeed = 9

// move はscriptを読みやすくするための短縮。
func move(slot int) battle.Action { return battle.MoveAction{Slot: slot} }

// goldenConfig は代表scenarioの設定を返す。
//
// 両者の行動をscriptで固定して、次を1本の対戦へ入れている。
//
//   - 反動（overdrive）… YTA-19の特殊挙動
//   - ねむり付与（spores）と、目覚めたturnに動けないGeneration Iの挙動
//   - 自分の意思による交代と、戦闘不能後の交代
//   - 急所
//   - こおり付与（icestorm）
//
// script が尽きたあとはFirstUsableが引き継ぐので、決着まで走る。
func goldenConfig() Config {
	return Config{
		Teams: [2][battle.TeamSize]battle.SpeciesID{
			{roster.SpeciesBull, roster.SpeciesStar, roster.SpeciesJolt},
			{roster.SpeciesPalm, roster.SpeciesCharm, roster.SpeciesWhale},
		},
		Seed: goldenSeed,
		Choosers: [2]Chooser{
			&Script{Actions: []battle.Action{
				move(1),                        // overdrive。次のturnは反動で動けない
				move(0),                        // 反動で流れるので、指定した技は使われない
				move(0),                        // ねむりで動けない
				battle.SwitchAction{Target: 1}, // ねむったbullを下げる
				move(0), move(0), move(0), move(0),
				move(0), move(0), move(0), move(0),
			}},
			&Script{Actions: []battle.Action{
				move(1), // mindblast
				move(0), // spores。bullをねむらせる
				move(1), move(1), move(1), move(1),
				move(1), move(1), move(1), move(1),
				move(1), move(1),
			}},
		},
		MaxTurns: 20,
	}
}

// goldenEvents は代表scenarioで起きるEventの全文。
//
// 個々の数値の正しさはYTA-16〜YTA-19の独立goldenが担保する。
// ここで固定するのは、mechanicsを通した結果が意図せず変わったことを検出するため。
var goldenEvents = []string{
	"MoveUsed{Side:player1 Slot:1 Move:overdrive}",
	"Damage{Side:player2 Amount:105 RemainingHP:96}",
	"MoveUsed{Side:player2 Slot:1 Move:mindblast}",
	"Damage{Side:player1 Amount:76 RemainingHP:105}",
	"Recharge{Side:player1}",
	"MoveUsed{Side:player2 Slot:0 Move:spores}",
	"StatusApplied{Side:player1 Status:sleep}",
	"ActionBlocked{Side:player1 Reason:sleep}",
	"MoveUsed{Side:player2 Slot:1 Move:mindblast}",
	"Damage{Side:player1 Amount:86 RemainingHP:19}",
	"Switched{Side:player1 From:0 To:1}",
	"MoveUsed{Side:player2 Slot:1 Move:mindblast}",
	"Damage{Side:player1 Amount:30 RemainingHP:136}",
	"StatStageChanged{Side:player1 Stat:special Delta:-1}",
	"MoveUsed{Side:player1 Slot:0 Move:tide}",
	"Damage{Side:player2 Amount:15 RemainingHP:81}",
	"MoveUsed{Side:player2 Slot:1 Move:mindblast}",
	"Damage{Side:player1 Amount:52 RemainingHP:84}",
	"StatStageChanged{Side:player1 Stat:special Delta:-1}",
	"MoveUsed{Side:player1 Slot:0 Move:tide}",
	"Damage{Side:player2 Amount:13 RemainingHP:68}",
	"MoveUsed{Side:player2 Slot:1 Move:mindblast}",
	"Damage{Side:player1 Amount:60 RemainingHP:24}",
	"MoveUsed{Side:player1 Slot:0 Move:tide}",
	"CriticalHit{Side:player2}",
	"Damage{Side:player2 Amount:48 RemainingHP:20}",
	"MoveUsed{Side:player2 Slot:1 Move:mindblast}",
	"Damage{Side:player1 Amount:24 RemainingHP:0}",
	"Fainted{Side:player1 Index:1}",
	"Switched{Side:player1 From:1 To:0}",
	"StatusRecovered{Side:player1 Status:sleep}",
	"ActionBlocked{Side:player1 Reason:wake_up}",
	"MoveUsed{Side:player2 Slot:1 Move:mindblast}",
	"Damage{Side:player1 Amount:19 RemainingHP:0}",
	"Fainted{Side:player1 Index:0}",
	"Switched{Side:player1 From:0 To:2}",
	"MoveUsed{Side:player1 Slot:0 Move:spark}",
	"Damage{Side:player2 Amount:20 RemainingHP:0}",
	"Fainted{Side:player2 Index:0}",
	"Switched{Side:player2 From:0 To:1}",
	"MoveUsed{Side:player1 Slot:0 Move:spark}",
	"Damage{Side:player2 Amount:77 RemainingHP:110}",
	"MoveUsed{Side:player2 Slot:1 Move:icestorm}",
	"CriticalHit{Side:player1}",
	"Damage{Side:player1 Amount:132 RemainingHP:55}",
	"StatusApplied{Side:player1 Status:freeze}",
	"ActionBlocked{Side:player1 Reason:freeze}",
	"MoveUsed{Side:player2 Slot:1 Move:icestorm}",
	"Damage{Side:player1 Amount:55 RemainingHP:0}",
	"Fainted{Side:player1 Index:2}",
}

// goldenFinalState は代表scenarioの決着時のBattleState。
var goldenFinalState = []string{
	"turn=12 status=player2_won",
	"player1 active=2",
	"  [0] bull Lv50 HP=0/181 status=none sleep=0 stages={Attack:0 Defense:0 Speed:0 Special:0 Accuracy:0 Evasion:0} recharging=false continuing={Move: TurnsLeft:0} moves=[{Move:slam PP:15 MaxPP:15} {Move:overdrive PP:4 MaxPP:5} {Move:quake PP:10 MaxPP:10} {Move:icestorm PP:5 MaxPP:5}]",
	"  [1] star Lv50 HP=0/166 status=none sleep=0 stages={Attack:0 Defense:0 Speed:0 Special:0 Accuracy:0 Evasion:0} recharging=false continuing={Move: TurnsLeft:0} moves=[{Move:tide PP:12 MaxPP:15} {Move:mindblast PP:10 MaxPP:10} {Move:spark PP:15 MaxPP:15} {Move:mend PP:20 MaxPP:20}]",
	"  [2] jolt Lv55 HP=0/187 status=freeze sleep=0 stages={Attack:0 Defense:0 Speed:0 Special:0 Accuracy:0 Evasion:0} recharging=false continuing={Move: TurnsLeft:0} moves=[{Move:spark PP:13 MaxPP:15} {Move:numb PP:20 MaxPP:20} {Move:needles PP:20 MaxPP:20} {Move:dash PP:30 MaxPP:30}]",
	"player2 active=1",
	"  [0] palm Lv50 HP=0/201 status=none sleep=0 stages={Attack:0 Defense:0 Speed:0 Special:0 Accuracy:0 Evasion:0} recharging=false continuing={Move: TurnsLeft:0} moves=[{Move:spores PP:14 MaxPP:15} {Move:mindblast PP:3 MaxPP:10} {Move:drain PP:10 MaxPP:10} {Move:burst PP:5 MaxPP:5}]",
	"  [1] charm Lv55 HP=110/187 status=none sleep=0 stages={Attack:0 Defense:0 Speed:0 Special:0 Accuracy:0 Evasion:0} recharging=false continuing={Move: TurnsLeft:0} moves=[{Move:slumber PP:10 MaxPP:10} {Move:icestorm PP:3 MaxPP:5} {Move:mindblast PP:10 MaxPP:10} {Move:doze PP:10 MaxPP:10}]",
	"  [2] whale Lv50 HP=236/236 status=none sleep=0 stages={Attack:0 Defense:0 Speed:0 Special:0 Accuracy:0 Evasion:0} recharging=false continuing={Move: TurnsLeft:0} moves=[{Move:tide PP:15 MaxPP:15} {Move:icestorm PP:5 MaxPP:5} {Move:slam PP:15 MaxPP:15} {Move:doze PP:10 MaxPP:10}]",
}

// formatEvent はEventを1行で表す。structのすべてのfieldが出るので、
// fieldが増えたり値が変わったりすればgoldenと食い違う。
func formatEvent(event battle.Event) string {
	return fmt.Sprintf("%s%+v", strings.TrimPrefix(fmt.Sprintf("%T", event), "battle."), event)
}

// formatState はBattleStateを行に展開する。対戦中に変わる値をすべて含める。
func formatState(state battle.BattleState) []string {
	lines := []string{fmt.Sprintf("turn=%d status=%v", state.Turn, state.Status)}
	for _, side := range sides {
		player := state.Players[side]
		lines = append(lines, fmt.Sprintf("%v active=%d", side, player.Active))
		for i, mon := range player.Team {
			lines = append(lines, fmt.Sprintf(
				"  [%d] %v Lv%d HP=%d/%d status=%v sleep=%d stages=%+v recharging=%v continuing=%+v moves=%+v",
				i, mon.Species, mon.Level, mon.CurrentHP, mon.Stats.HP, mon.Status, mon.SleepTurns,
				mon.Stages, mon.Recharging, mon.ContinuingMove, mon.Moves))
		}
	}
	return lines
}

// compareLines は行ごとに比較して、食い違った場所だけを報告する。
func compareLines(t *testing.T, label string, got, want []string) {
	t.Helper()

	for i := 0; i < len(got) && i < len(want); i++ {
		if got[i] != want[i] {
			t.Errorf("%s[%d] = %s, want %s", label, i, got[i], want[i])
		}
	}
	if len(got) != len(want) {
		t.Errorf("%s の行数 = %d, want %d", label, len(got), len(want))
		for i := len(want); i < len(got); i++ {
			t.Errorf("%s[%d] 余分: %s", label, i, got[i])
		}
		for i := len(got); i < len(want); i++ {
			t.Errorf("%s[%d] 不足: %s", label, i, want[i])
		}
	}
}

// TestRunGoldenScenario は代表scenarioのEvent全文とfinal BattleStateを固定する。
func TestRunGoldenScenario(t *testing.T) {
	result, err := Run(goldenConfig())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	events := make([]string, len(result.Events))
	for i, event := range result.Events {
		events[i] = formatEvent(event)
	}
	compareLines(t, "Events", events, goldenEvents)
	compareLines(t, "Final", formatState(result.Final), goldenFinalState)

	if result.Turns != 11 {
		t.Errorf("Turns = %d, want 11", result.Turns)
	}
	if result.TurnLimitReached {
		t.Error("TurnLimitReached = true, want false（上限前に決着する）")
	}
}

// TestGoldenScenarioCoversRequiredBehaviour は代表scenarioが
// switch / status / critical / faint / YTA-19の特殊挙動を含むことを確かめる。
//
// goldenを更新するときに、これらが抜け落ちたscenarioへ差し替わるのを防ぐ。
func TestGoldenScenarioCoversRequiredBehaviour(t *testing.T) {
	result, err := Run(goldenConfig())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	var (
		switched   bool
		status     bool
		critical   bool
		fainted    bool
		recharging bool // YTA-19のoverdriveの反動
	)
	for _, event := range result.Events {
		switch event.(type) {
		case battle.Switched:
			switched = true
		case battle.StatusApplied:
			status = true
		case battle.CriticalHit:
			critical = true
		case battle.Fainted:
			fainted = true
		case battle.Recharge:
			recharging = true
		}
	}

	for _, check := range []struct {
		name string
		got  bool
	}{
		{"switch", switched},
		{"status", status},
		{"critical", critical},
		{"faint", fainted},
		{"recharge", recharging},
	} {
		if !check.got {
			t.Errorf("代表scenarioに %s が含まれていない", check.name)
		}
	}
}

// TestRunIsDeterministic は同じConfigから常に同じ結果が出ることを確かめる。
//
// goldenの値ではなく、2回の実行が完全に一致することで再現性を見る。
func TestRunIsDeterministic(t *testing.T) {
	first, err := Run(goldenConfig())
	if err != nil {
		t.Fatalf("1回目のRun() error = %v", err)
	}
	second, err := Run(goldenConfig())
	if err != nil {
		t.Fatalf("2回目のRun() error = %v", err)
	}

	if first.Final != second.Final {
		t.Error("2回の実行でfinal BattleStateが一致しない")
	}
	if first.Turns != second.Turns || first.TurnLimitReached != second.TurnLimitReached {
		t.Errorf("turn数が一致しない: %d/%v と %d/%v",
			first.Turns, first.TurnLimitReached, second.Turns, second.TurnLimitReached)
	}
	if len(first.Events) != len(second.Events) {
		t.Fatalf("Event数が一致しない: %d と %d", len(first.Events), len(second.Events))
	}
	for i := range first.Events {
		if first.Events[i] != second.Events[i] {
			t.Errorf("Events[%d] が一致しない: %v と %v", i, first.Events[i], second.Events[i])
		}
	}
}

// TestRunDifferentSeed は別のseedでは別の結果になることを確かめる。
//
// seedを無視して固定の乱数を引いていないことの確認。
func TestRunDifferentSeed(t *testing.T) {
	base, err := Run(goldenConfig())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	other := goldenConfig()
	other.Seed = goldenSeed + 1
	changed, err := Run(other)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if base.Final == changed.Final && len(base.Events) == len(changed.Events) {
		t.Error("seedを変えても結果が変わらない")
	}
}

// TestRunStopsAtTurnLimit は上限で止めたとき、勝敗を作らないことを確かめる。
func TestRunStopsAtTurnLimit(t *testing.T) {
	cfg := goldenConfig()
	cfg.MaxTurns = 3

	result, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Turns != 3 {
		t.Errorf("Turns = %d, want 3", result.Turns)
	}
	if !result.TurnLimitReached {
		t.Error("TurnLimitReached = false, want true")
	}
	if result.Final.Status != battle.Ongoing {
		t.Errorf("Status = %v, want %v（上限では勝敗を決めない）", result.Final.Status, battle.Ongoing)
	}
}

// TestRunAllCombinations は20通りの組み合わせ同士を総当たりで完走させる。
//
// Event列は固定しない。errorなく走り切ること、turn数が上限を超えないこと、
// 上限に達した場合はTurnLimitReachedで分かることだけを確かめる。
func TestRunAllCombinations(t *testing.T) {
	combinations := roster.Combinations()
	if len(combinations) != 20 {
		t.Fatalf("組み合わせ数 = %d, want 20", len(combinations))
	}

	const maxTurns = 200
	unresolved := 0
	for i, first := range combinations {
		for j, second := range combinations {
			result, err := Run(Config{
				Teams:    [2][battle.TeamSize]battle.SpeciesID{first, second},
				Seed:     uint64(i*len(combinations) + j),
				MaxTurns: maxTurns,
			})
			if err != nil {
				t.Fatalf("Run(%v vs %v) error = %v", first, second, err)
			}
			if result.Turns > maxTurns {
				t.Fatalf("Run(%v vs %v) のturn数 = %d, 上限 %d を超えた", first, second, result.Turns, maxTurns)
			}
			if result.TurnLimitReached {
				unresolved++
				if result.Final.Status != battle.Ongoing {
					t.Fatalf("Run(%v vs %v) は未決着なのにStatus = %v", first, second, result.Final.Status)
				}
				continue
			}
			if result.Final.Status == battle.Ongoing {
				t.Fatalf("Run(%v vs %v) は決着したのにStatus = %v", first, second, result.Final.Status)
			}
		}
	}
	t.Logf("%d対戦のうち上限で未決着になったのは %d", len(combinations)*len(combinations), unresolved)
}

// TestRunRejectsUnknownCharacter は定義にないキャラクターをerrorにすることを確かめる。
func TestRunRejectsUnknownCharacter(t *testing.T) {
	cfg := goldenConfig()
	cfg.Teams[battle.Player2][0] = "nobody"

	if _, err := Run(cfg); !errors.Is(err, roster.ErrUnknownCharacter) {
		t.Errorf("Run() error = %v, want %v", err, roster.ErrUnknownCharacter)
	}
}

// TestRunUsesDefaultMaxTurns はMaxTurnsを省略しても止まることを確かめる。
func TestRunUsesDefaultMaxTurns(t *testing.T) {
	cfg := goldenConfig()
	cfg.MaxTurns = 0

	result, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Turns > DefaultMaxTurns {
		t.Errorf("Turns = %d, DefaultMaxTurns %d を超えた", result.Turns, DefaultMaxTurns)
	}
}

// TestCombinationsAreUnique は20通りの組み合わせに重複が無いことを確かめる。
func TestCombinationsAreUnique(t *testing.T) {
	seen := make(map[[battle.TeamSize]battle.SpeciesID]bool)
	for _, combination := range roster.Combinations() {
		sorted := combination
		slices.Sort(sorted[:])
		if seen[sorted] {
			t.Errorf("組み合わせが重複している: %v", combination)
		}
		seen[sorted] = true
	}
	if len(seen) != 20 {
		t.Errorf("異なる組み合わせ = %d, want 20", len(seen))
	}
}
