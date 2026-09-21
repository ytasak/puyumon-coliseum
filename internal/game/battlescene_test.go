package game

import (
	"reflect"
	"testing"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/battleui"
	"github.com/ytasak/puyumon-coliseum/internal/singleplayer"
)

// maxSceneSteps は1試合を打ち切る操作回数。決着しないことを検出するための安全装置。
const maxSceneSteps = 2000

// 作った直後はlead選択から始まり、3体すべてが候補になる。
func TestBattleSceneStartsAtLeadSelection(t *testing.T) {
	t.Parallel()

	scene := newSceneOrFatal(t, 1)

	if got := scene.view.Phase; got != singleplayer.PhaseLeadSelection {
		t.Errorf("phase = %s, want %s", got, singleplayer.PhaseLeadSelection)
	}
	if got := scene.view.Commands.Kind; got != battleui.CommandChooseLead {
		t.Errorf("command = %s, want %s", got, battleui.CommandChooseLead)
	}
	if got := len(scene.view.Commands.Leads); got != battle.TeamSize {
		t.Errorf("lead候補が %d 件（%d 件のはず）", got, battle.TeamSize)
	}
	if scene.busy() {
		t.Error("開始直後からcueを消化している")
	}
}

// leadを選ぶとBotも選び、対戦が始まる。
func TestSubmittingALeadStartsTheBattle(t *testing.T) {
	t.Parallel()

	scene := newSceneOrFatal(t, 1)
	submitOrFatal(t, scene, command{kind: commandLead, index: 2})

	if got := scene.view.Phase; got != singleplayer.PhaseBattle {
		t.Fatalf("phase = %s, want %s", got, singleplayer.PhaseBattle)
	}
	if got := scene.view.Commands.Kind; got != battleui.CommandChooseAction {
		t.Errorf("command = %s, want %s", got, battleui.CommandChooseAction)
	}
	if got := scene.view.You.Active; got != 2 {
		t.Errorf("自分のactive = %d, want 2", got)
	}
	if got := scene.view.Foe.Active; got == battleui.NoActive {
		t.Error("Botのleadが決まっていない")
	}
	// lead待ちは終わっている。
	if scene.pending.Lead {
		t.Error("Battleへ移ってもlead pendingが残っている")
	}
}

// 選択肢に無いleadは受け付けない。
func TestSceneRejectsAnUnknownLead(t *testing.T) {
	t.Parallel()

	scene := newSceneOrFatal(t, 1)
	for _, c := range []command{
		{kind: commandLead, index: -1},
		{kind: commandLead, index: battle.TeamSize},
		{kind: commandMove, index: 0},
		{kind: commandNewMatch},
	} {
		if accepted, err := scene.submit(c); accepted || err != nil {
			t.Errorf("%+v が受理された（accepted=%v err=%v）", c, accepted, err)
		}
	}
	if scene.view.Phase != singleplayer.PhaseLeadSelection {
		t.Error("拒否したのにphaseが進んだ")
	}
}

// PPの尽きた技は送信できない。
func TestSceneRejectsAMoveWithoutPP(t *testing.T) {
	t.Parallel()

	scene, empty, ok := sceneWithDisabledMove(t)
	if !ok {
		t.Fatal("PPを使い切る局面を作れなかった")
	}

	if accepted, err := scene.submit(command{kind: commandMove, index: empty}); accepted || err != nil {
		t.Errorf("PP 0 の技が受理された（accepted=%v err=%v）", accepted, err)
	}
	if scene.busy() {
		t.Error("拒否したのにturnが進んだ")
	}

	// 残っている技は通る。
	usable, ok := enabledSlot(scene)
	if !ok {
		t.Fatal("technical: 使える技が1つも無い")
	}
	submitOrFatal(t, scene, command{kind: commandMove, index: usable})
}

// 使える技が残っているうちはStruggleを送信できない。
func TestSceneRejectsStruggleWhileMovesRemain(t *testing.T) {
	t.Parallel()

	scene := startedScene(t, 1)

	if scene.view.Commands.Struggle {
		t.Skip("開始直後からStruggleしか無い局面ではない")
	}
	if accepted, _ := scene.submit(command{kind: commandStruggle}); accepted {
		t.Error("使える技が残っているのにStruggleが受理された")
	}
}

