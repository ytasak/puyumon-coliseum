package battle

// RNG はBattle Engineが使う乱数。
//
// engineはglobalな乱数状態を触らず、必要な乱数をすべてこのinterface経由で得る。
// そのため同じ初期状態・同じAction列・同じseedから常に同じ結果を再現できる。
// testでは決められた値を返す実装へ差し替えられる。
type RNG interface {
	// IntN は0以上n未満の整数を返す。nが1未満ならpanicする。
	IntN(n int) int
}

// Rand はseedから再現できるRNGの実装。
//
// 乱数列はsplitmix64で生成する。標準ライブラリを使わず自前で持つのは、
// 生成列をこのリポジトリのコードだけで固定するため。golden testが
// Goのversion差で壊れないことを優先している。
//
// 暗号用途には使わない。対戦の再現性のためのもの。
type Rand struct {
	state uint64
}

// splitmix64の定数。原典の実装に合わせる。
const (
	splitmixGamma = 0x9e3779b97f4a7c15
	splitmixMixA  = 0xbf58476d1ce4e5b9
	splitmixMixB  = 0x94d049bb133111eb
)

// NewRand はseedから始まるRandを返す。同じseedからは常に同じ列が出る。
func NewRand(seed uint64) *Rand {
	return &Rand{state: seed}
}

// Uint64 は次の64bit乱数を返す。
func (r *Rand) Uint64() uint64 {
	r.state += splitmixGamma
	z := r.state
	z = (z ^ (z >> 30)) * splitmixMixA
	z = (z ^ (z >> 27)) * splitmixMixB
	return z ^ (z >> 31)
}

// IntN は0以上n未満の整数を返す。
//
// 単純な剰余では値の範囲によって偏りが出るため、剰余が均等にならない先頭部分を
// 捨ててから求める。捨てたときはUint64を引き直すので、1回の呼び出しが消費する
// 乱数は1つとは限らない。棄却が起きる確率は2^64をnで割った余りの分で、
// nが小さければ極めて稀だが0ではない。
//
// 消費数が変わっても再現性は失われない。同じseedへ同じ順序で同じ呼び出しをすれば、
// 棄却が起きる位置まで含めて必ず同じ列になる。
func (r *Rand) IntN(n int) int {
	if n < 1 {
		panic("battle: IntN requires n >= 1")
	}

	limit := uint64(n)
	// 2^64をlimitで割った余り。ここまでの値を捨てると残りがlimitの倍数になる。
	threshold := (^uint64(0) - limit + 1) % limit
	for {
		if v := r.Uint64(); v >= threshold {
			return int(v % limit)
		}
	}
}

// RandがRNGを満たすことをcompile時に確かめる。
var _ RNG = (*Rand)(nil)
