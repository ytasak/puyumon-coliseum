package game

import (
	"slices"
	"testing"
	"unicode"

	"github.com/ytasak/puyumon-coliseum/internal/anim"
	"github.com/ytasak/puyumon-coliseum/internal/emoji"
)

func newEmojiSet(t *testing.T) *emoji.Set {
	t.Helper()

	set, err := emoji.New()
	if err != nil {
		t.Fatalf("emoji.New() returned error: %v", err)
	}
	return set
}

// ナッシー型はIssueが指定した素材（body 🌴 / faces 🥺 😫 🤪）で構成する。
func TestNassyIsBuiltFromTheSpecifiedMaterials(t *testing.T) {
	t.Parallel()

	var got []string
	for _, p := range nassy.Parts {
		got = append(got, p.Emoji)
	}
	slices.Sort(got)

	want := []string{"🌴", "🤪", "😫", "🥺"}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("nassy parts = %v, want %v", got, want)
	}

	if body := nassy.Parts[0]; body.Emoji != "🌴" {
		t.Errorf("first drawn part = %q, want the body %q", body.Emoji, "🌴")
	}
}

// 描画順はZの昇順で、bodyが最初（もっとも奥）に来る。
func TestDemoCharactersAreOrderedByZ(t *testing.T) {
	t.Parallel()

	for _, demo := range spriteDemos {
		t.Run(demo.label, func(t *testing.T) {
			for i := 1; i < len(demo.character.Parts); i++ {
				if prev, cur := demo.character.Parts[i-1].Z, demo.character.Parts[i].Z; prev > cur {
					t.Errorf("part %d has Z=%d after Z=%d; parts must be ordered back to front", i, cur, prev)
				}
			}
		})
	}
}

// キャラクターにもparticleにも、同梱フォントでカラー描画できるEmojiだけを使う。
func TestDemoEmojiUseColorGlyphs(t *testing.T) {
	t.Parallel()

	set := newEmojiSet(t)
	for _, demo := range spriteDemos {
		for _, p := range demo.character.Parts {
			if !set.IsColorGlyph(p.Emoji) {
				t.Errorf("%s: character part %q is not a color glyph", demo.label, p.Emoji)
			}
		}
		if !set.IsColorGlyph(demo.particle) {
			t.Errorf("%s: particle %q is not a color glyph", demo.label, demo.particle)
		}
	}
}

// Scaleが0の部品は見えない。定義の書き間違いを拾う。
func TestDemoCharacterPartsHavePositiveScale(t *testing.T) {
	t.Parallel()

	for _, demo := range spriteDemos {
		for _, p := range demo.character.Parts {
			if p.Scale <= 0 {
				t.Errorf("%s: part %q has Scale = %v, want > 0", demo.label, p.Emoji, p.Scale)
			}
		}
	}
}

// 画面でIdle / Attack / Hit / Emphasis のすべてを確認できる。
func TestDemoCoversEveryAnimationPrimitive(t *testing.T) {
	t.Parallel()

	var idleOnly bool
	played := map[anim.Motion]bool{}
	for _, demo := range spriteDemos {
		if demo.playMotion {
			played[demo.motion] = true
		} else {
			idleOnly = true
		}
	}

	if !idleOnly {
		t.Error("no demo shows the idle motion on its own")
	}
	for _, m := range []anim.Motion{anim.Attack, anim.Hit, anim.Emphasis} {
		if !played[m] {
			t.Errorf("no demo plays %v", m)
		}
	}
}

// particleにはIssueが指定した4種を使う。
func TestDemoParticlesAreTheRequiredEmoji(t *testing.T) {
	t.Parallel()

	var got []string
	for _, demo := range spriteDemos {
		got = append(got, demo.particle)
	}
	slices.Sort(got)

	want := []string{"⚡", "❄️", "💥", "💤"}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("demo particles = %v, want %v", got, want)
	}
}

// 各demoは1周期にちょうど1回、隣と重ならないタイミングで動き出す。
func TestDemoTriggersAreStaggeredWithinOnePeriod(t *testing.T) {
	t.Parallel()

	counts := make([]int, len(spriteDemos))
	for tick := uint64(0); tick < demoPeriod; tick++ {
		triggered := 0
		for i := range spriteDemos {
			if demoTriggers(i, tick) {
				counts[i]++
				triggered++
			}
		}
		if triggered > 1 {
			t.Errorf("tick %d triggers %d demos at once; they should be staggered", tick, triggered)
		}
	}

	for i, got := range counts {
		if got != 1 {
			t.Errorf("demo %q triggered %d times per period, want 1", spriteDemos[i].label, got)
		}
	}
}

// ゲームループを回すとparticleが出て、寿命が尽きた分は消える。
func TestUpdateSpawnsAndRetiresParticles(t *testing.T) {
	t.Parallel()

	g := newGame(t)
	if got := g.particles.Len(); got != 0 {
		t.Fatalf("particles before the first update = %d, want 0", got)
	}

	var spawned bool
	for i := 0; i < demoPeriod*4; i++ {
		if err := g.Update(); err != nil {
			t.Fatalf("Update() returned error: %v", err)
		}
		if g.particles.Len() > 0 {
			spawned = true
		}
		if got, limit := g.particles.Len(), len(spriteDemos); got > limit {
			t.Fatalf("tick %d: %d particles alive, want at most %d; they are not being retired", i, got, limit)
		}
	}

	if !spawned {
		t.Error("no particle was ever spawned")
	}
}

// 画面のラベルはebitenutil.DebugPrintAtの組み込みASCIIフォントで描くため、
// 非ASCII文字が混ざると表示できない。
func TestSceneLabelsAreASCIIOnly(t *testing.T) {
	t.Parallel()

	for _, demo := range spriteDemos {
		for _, r := range demo.label {
			if r > unicode.MaxASCII {
				t.Errorf("label %q contains non-ASCII rune %q", demo.label, r)
			}
		}
	}
}
