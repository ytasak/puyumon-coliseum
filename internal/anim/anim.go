// Package anim はComposite Emoji Spriteの基本アニメーションを提供する。
//
// このpackageはキャラクターの定義を持たない。持つのはアニメーションの状態だけで、
// 描画に使う transform は base transform から毎回計算し直す。
// そのためアニメーションが終わったあとに位置や大きさがずれて残ることがない。
//
// 時間の単位はEbitengineのtickに統一する。Updateが1回呼ばれるたびに1 tick進み、
// 経過フレーム数や実時間には依存しない。testからも同じ単位で進められる。
//
// Ebitengineには依存しないため、動きの検証は描画contextなしで行える。
package anim

import (
	"math"

	"github.com/ytasak/puyumon-coliseum/internal/sprite"
)

// TPS はEbitengineの既定のtick数（1秒あたり）。
// durationをtickで書くときの目安に使う。
const TPS = 60

// Motion は単発で再生するアニメーションの種類。
//
// 待機中の動き（Idle）はMotionではない。Idleは何も再生していないときに
// 常に適用される既定の動きで、Playの対象にならない。
type Motion int

const (
	// Attack は前方へ踏み込んで戻る。
	Attack Motion = iota
	// Hit は短く揺れる。
	Hit
	// Emphasis は一度大きくなって戻る。
	Emphasis
)

// String はMotionの名前を返す。画面のラベルとtestの出力に使う。
func (m Motion) String() string {
	switch m {
	case Attack:
		return "attack"
	case Hit:
		return "hit"
	case Emphasis:
		return "emphasis"
	default:
		return "unknown"
	}
}

// アニメーションの大きさと長さ。距離はcharacter-local座標（1.0 = Emojiセル1個分）で
// 持ち、base transformのScaleを掛けてscreen座標へ変換する。
// こうしておくとキャラクターを拡大縮小しても動きの見た目の比率が変わらない。
const (
	idlePeriod    = TPS     // 待機の上下動1周期
	idleRise      = 0.035   // 待機で持ち上がる高さ
	attackTicks   = TPS / 3 // 踏み込みから戻るまで
	attackReach   = 0.35    // 踏み込む距離
	hitTicks      = TPS / 4 // 揺れている時間
	hitAmplitude  = 0.09    // 揺れ幅
	hitShakes     = 3       // 揺れる往復回数
	emphasisTicks = TPS / 3 // 拡大して戻るまで
	emphasisGrow  = 0.35    // 拡大率の増分
)

// motion は単発アニメーション1種類分の定義。
//
// applyは進捗（0.0で開始、1.0で終了）からbase transformへの変化を決める。
// 純粋な関数なので、進捗を与えるだけで結果を検証できる。
type motion struct {
	ticks int
	apply func(progress float64, base sprite.Transform) sprite.Transform
}

// motions はMotionごとの定義。特定のキャラクターに依存する条件は持たない。
var motions = map[Motion]motion{
	Attack: {
		ticks: attackTicks,
		apply: func(progress float64, base sprite.Transform) sprite.Transform {
			// 0 -> 1 -> 0 と踏み込んで戻る。
			base.X += attackReach * base.Scale * math.Sin(math.Pi*progress)
			return base
		},
	},
	Hit: {
		ticks: hitTicks,
		apply: func(progress float64, base sprite.Transform) sprite.Transform {
			// 揺れ幅を時間とともに減衰させ、最後は必ず0になる。
			damping := 1 - progress
			base.X += hitAmplitude * base.Scale * damping * math.Sin(2*math.Pi*hitShakes*progress)
			return base
		},
	},
	Emphasis: {
		ticks: emphasisTicks,
		apply: func(progress float64, base sprite.Transform) sprite.Transform {
			base.Scale *= 1 + emphasisGrow*math.Sin(math.Pi*progress)
			return base
		},
	},
}

// Player は1体分のアニメーション状態。
//
// Character定義もbase transformも持たない。描画に使うtransformは
// Transformへbase transformを渡して都度計算する。
//
// 単発アニメーションは同時に再生しない。再生中に別の単発アニメーションを
// 要求された場合は順番待ちへ積み、現在のものが終わってから再生する。
type Player struct {
	// idleTicks は待機の動きに使う通算tick。単発再生中も進み続けるので、
	// 単発が終わったときに待機の位相が飛ばない。
	idleTicks int

	// playing は再生中の単発アニメーション。nilなら待機のみ。
	playing *Motion
	// elapsed は再生中の単発アニメーションの経過tick。
	elapsed int
	// queue は順番待ちの単発アニメーション。
	queue []Motion
}

// Play は単発アニメーションを再生する。
//
// 再生中であれば順番待ちへ積む。既に再生中のものを中断することはない。
func (p *Player) Play(m Motion) {
	if _, ok := motions[m]; !ok {
		return
	}
	if p.playing == nil {
		p.start(m)
		return
	}
	p.queue = append(p.queue, m)
}

// start はmを再生中にする。
func (p *Player) start(m Motion) {
	playing := m
	p.playing = &playing
	p.elapsed = 0
}

// Update は1 tick進める。Ebitengineのゲームループから毎tick呼ぶ。
//
// 描画は行わない。Drawの責務はここへ入らない。
func (p *Player) Update() {
	p.idleTicks++

	if p.playing == nil {
		if len(p.queue) > 0 {
			p.start(p.queue[0])
			p.queue = p.queue[1:]
		}
		return
	}

	p.elapsed++
	if p.elapsed < motions[*p.playing].ticks {
		return
	}

	// 単発アニメーションが終わったら、待機へ戻すか次を始める。
	p.playing = nil
	p.elapsed = 0
	if len(p.queue) > 0 {
		p.start(p.queue[0])
		p.queue = p.queue[1:]
	}
}

// Playing は再生中の単発アニメーションを返す。再生中でなければokがfalse。
func (p *Player) Playing() (Motion, bool) {
	if p.playing == nil {
		return 0, false
	}
	return *p.playing, true
}

// Queued は順番待ちの数を返す。
func (p *Player) Queued() int {
	return len(p.queue)
}

// Transform はbaseへ現在のアニメーションを適用したtransformを返す。
//
// 毎回baseから計算し直すため、再生を繰り返しても位置や大きさがずれて
// 蓄積することはない。単発アニメーションを再生していないときは
// 待機の動きだけが乗る。
func (p *Player) Transform(base sprite.Transform) sprite.Transform {
	base = applyIdle(p.idleTicks, base)

	if p.playing == nil {
		return base
	}
	m := motions[*p.playing]
	return m.apply(float64(p.elapsed)/float64(m.ticks), base)
}

// applyIdle は待機の上下動をbaseへ乗せる。
//
// 画面上で持ち上がる向き（Yの負方向）へ動かす。1周期でちょうど元へ戻る。
func applyIdle(ticks int, base sprite.Transform) sprite.Transform {
	phase := 2 * math.Pi * float64(ticks%idlePeriod) / idlePeriod
	base.Y -= idleRise * base.Scale * (1 - math.Cos(phase)) / 2
	return base
}
