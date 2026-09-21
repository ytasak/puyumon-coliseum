package game

import (
	"fmt"

	"github.com/ytasak/puyumon-coliseum/internal/anim"
	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/battleui"
	"github.com/ytasak/puyumon-coliseum/internal/bot"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
	"github.com/ytasak/puyumon-coliseum/internal/singleplayer"
)

// 1人用でのsideの割り当て。画面を見ているのは常にplayer側。
const (
	viewer = singleplayer.Player
	foe    = singleplayer.Opponent
)

// battleSides は両sideを順に処理するための一覧。
var battleSides = [...]battle.Side{viewer, foe}

// cueTicks はcue 1つを見せているtick数。
//
// このあいだは入力を受け付けない。演出そのものの長さは描画を載せるときに
// 決めるので、ここでは「消化に時間がかかる」ことだけを表す。
const cueTicks = 20

// commandKind は画面が送れる操作の種類。
type commandKind int

const (
	// commandNone は何も選んでいないこと。
	commandNone commandKind = iota
	// commandLead は最初に出す1体を選ぶ。
	commandLead
	// commandMove は技を選ぶ。
	commandMove
	// commandStruggle はStruggleを選ぶ。
	commandStruggle
	// commandSwitch は控えへ交代する。voluntary switchとreplacementの両方で使う。
	commandSwitch
	// commandNewMatch は決着後に新しい対戦を始める。
	commandNewMatch
)

// command は画面が送る操作。
//
// **画面座標を持たない。** どの領域を押したらどのcommandになるかはlayoutの話で、
// この型は「何を選んだか」だけを表す。座標との対応はlayoutが決める。
type command struct {
	kind commandKind

	// index は対象。commandLeadとcommandSwitchではteam index、
	// commandMoveでは技のslot。それ以外では使わない。
	index int
}

// shownState はcueを消化しながら画面へ見せている値。
//
// Snapshotが持つのは落ち着いたあとの値なので、途中経過はこちらで持つ。
// cueを1つ消化するたびに更新し、queueが空になった時点でSnapshotと一致する。
// HPを最終値へ先に飛ばさないためにこの分離が要る。
type shownState struct {
	hp      [2][battle.TeamSize]int
	fainted [2][battle.TeamSize]bool

	// status は枠へ出す状態。cueで付いたり解けたりするたびに更新する。
	// Snapshotから読むと、cueを消化しきるまで古いままになる。
	status [2][battle.TeamSize]battleui.StatusView

	// active は枠へ出す1体。交代の途中でも、出るまでは下がる側を指したままにする。
	active [2]int

	// onField は場に誰か立っているか。交代の途中は誰も立っていない。
	// spriteを描くかどうかだけを決め、枠の表示には影響させない。
	onField [2]bool
}

// battleScene は1試合分の画面の状態。
//
// 対戦のルールは持たない。行動の合法性もturnの解決もsessionとBattle Engineが決め、
// この型は「今どの入力を受け付けるか」「どこまで見せたか」だけを持つ。
type battleScene struct {
	seed    uint64
	session *singleplayer.Session
	bot     *bot.Bot

	// view は直近のSnapshot。cueを消化しているあいだは更新しない。
	view battleui.View

	// pending は受理した自分の入力。相手待ちのあいだ二重入力を防ぐ。
	pending battleui.Pending

	// menu はコマンドのどの階層を開いているか。sessionの状態ではなく画面の都合。
	menu menuState

	// queue は再生待ちのcue。先頭から1つずつ消化する。
	queue   []battleui.Cue
	elapsed int

	shown shownState

	// message は画面に出す1行。cueを見せるたびに入れ替え、消化しきったら
	// 次に何をすればよいかへ戻す。
	message string

	// players は両activeのanimation状態。キャラクター定義とは分けて持つ。
	players [2]anim.Player

	// particles は表示中のEmoji particle。
	particles anim.Particles
}

// newBattleScene はseedから1試合分の状態を作る。
func newBattleScene(seed uint64) (*battleScene, error) {
	session, err := singleplayer.NewSession(singleplayer.Config{Seed: seed})
	if err != nil {
		return nil, fmt.Errorf("game: %w", err)
	}

	scene := &battleScene{
		seed:    seed,
		session: session,
		bot:     bot.New(roster.Data(), session.BotSeed()),
	}
	scene.refresh()
	return scene, nil
}