// 技を選ぶとturnが解決し、cueが積まれる。
func TestSubmittingAMoveResolvesTheTurn(t *testing.T) {
	t.Parallel()

	scene := startedScene(t, 1)
	before, _ := scene.session.State()

	submitOrFatal(t, scene, command{kind: commandMove, index: 0})

	if !scene.busy() {
		t.Fatal("turnが解決したのにcueが積まれていない")
	}
	// 相手の分もsubmit済みなので、行動待ちは残っていない。
	if scene.pending.Action {
		t.Error("turn解決後もaction pendingが残っている")
	}

	drainCues(t, scene)

	after, _ := scene.session.State()
	if after.Turn <= before.Turn && after.Status == battle.Ongoing {
		t.Errorf("turnが進んでいない: %d -> %d", before.Turn, after.Turn)
	}
}

// cueを消化しているあいだは入力を受け付けない。
func TestSceneIgnoresInputWhileCuesPlay(t *testing.T) {
	t.Parallel()

	scene := startedScene(t, 1)
	submitOrFatal(t, scene, command{kind: commandMove, index: 0})
	if !scene.busy() {
		t.Fatal("cueが積まれていない")
	}

	turnBefore := scene.view.You.Team[scene.view.You.Active].HP
	for _, c := range []command{
		{kind: commandMove, index: 1},
		{kind: commandSwitch, index: 1},
		{kind: commandStruggle},
	} {
		if accepted, err := scene.submit(c); accepted || err != nil {
			t.Errorf("再生中に %+v が受理された（accepted=%v err=%v）", c, accepted, err)
		}
	}
	if got := scene.view.You.Team[scene.view.You.Active].HP; got != turnBefore {
		t.Error("拒否したのに状態が動いた")
	}
}

// HPは各cueの途中の値をたどり、最終値へ先に飛ばない。
func TestShownHPFollowsEachCue(t *testing.T) {
	t.Parallel()

	scene, steps, ok := sceneWithDamageCues(t)
	if !ok {
		t.Fatal("HPが動くturnを作れなかった")
	}

	seen := map[int]bool{}
	for scene.busy() {
		if err := scene.update(); err != nil {
			t.Fatalf("update()に失敗: %v", err)
		}
		for _, step := range steps {
			if scene.shown.hp[step.Target.Side][step.Target.Index] == step.HP {
				seen[step.HP] = true
			}
		}
	}

	for _, step := range steps {
		if !seen[step.HP] {
			t.Errorf("HP %d を見せずに飛ばした", step.HP)
		}
	}
}

// cueを消化しきるまでSnapshotの最終値を先出ししない。
func TestViewDoesNotJumpAheadOfTheCues(t *testing.T) {
	t.Parallel()

	scene := startedScene(t, 1)
	turnBefore := scene.view.You.Team[0].HP
	phaseBefore := scene.view.Phase

	submitOrFatal(t, scene, command{kind: commandMove, index: 0})
	if !scene.busy() {
		t.Skip("cueが積まれない局面だった")
	}

	// 1 tickだけ進めても、viewはturn前のまま。
	if err := scene.update(); err != nil {
		t.Fatalf("update()に失敗: %v", err)
	}
	if scene.view.Phase != phaseBefore {
		t.Errorf("再生中にphaseが進んだ: %s -> %s", phaseBefore, scene.view.Phase)
	}
	if scene.view.You.Team[0].HP != turnBefore {
		t.Error("再生中にviewのHPが動いた")
	}

	drainCues(t, scene)
	if scene.shown.hp[scene.view.You.Side][0] != scene.view.You.Team[0].HP {
		t.Error("消化しきってもviewと見せている値が一致しない")
	}
}

// 戦闘不能のあとは控えだけを受け付ける。
func TestReplacementRejectsFightCommands(t *testing.T) {
	t.Parallel()

	scene, ok := sceneAtReplacement(t)
	if !ok {
		t.Skip("replacementへ到達する局面を作れなかった")
	}

	for _, c := range []command{
		{kind: commandMove, index: 0},
		{kind: commandStruggle},
		{kind: commandLead, index: 0},
	} {
		if accepted, err := scene.submit(c); accepted || err != nil {
			t.Errorf("replacement中に %+v が受理された（accepted=%v err=%v）", c, accepted, err)
		}
	}

	target := scene.view.Commands.Switches[0].Index
	submitOrFatal(t, scene, command{kind: commandSwitch, index: target})
	drainCues(t, scene)

	if scene.view.Phase == singleplayer.PhaseReplacement && scene.view.Commands.Kind == battleui.CommandChooseReplacement {
		t.Error("控えを出してもreplacementのままになっている")
	}
}

