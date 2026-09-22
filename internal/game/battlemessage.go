package game

import (
	"fmt"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/battleui"
)

// 対戦の経過を出す文字列を日本語で組む。
//
// 説明されなくても分かる短い文にする。1行に収まる長さで、専門用語や
// 内部IDをそのまま出さない。表示名は jptext.go の対応表から引く。
//
// 文面はcueとviewから作る。**どの技が効いたか、なぜ効かなかったかを
// ここで判定し直さない。** それはBattle Engineが決めてcueに載せている。

// cueMessage はcue 1つ分の短い説明を返す。出すものが無ければ空を返す。
func cueMessage(cue battleui.Cue, viewer battle.Side) string {
	switch c := cue.(type) {
	case battleui.MoveUsedCue:
		return fmt.Sprintf("%sの%s！", refLabel(c.Actor, viewer), moveName(c.Move))

	case battleui.DamageCue:
		message := fmt.Sprintf("%sに%dのダメージ", refLabel(c.Target, viewer), c.Amount)
		if c.Critical {
			message += " 急所に当たった"
		}
		return message

	case battleui.HealCue:
		return fmt.Sprintf("%sは%d回復した", refLabel(c.Target, viewer), c.Amount)

	case battleui.StatusCue:
		if c.Applied {
			return fmt.Sprintf("%sは%sになった", refLabel(c.Target, viewer), majorStatusName(c.Status))
		}
		return fmt.Sprintf("%sの%sが治った", refLabel(c.Target, viewer), majorStatusName(c.Status))

	case battleui.StatStageCue:
		direction := "下がった"
		if c.Delta > 0 {
			direction = "上がった"
		}
		return fmt.Sprintf("%sの%sが%s", refLabel(c.Target, viewer), statName(c.Stat), direction)

	case battleui.MultiHitCue:
		return fmt.Sprintf("%d回当たった", c.Hits)

	case battleui.MissCue:
		return fmt.Sprintf("%sの攻撃は外れた", refLabel(c.Actor, viewer))

	case battleui.UnaffectedCue:
		return fmt.Sprintf("%sには効果がないようだ", refLabel(c.Target, viewer))

	case battleui.FailedCue:
		return "うまく決まらなかった"

	case battleui.BlockedCue:
		return fmt.Sprintf("%sは%s", refLabel(c.Target, viewer), blockedWord(c.Reason))

	case battleui.SwitchOutCue:
		if c.Target.Side == viewer {
			return fmt.Sprintf("戻れ！ %s", speciesName(c.Target.Species))
		}
		return fmt.Sprintf("%sは引っ込んだ", refLabel(c.Target, viewer))

	case battleui.SwitchInCue:
		if c.Target.Side == viewer {
			return fmt.Sprintf("行け！ %s", speciesName(c.Target.Species))
		}
		return fmt.Sprintf("%sが出てきた", refLabel(c.Target, viewer))

	case battleui.FaintCue:
		return fmt.Sprintf("%sは倒れた", refLabel(c.Target, viewer))

	default:
		return ""
	}
}

// promptMessage はcueを消化しきったあとに出す1行を返す。
func promptMessage(view battleui.View) string {
	if view.Result.Decided {
		return resultMessage(view.Result, view.You.Side)
	}

	switch view.Commands.Kind {
	case battleui.CommandChooseLead:
		return "最初に出す1体を選んでください"
	case battleui.CommandChooseAction:
		return "どうしますか？"
	case battleui.CommandChooseReplacement:
		return "次に出す1体を選んでください"
	case battleui.CommandWaiting:
		return "相手の行動を待っています"
	default:
		return ""
	}
}

// resultMessage は決着の1行を返す。
func resultMessage(result battleui.Result, viewer battle.Side) string {
	switch {
	case !result.Decided:
		return ""
	case result.Draw:
		return "引き分け"
	case result.Winner == viewer:
		return "あなたの勝ち！"
	default:
		return "あなたの負け"
	}
}

// refLabel は誰のことかを短く指す。相手側には「相手の」を付ける。
func refLabel(ref battleui.Ref, viewer battle.Side) string {
	name := speciesName(ref.Species)
	if ref.Side == viewer {
		return name
	}
	return "相手の" + name
}

// majorStatusName は状態異常の言い方を返す。
//
// 情報枠の表記と同じ語を使う。同じ状態を枠と文章で別の言葉にすると、
// どちらを見ているかで呼び名が変わってしまう。
func majorStatusName(status battle.MajorStatus) string {
	if name := statusName(statusViewOf(status)); name != "" {
		return name
	}
	return unknownName
}

// blockedWord は動けなかった理由の言い方を返す。
func blockedWord(reason battleui.BlockReason) string {
	switch reason {
	case battleui.BlockSleep:
		return "ぐっすり眠っている"
	case battleui.BlockWakeUp:
		return "目を覚ました"
	case battleui.BlockFreeze:
		return "凍って動けない"
	case battleui.BlockParalysis:
		return "しびれて動けない"
	case battleui.BlockRecharge:
		return "反動で動けない"
	default:
		return "動けない"
	}
}
