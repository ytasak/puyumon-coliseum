package balance

import (
	"fmt"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/simulation"
)

// Policy は行動の選び方。
//
// 結果の読み方が方針によって変わるため、Reportへそのまま残す。
// 値は記録に残る識別子なので、一度決めたら変えない。
type Policy string

const (
	// PolicyFirstUsable は使える先頭の技を選ぶ。
	//
	// PPが切れるまで同じ技を撃ち続けるため、結果は「slot 0の質」へ寄る。
	PolicyFirstUsable Policy = "first_usable"

	// PolicyUniformUsable は使える技から一様ランダムに選ぶ。
	//
	// slot 0への偏りを崩し、rosterの構造的な偏りと行動方針由来の偏りを
	// 見分けるための対照用。賢いBotではない。
	PolicyUniformUsable Policy = "uniform_usable"
)

// DefaultPolicy はPolicyを指定しなかったときに使う方針。
const DefaultPolicy = PolicyFirstUsable

// policySeedBase は行動選択のseedを対戦のseedから引き離すための定数。
//
// 同じ値にすると、行動選択とダメージ・命中がまったく同じ乱数列を辿ることになる。
// splitmix64のgammaをそのまま借りているだけで、値自体に意味は無い。
const policySeedBase = 0x9e3779b97f4a7c15

// policySeed は行動選択に使うseedを返す。
//
// 対戦のseedとsideだけから決まるので、Configのseed範囲と試行数が決まれば
// 全matchupの乱数が決まる。両sideへ別のseedを配るのは、同じ乱数列だと
// 両者が同じ位置の技を選び続けてしまうため。
func policySeed(battleSeed uint64, side battle.Side) uint64 {
	return policySeedBase + battleSeed*2 + uint64(side)
}

// firstUsableFactory はFirstUsableを返すChooserFactory。
//
// FirstUsableは状態を持たないので、Runごとに作り直す必要が無い。
var firstUsableFactory simulation.ChooserFactory = func() simulation.Chooser { return simulation.FirstUsable{} }

// choosers はこの対戦で使うChooserの作り方を返す。
//
// 呼ばれる時点でPolicyはnormalizeを通っている。
func (p Policy) choosers(battleSeed uint64) [2]simulation.ChooserFactory {
	var factories [2]simulation.ChooserFactory
	for _, side := range sides {
		if p == PolicyUniformUsable {
			factories[side] = simulation.UniformUsablePolicy(policySeed(battleSeed, side))
			continue
		}
		factories[side] = firstUsableFactory
	}
	return factories
}

// validate は定義済みの方針かを確かめる。
func (p Policy) validate() error {
	switch p {
	case PolicyFirstUsable, PolicyUniformUsable:
		return nil
	default:
		return fmt.Errorf("%w: unknown policy %q", ErrInvalidConfig, p)
	}
}
