package bot

import "github.com/ytasak/puyumon-coliseum/internal/battle"

// opponentView は相手について、対戦画面から読み取れる情報だけを取り出したもの。
//
// 相手の技構成と残りPPは画面に出ないので、この型には入れない。評価はすべて
// この型を経由するため、相手のmove slotを読むコードを書けない形にしてある
// （YTA-29 Design decision 4）。
//
// 今のv1が実際に見ているのはタイプと状態異常だけ。HPやひんしも画面には
// 出ているので、判断に要るようになった時点で足せばよい。
type opponentView struct {
	// typing は相手のタイプ構成。SpeciesIDから引ける。
	typing battle.Typing

	// status は状態異常。重ねがけを避けるために見る。
	status battle.MajorStatus
}

// viewOf は相手の個体から、見てよい情報だけを取り出す。
func (b *Bot) viewOf(p battle.Pokemon) opponentView {
	return opponentView{
		typing: b.typingOf(p.Species),
		status: p.Status,
	}
}

// typingOf は定義からタイプ構成を引く。
//
// 定義が無ければ空のTypingを返す。相性はすべて等倍として扱われるので、
// 判断は鈍るが落ちはしない。
func (b *Bot) typingOf(id battle.SpeciesID) battle.Typing {
	species, err := b.data.LookupSpecies(id)
	if err != nil {
		return battle.Typing{}
	}
	return species.Typing
}
