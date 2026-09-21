package singleplayer

import (
	"fmt"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
)

// seeds は1つのroot seedから導出した、用途ごとのseed。
type seeds struct {
	// setup は6体の配布だけに使う。
	setup uint64

	// battle はBattle Engineの命中・急所・ダメージ・状態異常に使う。
	battle uint64

	// bot はBotの判断に使う。このpackageは値を渡すだけで自分では引かない。
	bot uint64
}

// deriveSeeds はroot seedから用途ごとのseedを導出する。
//
// 用途ごとに別のRandを作るため、ある用途が乱数をいくつ消費しても
// 他の列は動かない。Botが何回乱数を引いてもBattle Engineの列は変わらない。
//
// 導出値はtestで固定してある。ここを変えると同じseedから別の試合になるので、
// 変更するときはtestの固定値も併せて更新する。
func deriveSeeds(root uint64) seeds {
	stream := battle.NewRand(root)
	return seeds{
		setup:  stream.Uint64(),
		battle: stream.Uint64(),
		bot:    stream.Uint64(),
	}
}

// deal は6体を重複なくシャッフルし、3体ずつ両sideへ配る。
//
// 6体すべてを使い切るので、同じキャラクターが両sideに出ることはない。
// 並べ替えはFisher-Yatesで、引く乱数の数はrosterの体数だけで決まる。
func deal(rng battle.RNG) ([2][battle.TeamSize]battle.SpeciesID, error) {
	var trios [2][battle.TeamSize]battle.SpeciesID

	all := roster.All()
	if len(all) != len(trios)*battle.TeamSize {
		return trios, fmt.Errorf("singleplayer: roster has %d characters, need %d", len(all), len(trios)*battle.TeamSize)
	}

	ids := make([]battle.SpeciesID, len(all))
	for i, character := range all {
		ids[i] = character.ID
	}
	for i := len(ids) - 1; i > 0; i-- {
		j := rng.IntN(i + 1)
		ids[i], ids[j] = ids[j], ids[i]
	}

	for side := range trios {
		copy(trios[side][:], ids[side*battle.TeamSize:])
	}
	return trios, nil
}