// busy はcueを消化中で入力を受け付けない状態かを返す。
//
// 消化中はviewが1つ前のまま止まるので、view.Commandsには直前の選択肢が
// 残っている。**描画側はここを見て、再生中は操作メニューを出さないか、
// 待機中と分かる表示にする。** view.Commandsを待機状態へ書き換えることはしない。
// 書き換えると、まだ見せていない結果を先出ししないための「viewを止める」
// 仕組みと食い違うため。
func (s *battleScene) busy() bool {
	return len(s.queue) > 0
}

// update は1 tick分進める。
//
// cueが残っていれば消化だけを行う。残っていなければ、相手の入力が要る状態かを見て
// Botへ委ねる。playerの入力はsubmitから入る。
func (s *battleScene) update() error {
	for _, side := range battleSides {
		s.players[side].Update()
	}
	s.particles.Update()

	if s.busy() {
		s.advanceCue()
		return nil
	}
	return s.driveOpponent()
}

// advanceCue は先頭のcueを見せ、時間が来たら次へ進む。
func (s *battleScene) advanceCue() {
	if s.elapsed == 0 {
		// 見せ始めた時点で反映する。HPはこのcueが持つ途中の値になる。
		s.apply(s.queue[0])
	}

	s.elapsed++
	if s.elapsed < cueTicks {
		return
	}

	s.queue = s.queue[1:]
	s.elapsed = 0
	if len(s.queue) == 0 {
		// 消化しきったところで、はじめて落ち着いたあとの値へ揃える。
		s.refresh()
	}
}

// apply はcue 1つ分を見せている値へ反映する。
func (s *battleScene) apply(cue battleui.Cue) {
	if message := cueMessage(cue, viewer); message != "" {
		s.message = message
	}

	switch c := cue.(type) {
	case battleui.MoveUsedCue:
		s.play(c.Actor.Side, anim.Attack, "")
	case battleui.DamageCue:
		s.setHP(c.Target, c.HP)
		s.play(c.Target.Side, anim.Hit, "")
	case battleui.HealCue:
		s.setHP(c.Target, c.HP)
		s.play(c.Target.Side, anim.Emphasis, "")
	case battleui.StatusCue:
		// 枠の表示もここで動かす。Snapshotは消化しきるまで更新しないので、
		// ここで反映しないと解けたあとも古い状態が残る。
		s.setStatus(c.Target, c.Status, c.Applied)
		s.play(c.Target.Side, anim.Emphasis, statusParticle(c.Status))
	case battleui.StatStageCue:
		s.play(c.Target.Side, anim.Emphasis, "")
	case battleui.SwitchOutCue:
		// 下がった時点でspriteを消す。出てくるまで場には誰も立っていない。
		if c.Target.Side == viewer || c.Target.Side == foe {
			s.shown.onField[c.Target.Side] = false
		}
		s.play(c.Target.Side, anim.Emphasis, "")
	case battleui.FaintCue:
		// ダメージを伴わずに倒れることがある（自爆や、自爆が外れた場合）。
		// このcueだけでHP 0・ひんしへ移してよい、というのがcueの契約。
		s.setHP(c.Target, 0)
		if inTeam(c.Target.Index) {
			s.shown.fainted[c.Target.Side][c.Target.Index] = true
		}
		s.play(c.Target.Side, anim.Hit, faintParticle)
	case battleui.SwitchInCue:
		if inTeam(c.Target.Index) {
			s.shown.active[c.Target.Side] = c.Target.Index
			s.shown.onField[c.Target.Side] = true
		}
		s.play(c.Target.Side, anim.Emphasis, "")
	}
}

// play はそのsideのanimationを再生し、必要ならparticleを出す。
//
// 使うのはYTA-9で作った3つのmotionとparticleだけ。交代とひんしにも
// 専用のmotionは足さず、表示の切り替えと既存motionで表す。
func (s *battleScene) play(side battle.Side, motion anim.Motion, particle string) {
	if side != viewer && side != foe {
		return
	}
	s.players[side].Play(motion)

	if particle == "" {
		return
	}
	anchor := spriteAnchors[side]
	s.particles.Spawn(particle, anchor.X, anchor.Y-anchor.Scale/2, particleSize)
}

