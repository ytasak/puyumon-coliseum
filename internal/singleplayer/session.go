// Package singleplayer は1人用の1試合を進めるsession layerを提供する。
//
// 対戦のルールは持たない。行動順もダメージも状態異常もPPも internal/battle の
// resolverが決め、キャラクターと技の定義は internal/roster から取る。
// このpackageの責務は「6体を3体ずつ配る」「leadを決める」「外から来たcommandを
// resolverへ渡す」「戦闘不能のあとの交代を挟む」「決着まで進める」に限る。
//
// UIには依存しない。UIはcommandを送ってstateとEventを受け取るだけで、
// resolverを直接呼ばない。後のnetwork clientでも同じ境界を使えるように、
// player側とopponent側で同じcommand APIを使う。Botもこの境界の外側に置き、
// sessionの内部には持たない。
package singleplayer

import (
	"errors"
	"fmt"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
)

// 1人用でのsideの割り当て。
const (
	// Player は操作する側。
	Player = battle.Player1

	// Opponent は相手側。sessionはその中身が人かBotかを知らない。
	Opponent = battle.Player2
)

// sides は両sideを順に処理するための一覧。
var sides = [...]battle.Side{Player, Opponent}

// noLead はleadがまだ選ばれていないことを表す。
const noLead = -1

// ErrInvalidCommand はsessionがその時点で受け付けられないcommandを表す。
//
// phaseに合わないcommand、同じturnに2度目のAction、範囲外のlead indexなどが該当する。
// 「対戦のルール上そのActionを取れない」ことはこのerrorでは表さない。
// それはresolverが battle.ErrInvalidAction として返す。
var ErrInvalidCommand = errors.New("singleplayer: invalid command")

// Config は1試合の設定。
type Config struct {
	// Seed は試合全体のroot seed。
	//
	// ここから配布用・Battle Engine用・Bot判断用のseedを導出する。
	// 同じSeedと同じcommand列からは必ず同じ試合になる。
	Seed uint64
}

// SubmitResult はActionをsubmitした結果。
//
// 片側だけが出した時点ではturnを解決しないので、Eventsが空になる場合が2通りある。
// Resolvedを見れば「相手待ち」と「解決した」を取り違えずに済む。
type SubmitResult struct {
	// Resolved はこのsubmitでturnが解決したか。falseなら相手のAction待ち。
	Resolved bool

	// Events はこのsubmitで解決したturnのEvent。Resolvedがfalseなら常にnil。
	Events []battle.Event
}

// Session は1人用の1試合。
//
// 値として複製せず、pointerのまま使う。commandを受けて内部の状態を進める。
type Session struct {
	phase Phase

	// dealt は配布した3体。対戦が進んでも配布時点のまま変えない。
	// 対戦中の現在値はstateが持つ。
	dealt [2][battle.TeamSize]battle.Pokemon

	// leads は各sideが選んだleadのteam index。未選択はnoLead。
	leads [2]int

	// state は対戦の現在の状態。startedがfalseの間は未生成。
	state   battle.BattleState
	started bool

	resolver *battle.Resolver

	// botSeed はBotの判断に使うseed。sessionは値を渡すだけで自分では使わない。
	botSeed uint64

	// actions は両sideが揃うまでActionを溜めておく場所。
	actions   [2]battle.Action
	submitted [2]bool

	// history は起きたEventを起きた順に溜めたもの。
	history []battle.Event
}

// NewSession は3体ずつの配布まで済ませたsessionを返す。
//
// 返した時点のphaseはPhaseLeadSelectionで、BattleStateはまだ無い。
// 配布された3体はTeamで参照でき、両者がleadを選んだ時点で対戦が始まる。
func NewSession(cfg Config) (*Session, error) {
	derived := deriveSeeds(cfg.Seed)

	trios, err := deal(battle.NewRand(derived.setup))
	if err != nil {
		return nil, err
	}

	session := &Session{
		phase:    PhaseLeadSelection,
		leads:    [2]int{noLead, noLead},
		resolver: &battle.Resolver{Data: roster.Data(), RNG: battle.NewRand(derived.battle)},
		botSeed:  derived.bot,
	}
	for _, side := range sides {
		team, err := roster.NewTeam(trios[side])
		if err != nil {
			return nil, fmt.Errorf("%s team: %w", side, err)
		}
		session.dealt[side] = team
	}
	return session, nil
}

