package battle

// MaxAccuracy は命中率の最大値。Generation Iは命中率を0〜255で保持する。
const MaxAccuracy = 255

// AccuracyPercent は百分率をGeneration Iの0〜255表現へ直す。
//
// 実機のmove dataも同じ計算で持っている（例: 100% = 255、95% = 242、75% = 191）。
func AccuracyPercent(percent int) int {
	if percent < 0 {
		return 0
	}
	accuracy := percent * MaxAccuracy / 100
	if accuracy > MaxAccuracy {
		return MaxAccuracy
	}
	return accuracy
}

// HitChance は命中stageと回避stageを反映した命中率（0〜255）を返す。
//
// Generation Iは技のbase accuracyへ、命中stageの倍率と回避stageの倍率を
// この順に掛ける。掛けるたびに切り捨て、0になったら1へ引き上げる。
// 最後に255で頭打ちにする。
//
// 回避stageは符号を反転したstageの倍率として効く。回避が+2なら命中率は0.5倍になる。
func HitChance(baseAccuracy, accuracyStage, evasionStage int) int {
	chance := baseAccuracy
	for _, stage := range [2]int{accuracyStage, -evasionStage} {
		chance = ApplyStage(chance, stage)
		if chance < 1 {
			chance = 1
		}
	}
	if chance > MaxAccuracy {
		return MaxAccuracy
	}
	return chance
}

// Hits は命中したかどうかを返す。
//
// 命中率が最大（255）なら必ず当たる。実機では乱数が255のときに外れるため
// どんな技でも1/256で外れるが、この1/256 missはIssueの指示により再現しない。
// 再現する場合はBattle Rules Specificationを先に更新する。
//
// 乱数は結果によらず常に1つ消費する。
func Hits(rng RNG, baseAccuracy, accuracyStage, evasionStage int) bool {
	chance := HitChance(baseAccuracy, accuracyStage, evasionStage)
	roll := rng.IntN(MaxAccuracy + 1)
	if chance >= MaxAccuracy {
		return true
	}
	return roll < chance
}