// setStatus は見せている状態を更新する。
//
// 付いたか解けたかはcueが持っている。ここで判定し直さない。
func (s *battleScene) setStatus(ref battleui.Ref, status battle.MajorStatus, applied bool) {
	if !inTeam(ref.Index) {
		return
	}
	if !applied {
		s.shown.status[ref.Side][ref.Index] = battleui.StatusNone
		return
	}
	s.shown.status[ref.Side][ref.Index] = statusViewOf(status)
}

// setHP は見せているHPを更新する。
func (s *battleScene) setHP(ref battleui.Ref, hp int) {
	if !inTeam(ref.Index) {
		return
	}
	s.shown.hp[ref.Side][ref.Index] = hp
}

// submit はplayerの操作を受け取る。受理したらtrueを返す。
//
// 受け付けられるのは、いまの画面が出している選択肢だけ。cueの再生中や
// 相手待ちのあいだは何も受け付けないので、二重入力にならない。
// 合法性の最終判断はsessionが行い、ここでルールを複製しない。
func (s *battleScene) submit(c command) (accepted bool, err error) {
	if s.busy() {
		return false, nil
	}

	// 受理できたときだけ階層を根へ戻す。弾かれたときに閉じてしまうと、
	// 選び直すのにもう一度開く必要が出る。
	defer func() {
		if accepted {
			s.menu = menuRoot
		}
	}()

	switch s.view.Commands.Kind {
	case battleui.CommandChooseLead:
		return s.submitLead(c)
	case battleui.CommandChooseAction:
		return s.submitAction(c)
	case battleui.CommandChooseReplacement:
		return s.submitReplacement(c)
	case battleui.CommandFinished:
		return s.startNewMatch(c)
	default:
		// CommandWaiting。相手待ちなので何も受け付けない。
		return false, nil
	}
}

// submitLead は最初に出す1体を決める。
func (s *battleScene) submitLead(c command) (bool, error) {
	if c.kind != commandLead || !offered(s.view.Commands.Leads, c.index) {
		return false, nil
	}

	if err := s.session.SubmitLead(viewer, c.index); err != nil {
		// 受理されなかったので、溜めてある値は触らない。
		return false, nil
	}
	s.pending.Lead = true
	s.refresh()

	return true, s.driveOpponent()
}

// submitAction はそのturnの行動を決める。
func (s *battleScene) submitAction(c command) (bool, error) {
	var action battle.Action
	switch c.kind {
	case commandMove:
		// PPの尽きた技と空き枠は選べない。どれが選べるかはSnapshotが決めている。
		if !inMoveSlots(c.index) || s.view.Commands.Moves[c.index].Disabled {
			return false, nil
		}
		action = battle.MoveAction{Slot: c.index}
	case commandStruggle:
		if !s.view.Commands.Struggle {
			return false, nil
		}
		action = battle.StruggleAction{}
	case commandSwitch:
		if !offered(s.view.Commands.Switches, c.index) {
			return false, nil
		}
		action = battle.SwitchAction{Target: c.index}
	default:
		return false, nil
	}

	before, ok := s.session.State()
	if !ok {
		return false, nil
	}
	result, err := s.session.SubmitAction(viewer, action)
	if err != nil {
		return false, nil
	}
	s.pending.Action = true
	if result.Resolved {
		s.pending.Action = false
		s.enqueue(before, result.Events)
	}
	s.refresh()

	return true, s.driveOpponent()
}

// submitReplacement は戦闘不能のあとに出す控えを決める。
func (s *battleScene) submitReplacement(c command) (bool, error) {
	if c.kind != commandSwitch || !offered(s.view.Commands.Switches, c.index) {
		return false, nil
	}

	before, ok := s.session.State()
	if !ok {
		return false, nil
	}
	events, err := s.session.SubmitReplacement(viewer, battle.SwitchAction{Target: c.index})
	if err != nil {
		return false, nil
	}
	s.enqueue(before, events)
	s.refresh()

	return true, s.driveOpponent()
}

// startNewMatch は決着後に新しい対戦を始める。
//
// 対戦の再生ではなく、新しいsessionを作り直す。
func (s *battleScene) startNewMatch(c command) (bool, error) {
	if c.kind != commandNewMatch || !s.view.Commands.NewMatch {
		return false, nil
	}

	next, err := newBattleScene(nextSeed(s.seed))
	if err != nil {
		return false, err
	}
	// 古いpendingを持ち越さないよう、状態ごと入れ替える。
	*s = *next
	return true, nil
}