// Phase は現在の進行段階を返す。
func (s *Session) Phase() Phase {
	return s.phase
}

// BotSeed はBotの判断に使うseedを返す。
//
// Battle Engineの乱数列とは別の列なので、Botが何回引いてもこの試合の
// 命中・急所・ダメージは変わらない。
func (s *Session) BotSeed() uint64 {
	return s.botSeed
}

// Team は配布された3体を返す。sideが不正ならfalseを返す。
//
// 返すのは配布時点のsnapshotで、対戦中の現在値ではない。
// 現在のHPやPPはStateから読む。値として複製して返すので、
// 書き換えてもsessionの中の状態には波及しない。
func (s *Session) Team(side battle.Side) ([battle.TeamSize]battle.Pokemon, bool) {
	if !validSide(side) {
		return [battle.TeamSize]battle.Pokemon{}, false
	}
	return s.dealt[side], true
}

// State は対戦の現在の状態を返す。
//
// 両者がleadを選ぶまでBattleStateは存在しないので、そのあいだはfalseを返す。
// 空のBattleStateを有効な状態として返さない。
func (s *Session) State() (battle.BattleState, bool) {
	if !s.started {
		return battle.BattleState{}, false
	}
	return s.state, true
}

// History は試合開始から今までのEventを起きた順に返す。
//
// 直近のcommandで起きた分はcommandの戻り値が返す。こちらは累積で、
// 呼ぶたびに複製を返すため、書き換えてもsessionの中には波及しない。
func (s *Session) History() []battle.Event {
	events := make([]battle.Event, len(s.history))
	copy(events, s.history)
	return events
}

// Outcome は決着した試合の結果を返す。決着していなければfalseを返す。
func (s *Session) Outcome() (battle.Status, bool) {
	if s.phase != PhaseFinished {
		return battle.Ongoing, false
	}
	return s.state.Status, true
}

// SubmitLead は最初に場へ出す1体を決める。
//
// 両sideが選んだ時点でBattleStateを作り、PhaseBattleへ進む。
// opponent側も同じAPIで選ぶ。Botに選ばせる場合もsessionの外から呼ぶ。
func (s *Session) SubmitLead(side battle.Side, index int) error {
	if !validSide(side) {
		return fmt.Errorf("%w: unknown side %d", ErrInvalidCommand, int(side))
	}
	if s.phase != PhaseLeadSelection {
		return fmt.Errorf("%w: leads are chosen during %s, not %s", ErrInvalidCommand, PhaseLeadSelection, s.phase)
	}
	if s.leads[side] != noLead {
		return fmt.Errorf("%w: %s has already chosen a lead", ErrInvalidCommand, side)
	}
	if index < 0 || index >= battle.TeamSize {
		return fmt.Errorf("%w: lead index %d is out of range [0,%d)", ErrInvalidCommand, index, battle.TeamSize)
	}

	s.leads[side] = index
	if s.leads[Player] == noLead || s.leads[Opponent] == noLead {
		return nil
	}
	if err := s.start(); err != nil {
		// 対戦を始められなかったので、選んでいない状態へ戻す。
		s.leads[side] = noLead
		return err
	}
	return nil
}

