package game

import (
	"fmt"
	"strings"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/battleui"
	"github.com/ytasak/puyumon-coliseum/internal/singleplayer"
)

// 対戦の経過を出す文字列はASCIIだけで作る。
//
// 画面に描けるのは組み込みのASCIIフォントとEmojiだけで、日本語のフォントは
// 持っていない（README「技術的な判断」）。表示名も未確定なので、
// キャラクターはinternal IDを大文字にして指す。
//
// 文面はcueとviewから作る。**どの技が効いたか、なぜ効かなかったかを
// ここで判定し直さない。** それはBattle Engineが決めてcueに載せている。

// cueMessage はcue 1つ分の短い説明を返す。出すものが無ければ空を返す。
func cueMessage(cue battleui.Cue, viewer battle.Side) string {
	switch c := cue.(type) {
	case battleui.MoveUsedCue:
		return fmt.Sprintf("%s USED %s", refLabel(c.Actor, viewer), strings.ToUpper(string(c.Move)))

	case battleui.DamageCue:
		message := fmt.Sprintf("%s TOOK %d", refLabel(c.Target, viewer), c.Amount)
		if c.Critical {
			message += " CRITICAL HIT"
		}
		return message

	case battleui.HealCue:
		return fmt.Sprintf("%s RECOVERED %d", refLabel(c.Target, viewer), c.Amount)

	case battleui.StatusCue:
		if c.Applied {
			return fmt.Sprintf("%s IS %s", refLabel(c.Target, viewer), statusWord(c.Status))
		}
		return fmt.Sprintf("%s SHOOK OFF %s", refLabel(c.Target, viewer), statusWord(c.Status))

	case battleui.StatStageCue:
		direction := "FELL"
		if c.Delta > 0 {
			direction = "ROSE"
		}
		return fmt.Sprintf("%s %s %s", refLabel(c.Target, viewer), strings.ToUpper(c.Stat.String()), direction)

	case battleui.MultiHitCue:
		return fmt.Sprintf("HIT %d TIMES", c.Hits)

	case battleui.MissCue:
		return fmt.Sprintf("%s MISSED", refLabel(c.Actor, viewer))

	case battleui.UnaffectedCue:
		return fmt.Sprintf("NO EFFECT ON %s", refLabel(c.Target, viewer))

	case battleui.FailedCue:
		return "BUT IT FAILED"

	case battleui.BlockedCue:
		return fmt.Sprintf("%s %s", refLabel(c.Target, viewer), blockedWord(c.Reason))

	case battleui.SwitchOutCue:
		if c.Target.Side == viewer {
			return fmt.Sprintf("COME BACK %s", refLabel(c.Target, viewer))
		}
		return fmt.Sprintf("%s WITHDREW", refLabel(c.Target, viewer))

	case battleui.SwitchInCue:
		if c.Target.Side == viewer {
			return fmt.Sprintf("GO %s", refLabel(c.Target, viewer))
		}
		return fmt.Sprintf("%s CAME OUT", refLabel(c.Target, viewer))

	case battleui.FaintCue:
		return fmt.Sprintf("%s FAINTED", refLabel(c.Target, viewer))

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
		return "CHOOSE WHO GOES FIRST"
	case battleui.CommandChooseAction:
		return "WHAT WILL YOU DO"
	case battleui.CommandChooseReplacement:
		return "SEND OUT THE NEXT ONE"
	case battleui.CommandWaiting:
		return "WAITING FOR THE OPPONENT"
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
		return "DRAW"
	case result.Winner == viewer:
		return "YOU WIN"
	default:
		return "YOU LOSE"
	}
}

// refLabel は誰のことかを短く指す。相手側にはFOEを付ける。
func refLabel(ref battleui.Ref, viewer battle.Side) string {
	name := speciesLabel(ref.Species)
	if name == "" {
		name = "?"
	}
	if ref.Side == viewer {
		return name
	}
	return "FOE " + name
}

// statusWord は状態異常の言い方を返す。
func statusWord(status battle.MajorStatus) string {
	switch status {
	case battle.Burn:
		return "BURNED"
	case battle.Freeze:
		return "FROZEN"
	case battle.Paralysis:
		return "PARALYZED"
	case battle.Poison:
		return "POISONED"
	case battle.Sleep:
		return "ASLEEP"
	default:
		return strings.ToUpper(status.String())
	}
}

// blockedWord は動けなかった理由の言い方を返す。
func blockedWord(reason battleui.BlockReason) string {
	switch reason {
	case battleui.BlockSleep:
		return "IS FAST ASLEEP"
	case battleui.BlockWakeUp:
		return "WOKE UP"
	case battleui.BlockFreeze:
		return "IS FROZEN SOLID"
	case battleui.BlockParalysis:
		return "CANNOT MOVE"
	case battleui.BlockRecharge:
		return "MUST RECHARGE"
	default:
		return "CANNOT MOVE"
	}
}

// phaseLabel は画面の隅に出す進行段階。
func phaseLabel(phase singleplayer.Phase) string {
	return strings.ToUpper(strings.ReplaceAll(phase.String(), " ", "-"))
}