// driveOpponent は相手の入力が要る状態ならBotへ委ねる。
//
// Botはsessionの外側のactorなので、playerと同じcommand APIから入れる。
func (s *battleScene) driveOpponent() error {
	switch s.view.Phase {
	case singleplayer.PhaseLeadSelection:
		// playerが選ぶまでは動かさない。
		if !s.pending.Lead {
			return nil
		}
		own, ok := s.session.Team(foe)
		if !ok {
			return nil
		}
		against, ok := s.session.Team(viewer)
		if !ok {
			return nil
		}
		if err := s.session.SubmitLead(foe, s.bot.Lead(own, against)); err != nil {
			return fmt.Errorf("game: opponent lead: %w", err)
		}
		// 両者揃ってBattleへ移ったので、leadの待ちは終わり。
		s.pending.Lead = false
		s.refresh()

	case singleplayer.PhaseBattle:
		if !s.pending.Action {
			return nil
		}
		before, ok := s.session.State()
		if !ok {
			return nil
		}
		result, err := s.session.SubmitAction(foe, s.bot.Action(before, foe))
		if err != nil {
			return fmt.Errorf("game: opponent action: %w", err)
		}
		if result.Resolved {
			// どちら側のsubmitで解決してもturnの待ちは終わり。
			s.pending.Action = false
			s.enqueue(before, result.Events)
		}
		s.refresh()

	case singleplayer.PhaseReplacement:
		before, ok := s.session.State()
		if !ok || !before.NeedsReplacement(foe) {
			return nil
		}
		events, err := s.session.SubmitReplacement(foe, s.bot.Replacement(before, foe))
		if err != nil {
			return fmt.Errorf("game: opponent replacement: %w", err)
		}
		s.enqueue(before, events)
		s.refresh()
	}
	return nil
}

// enqueue はEvent列をcueへ変換して再生待ちへ積む。
func (s *battleScene) enqueue(before battle.BattleState, events []battle.Event) {
	if len(events) == 0 {
		return
	}
	s.queue = append(s.queue, battleui.Cues(before, events)...)
}

// refresh はSnapshotを取り直す。
//
// **cueを消化しているあいだは何もしない。** ここで取り直すと、まだ見せていない
// 途中経過を飛ばして、HPも結果も落ち着いたあとの値になってしまう。
// queueが空になってから揃える。
func (s *battleScene) refresh() {
	if s.busy() {
		return
	}

	in := battleui.Input{
		Phase:   s.session.Phase(),
		Pending: s.pending,
	}
	for _, side := range battleSides {
		team, ok := s.session.Team(side)
		if !ok {
			return
		}
		in.Teams[side] = team
	}
	in.State, in.Started = s.session.State()

	view, ok := battleui.Snapshot(in, viewer)
	if !ok {
		return
	}
	s.view = view
	s.shown = shownFrom(view)
	s.message = promptMessage(view)
}

// shownFrom は落ち着いたあとの値から、見せている値を作り直す。
func shownFrom(view battleui.View) shownState {
	var shown shownState
	shown.active[view.You.Side] = view.You.Active
	shown.active[view.Foe.Side] = view.Foe.Active
	shown.onField[view.You.Side] = inTeam(view.You.Active)
	shown.onField[view.Foe.Side] = inTeam(view.Foe.Active)

	for i, pokemon := range view.You.Team {
		shown.hp[view.You.Side][i] = pokemon.HP
		shown.fainted[view.You.Side][i] = pokemon.Fainted
		shown.status[view.You.Side][i] = pokemon.Status
	}
	for i, pokemon := range view.Foe.Team {
		shown.hp[view.Foe.Side][i] = pokemon.HP
		shown.fainted[view.Foe.Side][i] = pokemon.Fainted
		shown.status[view.Foe.Side][i] = pokemon.Status
	}
	return shown
}

// offered は選択肢に含まれるteam indexかを返す。
func offered(options []battleui.TeamOption, index int) bool {
	for _, option := range options {
		if option.Index == index {
			return true
		}
	}
	return false
}

// inTeam はteam indexとして取り得る値かを返す。
func inTeam(index int) bool {
	return index >= 0 && index < battle.TeamSize
}

// inMoveSlots は技のslotとして取り得る値かを返す。
func inMoveSlots(slot int) bool {
	return slot >= 0 && slot < battle.MoveSlots
}