// SubmitAction は通常のturnの行動を受け取る。
//
// 両sideが揃った時点で1 turnを解決する。片側だけならResolvedがfalseのまま返り、
// 相手のsubmitを待つ。同じsideが同じturnに2度submitすることはできない。
//
// 取れないActionは溜める前に弾くので、拒否されても相手の溜めたActionは残る。
// 出した側だけが出し直せばよい。合法性の判断はresolverが行い、sessionは複製しない。
func (s *Session) SubmitAction(side battle.Side, action battle.Action) (SubmitResult, error) {
	if !validSide(side) {
		return SubmitResult{}, fmt.Errorf("%w: unknown side %d", ErrInvalidCommand, int(side))
	}
	if s.phase != PhaseBattle {
		return SubmitResult{}, fmt.Errorf("%w: actions are submitted during %s, not %s", ErrInvalidCommand, PhaseBattle, s.phase)
	}
	if s.submitted[side] {
		return SubmitResult{}, fmt.Errorf("%w: %s has already submitted an action this turn", ErrInvalidCommand, side)
	}
	if action == nil {
		return SubmitResult{}, fmt.Errorf("%w: %s submitted no action", ErrInvalidCommand, side)
	}
	// 対戦のルール上取れないActionは、溜める前にここで落とす。溜めてから
	// 解決時に落とすと、どちらのActionが原因かを区別できないまま
	// 相手の分まで捨てることになる。
	if err := s.resolver.ValidateAction(s.state, side, action); err != nil {
		return SubmitResult{}, err
	}

	s.actions[side] = action
	s.submitted[side] = true
	if !s.submitted[Player] || !s.submitted[Opponent] {
		return SubmitResult{}, nil
	}

	next, events, err := s.resolver.ResolveTurn(s.state, s.actions[battle.Player1], s.actions[battle.Player2])
	// 溜めた時点で両方とも取れるActionであることを確かめてあり、そのあいだ
	// 状態は動かないのでここでは落ちない想定。落ちた場合は状態を進めず、
	// 両者が出し直すところからやり直せるようにする。
	s.clearActions()
	if err != nil {
		return SubmitResult{}, err
	}

	s.state = next
	s.history = append(s.history, events...)
	s.advance()
	return SubmitResult{Resolved: true, Events: events}, nil
}

// SubmitReplacement は戦闘不能になったあとに場へ出す控えを決める。
//
// 両sideが同時に戦闘不能になった場合は、両方がsubmitするまでPhaseReplacementのまま。
func (s *Session) SubmitReplacement(side battle.Side, action battle.SwitchAction) ([]battle.Event, error) {
	if !validSide(side) {
		return nil, fmt.Errorf("%w: unknown side %d", ErrInvalidCommand, int(side))
	}
	if s.phase != PhaseReplacement {
		return nil, fmt.Errorf("%w: replacements are submitted during %s, not %s", ErrInvalidCommand, PhaseReplacement, s.phase)
	}

	next, events, err := s.resolver.ResolveReplacement(s.state, side, action)
	if err != nil {
		return nil, err
	}

	s.state = next
	s.history = append(s.history, events...)
	s.advance()
	return events, nil
}

// start は両者のleadからBattleStateを作る。
//
// 配布順のindexは変えない。誰を先に出すかはActiveで表す。こうしておくと
// UIの並び、Eventが指すindex、交代先のindexが同じ空間のままになる。
func (s *Session) start() error {
	state, err := battle.NewBattleState(s.dealt[battle.Player1], s.dealt[battle.Player2])
	if err != nil {
		return err
	}
	for _, side := range sides {
		state.Players[side].Active = s.leads[side]
	}
	if err := state.Validate(); err != nil {
		return err
	}

	s.state = state
	s.started = true
	s.phase = PhaseBattle
	return nil
}

// advance は解決後の状態から次のphaseを決める。
func (s *Session) advance() {
	switch {
	case s.state.Status != battle.Ongoing:
		s.phase = PhaseFinished
	case s.state.NeedsReplacement(Player) || s.state.NeedsReplacement(Opponent):
		s.phase = PhaseReplacement
	default:
		s.phase = PhaseBattle
	}
}

// clearActions は溜めていたActionを捨てる。
func (s *Session) clearActions() {
	s.actions = [2]battle.Action{}
	s.submitted = [2]bool{}
}

// validSide は引数のsideが両者のどちらかを指しているかを返す。
func validSide(side battle.Side) bool {
	return side == Player || side == Opponent
}
