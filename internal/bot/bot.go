// Package bot はSingle Player用のBot v1 policyを持つ。
//
// 目的は人間に勝つことではなく、**固定6体の役割と交代が最低限成立し、
// playtestの相手として使えること**。minimaxもrolloutも行わず、
// タイプ相性と手元の技だけを見て粗く決める。
//
// 対戦のルールは持たない。damageもstatusも行動順も internal/battle が決め、
// このpackageは「どのActionを選ぶか」だけを返す。Botはsessionの内部ではなく
// 外側のactorで、player側と同じcommand APIから注入される。
//
// # 相手について見てよい情報
//
// Botは**相手の技構成を読まない**。固定rosterなのでdataからは引けるが、
// 現在のUIは相手の4技を表示しないため、Botだけが読めると
// プレイヤーから説明のつかない択の通し方になる（YTA-29 Design decision 4）。
// 相手について見るのはopponentViewが持つぶん、つまり画面から読める情報だけ。
//
// # 乱数
//
// Botの乱数はBattle Engineの乱数とは別の列で、命中・急所・ダメージには影響しない。
// 同じ状態・同じseedなら必ず同じActionを返す。
package bot

import "github.com/ytasak/puyumon-coliseum/internal/battle"

// Bot は1試合ぶんの判断を持つ。
//
// 交代のcooldownという状態を持つため、試合ごとに1つ作る。
// 使い回すと前の試合の状態が残る。
type Bot struct {
	data battle.Data
	rng  battle.RNG

	// justSwitchedIn は場に出た直後か。trueのあいだは自発的な交代をしない。
	// 出した直後にまた引っ込む往復を防ぐためだけの状態で、
	// Battle Engineの状態には持たせない。
	justSwitchedIn bool
}

// New はseedからBotを作る。
//
// seedはBattle Engineの乱数とは別の列にすること。session経由なら
// singleplayer.Session.BotSeed() がその値を返す。
func New(data battle.Data, seed uint64) *Bot {
	return &Bot{data: data, rng: battle.NewRand(seed)}
}

// Lead は最初に場へ出す1体のteam indexを返す。
//
// 配られた3体それぞれについて、相手3体すべてとの相性を足し合わせ、
// 合計が最も良いものを選ぶ。同点はBotの乱数で決める。
func (b *Bot) Lead(own, opponent [battle.TeamSize]battle.Pokemon) int {
	views := make([]opponentView, 0, battle.TeamSize)
	for _, p := range opponent {
		views = append(views, b.viewOf(p))
	}

	scores := make([]int, battle.TeamSize)
	for i, p := range own {
		for _, view := range views {
			scores[i] += b.matchupScore(p, view)
		}
	}
	return b.pickBest(scores)
}

// Action は通常のturnの行動を選ぶ。
//
// 使える技が1つも無ければStruggle、明らかに相性が悪くて控えで改善するなら交代、
// それ以外は最も点の高い技を出す。
func (b *Bot) Action(state battle.BattleState, side battle.Side) battle.Action {
	player := state.Players[side]
	active := player.Team[player.Active]

	if !active.HasUsableMove() {
		b.justSwitchedIn = false
		return battle.StruggleAction{}
	}

	target := b.viewOf(*state.Players[side.Opponent()].ActivePokemon())

	switch {
	case b.justSwitchedIn:
		// 出てきた直後のturnは交代を考えない。ここで解除するので、
		// 次のturnからはまた交代できる。
		b.justSwitchedIn = false
	default:
		if action, ok := b.switchAway(player, active, target); ok {
			b.justSwitchedIn = true
			return action
		}
	}
	return b.bestMove(active, target)
}

// Replacement は戦闘不能のあとに場へ出す控えを選ぶ。
//
// 生存している控えだけを候補にするので、返すのは必ず合法な交代になる。
func (b *Bot) Replacement(state battle.BattleState, side battle.Side) battle.SwitchAction {
	player := state.Players[side]
	reserve := player.Reserve()
	if len(reserve) == 0 {
		// 交代が必要な状態でだけ呼ばれる前提。合法な候補が無いので、
		// 呼び出し側がerrorとして扱えるようそのまま返す。
		return battle.SwitchAction{}
	}

	target := b.viewOf(*state.Players[side.Opponent()].ActivePokemon())
	scores := make([]int, len(reserve))
	for i, index := range reserve {
		scores[i] = b.matchupScore(player.Team[index], target)
	}

	// 出した直後に引っ込まないよう、交代と同じ扱いにする。
	b.justSwitchedIn = true
	return battle.SwitchAction{Target: reserve[b.pickBest(scores)]}
}

