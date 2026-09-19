// Package balance は20通りのroster combinationを総当たりで回し、
// balance検証のための指標へ集計する。
//
// 対戦そのものは internal/simulation が回す。このpackageは対戦のルールを持たず、
// 行動の選び方も足さない。結果を数えて分布にするだけ。
//
// 合否は判定しない。「何%なら偏っている」という閾値はここに置かず、観測した数だけを返す。
// どこからを偏りと見るかはReportを読む側が決める。
package balance

import (
	"errors"
	"fmt"

	"github.com/ytasak/puyumon-coliseum/internal/simulation"
)

// ErrInvalidConfig は設定が取り得ない値であることを表す。
var ErrInvalidConfig = errors.New("balance: invalid config")

// Config はbatch simulationの設定。
//
// 同じConfigからは必ず同じReportが出る。再現に必要なのはこの3つの値だけなので、
// baselineを記録するときはこれを残せばよい。
type Config struct {
	// FirstSeed は最初のseed。
	FirstSeed uint64

	// Trials は1 matchupあたりの試行数。seedはFirstSeedから1ずつ増やしてTrials個使う。
	//
	// seedはmatchupをまたいで共通にする。すべてのmatchupへ同じ乱数列を配ることで、
	// combination間の差がseedの引きの差に埋もれないようにする。
	Trials int

	// MaxTurns はsimulationを打ち切るturn数。0なら simulation.DefaultMaxTurns を使う。
	//
	// simulation側の安全装置をそのまま渡すだけで、対戦のルールではない。
	// 到達した対戦は勝敗を付けず、unresolvedとして数える。
	MaxTurns int
}

// normalize は省略された値を埋めたConfigを返す。
//
// Reportへはこの結果を載せる。省略値のまま残すと、同じ設定で再実行するために
// defaultの値まで知っている必要が出てしまう。
func (c Config) normalize() (Config, error) {
	if c.Trials < 1 {
		return Config{}, fmt.Errorf("%w: trials %d is less than 1", ErrInvalidConfig, c.Trials)
	}
	if c.MaxTurns < 0 {
		return Config{}, fmt.Errorf("%w: max turns %d is negative", ErrInvalidConfig, c.MaxTurns)
	}
	if c.MaxTurns == 0 {
		c.MaxTurns = simulation.DefaultMaxTurns
	}
	return c, nil
}

// seed はtrial番目の試行に使うseedを返す。
func (c Config) seed(trial int) uint64 {
	return c.FirstSeed + uint64(trial)
}
