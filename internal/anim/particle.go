package anim

import "github.com/ytasak/puyumon-coliseum/internal/sprite"

// particleのふるまい。
const (
	// particleTicks はparticleが表示されている時間。
	particleTicks = TPS * 5 / 6
	// particleRise は生存中に上昇する距離。particleの大きさに対する倍率。
	particleRise = 0.9
	// particleShrinkFrom はこの進捗から縮み始める。縮み切ったところで消える。
	particleShrinkFrom = 0.6
)

// particle は表示中のparticle 1つ分の状態。
//
// 位置や大きさの変化は保持せず、経過tickから毎回導出する。
// Playerと同じく、状態を足し込んでいかないのでずれが蓄積しない。
type particle struct {
	// character はEmoji 1つだけのCharacter。生成時に1度だけ作る。
	character sprite.Character
	// x, y は生成位置（screen座標）。
	x, y float64
	// size はparticleの大きさ。Emojiセル1個分にあたるpixel数。
	size float64

	elapsed  int
	lifetime int
}

// Particles はEmoji particleの生成・更新・破棄をまとめて扱う。
//
// 描画は行わず、描画に必要な情報を返すだけにとどめる。
// Ebitengineへ依存しないので、生成から消滅までをtestで追える。
type Particles struct {
	items []particle
}

// Spawn は指定位置へEmojiのparticleを生成する。
//
// sizeはEmojiセル1個分にあたるpixel数。一定時間後に自動で消える。
func (ps *Particles) Spawn(emoji string, x, y, size float64) {
	ps.items = append(ps.items, particle{
		character: sprite.NewCharacter(sprite.Part{Emoji: emoji, Scale: 1}),
		x:         x,
		y:         y,
		size:      size,
		lifetime:  particleTicks,
	})
}

// Update は1 tick進め、寿命の尽きたparticleを取り除く。
//
// 取り除きは元のsliceを詰め直すだけなので、定常状態では追加の確保が起きない。
func (ps *Particles) Update() {
	live := ps.items[:0]
	for _, it := range ps.items {
		it.elapsed++
		if it.elapsed < it.lifetime {
			live = append(live, it)
		}
	}
	ps.items = live
}

// Len は生存しているparticleの数を返す。
func (ps *Particles) Len() int {
	return len(ps.items)
}

// At はi番目のparticleを描画するための情報を返す。
//
// 返り値をそのまま sprite.Renderer.Draw へ渡せる。
func (ps *Particles) At(i int) (sprite.Character, sprite.Transform) {
	it := &ps.items[i]
	progress := float64(it.elapsed) / float64(it.lifetime)

	return it.character, sprite.Transform{
		X:     it.x,
		Y:     it.y - particleRise*it.size*progress,
		Scale: it.size * particleShrink(progress),
	}
}

// particleShrink は進捗に対する大きさの倍率を返す。
// 終盤だけ縮み、消える瞬間に0になる。
func particleShrink(progress float64) float64 {
	if progress < particleShrinkFrom {
		return 1
	}
	return (1 - progress) / (1 - particleShrinkFrom)
}
