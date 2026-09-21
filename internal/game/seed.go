package game

import "github.com/ytasak/puyumon-coliseum/internal/battle"

// nextSeed は次の対戦に使うseedを返す。
//
// 「もう一度」で新しい対戦を始めるときに使う。起動時のseedから決まった規則で
// 導くので、同じseedで起動すれば何試合目でも同じ対戦になる。
//
// グローバルな乱数状態には触らない。Battle Engineが持つsplitmix64をそのまま
// 使うため、新しい依存も増えない。
func nextSeed(seed uint64) uint64 {
	return battle.NewRand(seed).Uint64()
}