// switchAway は相性が悪いときに交代先を返す。
//
// 毎turn最善の交代を探すことはしない。「今が明らかに悪く」「明らかに良くなる
// 控えがいる」ときだけ動く。
func (b *Bot) switchAway(player battle.Player, active battle.Pokemon, target opponentView) (battle.SwitchAction, bool) {
	if !b.badMatchup(active, target) {
		return battle.SwitchAction{}, false
	}

	reserve := player.Reserve()
	current := b.matchupScore(active, target)

	candidates := make([]int, 0, len(reserve))
	scores := make([]int, 0, len(reserve))
	for _, index := range reserve {
		candidate := player.Team[index]
		score := b.matchupScore(candidate, target)
		if score <= current || !b.improves(candidate, target) {
			continue
		}
		candidates = append(candidates, index)
		scores = append(scores, score)
	}
	if len(candidates) == 0 {
		return battle.SwitchAction{}, false
	}
	return battle.SwitchAction{Target: candidates[b.pickBest(scores)]}, true
}

// badMatchup は今の相手に対して明らかに分が悪いかを返す。
//
// 相手のタイプがこちらへ2倍で通り、かつこちらの技が等倍までしか通らない状態を指す。
func (b *Bot) badMatchup(active battle.Pokemon, target opponentView) bool {
	risk := b.incomingRisk(target, b.typingOf(active.Species))
	return risk >= battle.SuperEffective && b.bestOffense(active, target) <= battle.Neutral
}

// improves は控えが今より明らかに良くなるかを返す。
//
// 2倍で殴り返せるか、相手のタイプを半減以下で受けられるかのどちらか。
func (b *Bot) improves(candidate battle.Pokemon, target opponentView) bool {
	if b.bestOffense(candidate, target) >= battle.SuperEffective {
		return true
	}
	return b.incomingRisk(target, b.typingOf(candidate.Species)) <= battle.NotVeryEffective
}

// bestMove は最も点の高い技を選ぶ。
func (b *Bot) bestMove(active battle.Pokemon, target opponentView) battle.Action {
	typing := b.typingOf(active.Species)

	slots := make([]int, 0, battle.MoveSlots)
	scores := make([]int, 0, battle.MoveSlots)
	for i := range active.Moves {
		slot := active.Moves[i]
		if !slot.Usable() {
			continue
		}
		move, err := b.data.LookupMove(slot.Move)
		if err != nil {
			continue
		}
		slots = append(slots, i)
		scores = append(scores, b.moveScore(move, active, typing, target))
	}

	if len(slots) == 0 {
		// 定義を引けない技しか残っていない場合。合法な選択肢を返すことを優先する。
		for i := range active.Moves {
			if active.Moves[i].Usable() {
				return battle.MoveAction{Slot: i}
			}
		}
		return battle.StruggleAction{}
	}
	return battle.MoveAction{Slot: slots[b.pickBest(scores)]}
}

// pickBest は最も点の高い候補のindexを返す。同点はBotの乱数で選ぶ。
//
// 候補の並び順だけで決めてしまうと、data定義の並びを変えただけで
// 挙動が変わってしまうため、同点は明示的に乱数で割る。
func (b *Bot) pickBest(scores []int) int {
	if len(scores) == 0 {
		return -1
	}

	best := scores[0]
	tied := 1
	for _, score := range scores[1:] {
		switch {
		case score > best:
			best, tied = score, 1
		case score == best:
			tied++
		}
	}
	if tied == 1 {
		for i, score := range scores {
			if score == best {
				return i
			}
		}
	}

	pick := b.rng.IntN(tied)
	for i, score := range scores {
		if score != best {
			continue
		}
		if pick == 0 {
			return i
		}
		pick--
	}
	return len(scores) - 1
}