// 決着したら結果が出て、新しい対戦を始められる。
func TestNewMatchStartsAFreshSession(t *testing.T) {
	t.Parallel()

	scene := startedScene(t, 1)
	playToFinish(t, scene)

	if got := scene.view.Commands.Kind; got != battleui.CommandFinished {
		t.Fatalf("決着後のcommand = %s, want %s", got, battleui.CommandFinished)
	}
	if !scene.view.Result.Decided {
		t.Fatal("決着したのにResultがDecidedでない")
	}
	outcome, ok := scene.session.Outcome()
	if !ok {
		t.Fatal("sessionのOutcomeが取れない")
	}
	assertResultMatches(t, scene.view.Result, outcome)

	oldSeed := scene.seed
	oldTeam := scene.view.You.Team
	submitOrFatal(t, scene, command{kind: commandNewMatch})

	if scene.seed == oldSeed {
		t.Error("新しい対戦でseedが変わっていない")
	}
	if scene.view.Phase != singleplayer.PhaseLeadSelection {
		t.Errorf("新しい対戦のphase = %s, want %s", scene.view.Phase, singleplayer.PhaseLeadSelection)
	}
	if scene.pending != (battleui.Pending{}) {
		t.Errorf("古いpendingを持ち越した: %+v", scene.pending)
	}
	if scene.busy() {
		t.Error("新しい対戦がcue再生中から始まった")
	}
	if reflect.DeepEqual(scene.view.You.Team, oldTeam) {
		t.Error("新しい対戦なのに配布が同じ")
	}
}

// 同じseedと同じ操作からは同じ試合になる。
func TestSceneIsDeterministic(t *testing.T) {
	t.Parallel()

	first := startedScene(t, 7)
	playToFinish(t, first)

	second := startedScene(t, 7)
	playToFinish(t, second)

	if !reflect.DeepEqual(first.session.History(), second.session.History()) {
		t.Error("同じseedと同じ操作で違うEvent列になった")
	}
	if !reflect.DeepEqual(first.view, second.view) {
		t.Error("同じseedと同じ操作で違うviewになった")
	}
}

// newSceneOrFatal はsceneを作る。
func newSceneOrFatal(t *testing.T, seed uint64) *battleScene {
	t.Helper()

	scene, err := newBattleScene(seed)
	if err != nil {
		t.Fatalf("newBattleScene(%d)に失敗: %v", seed, err)
	}
	return scene
}

// startedScene はleadを決めてBattleまで進めたsceneを返す。
func startedScene(t *testing.T, seed uint64) *battleScene {
	t.Helper()

	scene := newSceneOrFatal(t, seed)
	submitOrFatal(t, scene, command{kind: commandLead, index: 0})
	drainCues(t, scene)
	if scene.view.Phase != singleplayer.PhaseBattle {
		t.Fatalf("phase = %s, want %s", scene.view.Phase, singleplayer.PhaseBattle)
	}
	return scene
}

// submitOrFatal は操作を送る。受理されなければ止める。
func submitOrFatal(t *testing.T, s *battleScene, c command) {
	t.Helper()

	accepted, err := s.submit(c)
	if err != nil {
		t.Fatalf("submit(%+v)に失敗: %v", c, err)
	}
	if !accepted {
		t.Fatalf("submit(%+v)が受理されなかった（command=%s）", c, s.view.Commands.Kind)
	}
}

// drainCues は再生待ちのcueを消化しきる。
func drainCues(t *testing.T, s *battleScene) {
	t.Helper()

	for ticks := 0; s.busy(); ticks++ {
		if ticks > cueTicks*10000 {
			t.Fatal("cueが尽きない")
		}
		if err := s.update(); err != nil {
			t.Fatalf("update()に失敗: %v", err)
		}
	}
}

// playToFinish は決着まで進める。選ぶのは常に先頭の選択肢。
func playToFinish(t *testing.T, s *battleScene) {
	t.Helper()

	for step := 0; s.view.Phase != singleplayer.PhaseFinished; step++ {
		if step >= maxSceneSteps {
			t.Fatalf("%d 手で決着しなかった", maxSceneSteps)
		}
		drainCues(t, s)
		if s.view.Phase == singleplayer.PhaseFinished {
			break
		}

		c, ok := firstChoice(s.view.Commands)
		if !ok {
			// 相手待ち。updateで進める。
			if err := s.update(); err != nil {
				t.Fatalf("update()に失敗: %v", err)
			}
			continue
		}
		submitOrFatal(t, s, c)
	}
	drainCues(t, s)
}

