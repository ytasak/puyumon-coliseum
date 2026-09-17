// Package battle はGeneration I準拠のBattle Engineのdomain modelを提供する。
//
// このpackageはUIに依存しない。Ebitengineをimportせず、描画・入力・実時間の概念を持たない。
// 扱うのは対戦の状態（BattleState）、プレイヤーの行動（Action）、
// 解決結果を表すEventだけで、turnの解決そのものは後続のresolverが担当する。
//
// 乱数はRNGとして外から注入する。global stateへ依存しないため、同じ初期状態・
// 同じAction列・同じseedからいつでも同じ結果を再現できる。
//
// 仕様の正はLinearのProject Document「Battle Rules Specification v0.1」。
// このpackageはそこで定義されたmodelの範囲だけを持ち、ダメージ計算・タイプ相性・
// 命中判定・状態異常の挙動は別のIssueで追加する。
package battle

import (
	"errors"
	"fmt"
)

// TeamSize は1プレイヤーが戦わせるPokemonの数。
//
// 本作は6体の候補から3体を配って戦うため、対戦中のチームは常にこの数になる。
const TeamSize = 3

// ErrInvalidState は状態やActionの内容が取り得ない値であることを表す。
//
// 検証で見つかる不整合はいずれもこのerrorをwrapして返すので、
// 呼び出し側は errors.Is で判別できる。
var ErrInvalidState = errors.New("battle: invalid state")

// Side はどちらのプレイヤーかを表す。BattleState.Playersの添字にそのまま使える。
type Side int

const (
	// Player1 は先手側のプレイヤー。
	Player1 Side = iota
	// Player2 は後手側のプレイヤー。
	Player2
)

// sides は両プレイヤーを順に処理するためのSide一覧。
var sides = [...]Side{Player1, Player2}

// Opponent は相手側を返す。
func (s Side) Opponent() Side {
	if s == Player1 {
		return Player2
	}
	return Player1
}

// String はSideの名前を返す。testとlogの出力に使う。
func (s Side) String() string {
	switch s {
	case Player1:
		return "player1"
	case Player2:
		return "player2"
	default:
		return "unknown"
	}
}

// valid はSideが定義済みの値かを返す。
func (s Side) valid() bool {
	return s == Player1 || s == Player2
}

// Status は対戦全体の進行状況。
type Status int

const (
	// Ongoing は対戦が継続中。
	Ongoing Status = iota
	// Player1Won はPlayer1の勝ち。
	Player1Won
	// Player2Won はPlayer2の勝ち。
	Player2Won
	// Draw は両者に戦えるPokemonが残っていない。相打ちで決着しない場合に使う。
	Draw
)

// String はStatusの名前を返す。
func (s Status) String() string {
	switch s {
	case Ongoing:
		return "ongoing"
	case Player1Won:
		return "player1_won"
	case Player2Won:
		return "player2_won"
	case Draw:
		return "draw"
	default:
		return "unknown"
	}
}

// Player は1プレイヤー分の状態。
type Player struct {
	// Team は戦わせる3体。交代してもindexは変わらないため、
	// Eventやactionからは常にこのindexでPokemonを指せる。
	Team [TeamSize]Pokemon

	// Active は場に出ているPokemonのTeam内index。残りがreserveにあたる。
	Active int
}

// ActivePokemon は場に出ているPokemonへのpointerを返す。
//
// BattleStateは値として複製できる前提なので、返ったpointerは
// 複製元ではなく手元のPlayerを指す。複製後に書き換えても元へは波及しない。
func (p *Player) ActivePokemon() *Pokemon {
	return &p.Team[p.Active]
}

// Reserve は場に出ていないPokemonのTeam内indexを昇順で返す。
func (p *Player) Reserve() []int {
	reserve := make([]int, 0, TeamSize-1)
	for i := range p.Team {
		if i != p.Active {
			reserve = append(reserve, i)
		}
	}
	return reserve
}

// BattleState は対戦の全状態。
//
// pointerを持たないため値として複製でき、解決前後の状態を別々に保持できる。
// 乱数はここに含めない。RNGはresolverへ注入するもので、状態の一部ではない。
type BattleState struct {
	// Turn は1から始まるturn数。
	Turn int

	// Players は両プレイヤーの状態。Sideをそのまま添字に使える。
	Players [2]Player

	// Status は対戦の進行状況。
	Status Status
}

// NewBattleState は1 turn目の開始状態を作る。
//
// 各プレイヤーはTeamの先頭（index 0）を場に出した状態から始まる。
// 内容が取り得ない値であればerrorを返し、状態は作らない。
func NewBattleState(team1, team2 [TeamSize]Pokemon) (BattleState, error) {
	state := BattleState{
		Turn: 1,
		Players: [2]Player{
			Player1: {Team: team1},
			Player2: {Team: team2},
		},
		Status: Ongoing,
	}
	if err := state.Validate(); err != nil {
		return BattleState{}, err
	}
	return state, nil
}

// Validate は状態が取り得ない値を含んでいないかを確かめる。
//
// ここで見るのは「その値が状態として成立するか」だけで、
// 勝敗やreplacementの要否といった進行の判断は行わない。
func (b *BattleState) Validate() error {
	if b.Turn < 1 {
		return fmt.Errorf("%w: turn %d is less than 1", ErrInvalidState, b.Turn)
	}
	switch b.Status {
	case Ongoing, Player1Won, Player2Won, Draw:
	default:
		return fmt.Errorf("%w: unknown status %d", ErrInvalidState, int(b.Status))
	}

	for _, side := range sides {
		player := b.Players[side]
		if player.Active < 0 || player.Active >= TeamSize {
			return fmt.Errorf("%w: %s active slot %d is out of range [0,%d)",
				ErrInvalidState, side, player.Active, TeamSize)
		}
		for i := range player.Team {
			if err := player.Team[i].validate(); err != nil {
				return fmt.Errorf("%s team[%d]: %w", side, i, err)
			}
		}
	}
	return nil
}
