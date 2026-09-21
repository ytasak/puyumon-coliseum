package bot

import "github.com/ytasak/puyumon-coliseum/internal/battle"

// heuristicの調整値。どれも実装の内側の話で、対戦のルールではない。
//
// 単位は damageScore に合わせてある。等倍・STABなし・必中の技なら
// おおよそ威力がそのままscoreになるので、「威力いくつ相当に見るか」で読める。
const (
	// stabBonus はSTABが乗る技の割り増し。百分率。
	stabBonus = 150
	// noStabBonus はSTABが乗らない技の倍率。百分率。
	noStabBonus = 100

	// statusScore は状態異常を与える技の基礎点。命中を掛けてから使う。
	statusScore = 110
	// healScore はHPが減っているときの回復技の点。
	healScore = 120
	// boostScore は能力を上げる技の点。
	boostScore = 60
	// explodeBonus は追い詰められたときの自爆の上乗せ。
	explodeBonus = 200
	// fallbackScore は個別に扱っていない威力0の技の点。
	fallbackScore = 50

	// healHPPercent は回復技を候補にする残りHPの割合。
	healHPPercent = 50
	// explodeHPPercent は自爆を候補にする残りHPの割合。
	explodeHPPercent = 25
)

// damageScore は攻撃技の粗い評価。
//
// 威力・STAB・タイプ相性・命中を掛けただけの**heuristicであって、
// Generation Iのdamage formulaではない。** 実際のダメージ量とは一致しないし、
// 能力値も乱数も急所も見ていない。技を並べて比べるためだけの数字。
func damageScore(move battle.Move, user battle.Typing, target opponentView) int {
	effect := battle.AgainstTyping(move.Type, target.typing)
	if effect == battle.NoEffect {
		return 0
	}

	stab := noStabBonus
	if battle.HasSTAB(move.Type, user) {
		stab = stabBonus
	}
	accuracy := move.Accuracy * 100 / battle.MaxAccuracy

	return move.Power * stab * effect.Percent() * accuracy / (100 * 100 * 100)
}

// moveScore は1つの技の評価。威力のある技と、効果そのものが内容の技を分けて見る。
func (b *Bot) moveScore(move battle.Move, user battle.Pokemon, userTyping battle.Typing, target opponentView) int {
	if move.Power > 0 {
		score := damageScore(move, userTyping, target)
		if move.Effect == battle.EffectExplode {
			// 自爆は普段の攻め手にしない。倒されるのが見えてから初めて候補にする。
			if !lowHP(user, explodeHPPercent) {
				return 0
			}
			return score + explodeBonus
		}
		return score
	}

	// 威力0の技も、タイプで無効化されるなら意味がない。
	if battle.AgainstTyping(move.Type, target.typing) == battle.NoEffect {
		return 0
	}

	switch move.Effect {
	case battle.EffectSleep, battle.EffectParalyze:
		// 既に状態異常なら重ねられない。連打しないための下げ。
		if target.status != battle.NoStatus {
			return 0
		}
		return statusScore * move.Accuracy / battle.MaxAccuracy

	case battle.EffectHeal, battle.EffectRest:
		// HPが高いうちは回復しない。実機でも満タンでは失敗する。
		if !lowHP(user, healHPPercent) {
			return 0
		}
		return healScore

	case battle.EffectSpeedUp2:
		if user.Stages.Speed >= battle.StageMax {
			return 0
		}
		return boostScore

	default:
		return fallbackScore
	}
}

// lowHP は残りHPが最大HPのpercent以下かを返す。
func lowHP(p battle.Pokemon, percent int) bool {
	if p.Stats.HP <= 0 {
		return false
	}
	return p.CurrentHP*100 <= p.Stats.HP*percent
}

// bestOffense は相手へ最もよく通る、自分の使える攻撃技の相性を返す。
//
// 見るのは自分の技なので、相手の手札を覗くことにはならない。
func (b *Bot) bestOffense(p battle.Pokemon, target opponentView) battle.Effectiveness {
	best := battle.NoEffect
	for i := range p.Moves {
		slot := p.Moves[i]
		if !slot.Usable() {
			continue
		}
		move, err := b.data.LookupMove(slot.Move)
		if err != nil || move.Power == 0 {
			continue
		}
		if effect := battle.AgainstTyping(move.Type, target.typing); effect > best {
			best = effect
		}
	}
	return best
}

// incomingRisk は相手のタイプが自分へどれだけ通るかの見積もり。
//
// **相手がそのタイプの攻撃技を実際に持っているかは見ない。** 相手の技構成は
// 画面に出ない情報なので、Botも読まない（YTA-29 Design decision 4）。
// タイプだけを手掛かりにした粗い見積もりで、外れることがある。
func (b *Bot) incomingRisk(target opponentView, own battle.Typing) battle.Effectiveness {
	worst := battle.NoEffect
	for _, attack := range target.typing.List() {
		if effect := battle.AgainstTyping(attack, own); effect > worst {
			worst = effect
		}
	}
	return worst
}

// matchupScore は相手activeに対する相性の良さ。大きいほど有利。
//
// 攻めて通るぶんを足し、受けて痛いぶんを引くだけ。HPや能力値は見ない。
func (b *Bot) matchupScore(p battle.Pokemon, target opponentView) int {
	offense := b.bestOffense(p, target)
	risk := b.incomingRisk(target, b.typingOf(p.Species))
	return offense.Percent() - risk.Percent()
}