// firstChoice はいまの選択肢の先頭を返す。
func firstChoice(commands battleui.Commands) (command, bool) {
	switch commands.Kind {
	case battleui.CommandChooseLead:
		return command{kind: commandLead, index: commands.Leads[0].Index}, true
	case battleui.CommandChooseAction:
		for slot, move := range commands.Moves {
			if !move.Disabled {
				return command{kind: commandMove, index: slot}, true
			}
		}
		if commands.Struggle {
			return command{kind: commandStruggle}, true
		}
		if len(commands.Switches) > 0 {
			return command{kind: commandSwitch, index: commands.Switches[0].Index}, true
		}
		return command{}, false
	case battleui.CommandChooseReplacement:
		return command{kind: commandSwitch, index: commands.Switches[0].Index}, true
	default:
		return command{}, false
	}
}

// enabledSlot は選べる技のslotを返す。
func enabledSlot(s *battleScene) (int, bool) {
	for slot, move := range s.view.Commands.Moves {
		if !move.Disabled {
			return slot, true
		}
	}
	return 0, false
}

// sceneWithDisabledMove は、同じ技を撃ち続けてPPを使い切らせた局面を返す。
//
// 状態を差し込めないので、実際に使い切るまで進める。
func sceneWithDisabledMove(t *testing.T) (*battleScene, int, bool) {
	t.Helper()

	const slot = 0
	for seed := uint64(1); seed < 30; seed++ {
		scene := startedScene(t, seed)
		for step := 0; step < maxSceneSteps; step++ {
			drainCues(t, scene)
			if scene.view.Phase == singleplayer.PhaseFinished {
				break
			}

			if scene.view.Commands.Kind == battleui.CommandChooseAction {
				if scene.view.Commands.Moves[slot].Disabled {
					return scene, slot, true
				}
				submitOrFatal(t, scene, command{kind: commandMove, index: slot})
				continue
			}

			c, ok := firstChoice(scene.view.Commands)
			if !ok {
				if err := scene.update(); err != nil {
					t.Fatalf("update()に失敗: %v", err)
				}
				continue
			}
			submitOrFatal(t, scene, c)
		}
	}
	return nil, 0, false
}

// sceneWithDamageCues は、HPが動くturnを解決した直後のsceneを返す。
//
// 状態異常だけのturnもあるので、ダメージが出る組み合わせを探す。
func sceneWithDamageCues(t *testing.T) (*battleScene, []battleui.DamageCue, bool) {
	t.Helper()

	for seed := uint64(1); seed < 30; seed++ {
		scene := startedScene(t, seed)
		c, ok := firstChoice(scene.view.Commands)
		if !ok {
			continue
		}
		submitOrFatal(t, scene, c)

		var steps []battleui.DamageCue
		for _, cue := range scene.queue {
			if damage, ok := cue.(battleui.DamageCue); ok {
				steps = append(steps, damage)
			}
		}
		if len(steps) > 0 {
			return scene, steps, true
		}
		drainCues(t, scene)
	}
	return nil, nil, false
}

// sceneAtReplacement は自分が交代を迫られている局面まで進めたsceneを返す。
func sceneAtReplacement(t *testing.T) (*battleScene, bool) {
	t.Helper()

	for seed := uint64(1); seed < 30; seed++ {
		scene := startedScene(t, seed)
		for step := 0; step < maxSceneSteps; step++ {
			drainCues(t, scene)
			if scene.view.Commands.Kind == battleui.CommandChooseReplacement {
				return scene, true
			}
			if scene.view.Phase == singleplayer.PhaseFinished {
				break
			}
			c, ok := firstChoice(scene.view.Commands)
			if !ok {
				if err := scene.update(); err != nil {
					t.Fatalf("update()に失敗: %v", err)
				}
				continue
			}
			submitOrFatal(t, scene, c)
		}
	}
	return nil, false
}

// assertResultMatches は表示用の結果がsessionの決着と一致することを確かめる。
func assertResultMatches(t *testing.T, result battleui.Result, outcome battle.Status) {
	t.Helper()

	switch outcome {
	case battle.Player1Won:
		if result.Draw || result.Winner != battle.Player1 {
			t.Errorf("Result = %+v, want player1の勝ち", result)
		}
	case battle.Player2Won:
		if result.Draw || result.Winner != battle.Player2 {
			t.Errorf("Result = %+v, want player2の勝ち", result)
		}
	case battle.Draw:
		if !result.Draw {
			t.Errorf("Result = %+v, want 引き分け", result)
		}
	default:
		t.Errorf("決着していないOutcome: %s", outcome)
	}
}
