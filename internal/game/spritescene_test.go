package game

import (
	"image"
	"slices"
	"testing"
	"unicode"

	"github.com/ytasak/puyumon-coliseum/internal/anim"
	"github.com/ytasak/puyumon-coliseum/internal/emoji"
)

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
func TestNassyIsOrderedByZ(t *testing.T) {
	t.Parallel()

	for i := 1; i < len(nassy.Parts); i++ {
		if prev, cur := nassy.Parts[i-1].Z, nassy.Parts[i].Z; prev > cur {
			t.Errorf("part %d has Z=%d after Z=%d; parts must be ordered back to front", i, cur, prev)
		}
	}
}

// 画面に出るEmojiはすべて同梱フォントでカラー描画できる必要がある。
// 実機でEmojiが欠けたりmonochromeになったりしたときに、
// フォント側の問題ではないと切り分けられるようにする。
func TestSceneEmojiUseColorGlyphs(t *testing.T) {
	t.Parallel()

	set, err := emoji.New()
	if err != nil {
		t.Fatalf("emoji.New() returned error: %v", err)
	}

	for _, p := range nassy.Parts {
		if !set.IsColorGlyph(p.Emoji) {
			t.Errorf("character part %q is not a color glyph", p.Emoji)
		}
	}
	for _, action := range actions {
		if !set.IsColorGlyph(action.particle) {
			t.Errorf("%s: particle %q is not a color glyph", action.label, action.particle)
		}
	}
	if !set.IsColorGlyph(idleParticle) {
		t.Errorf("idle particle %q is not a color glyph", idleParticle)
	}
}

// タップで Attack / Hit / Emphasis のすべてを発火できる。
func TestActionsCoverEveryMotion(t *testing.T) {
	t.Parallel()

	playable := map[anim.Motion]bool{}
	for _, action := range actions {
		playable[action.motion] = true
	}

	for _, m := range []anim.Motion{anim.Attack, anim.Hit, anim.Emphasis} {
		if !playable[m] {
			t.Errorf("no action plays %v", m)
		}
	}
}

// particleにはIssueが指定した4種を使う。
func TestSceneParticlesAreTheRequiredEmoji(t *testing.T) {
	t.Parallel()

	got := []string{idleParticle}
	for _, action := range actions {
		got = append(got, action.particle)
	}
	slices.Sort(got)

	want := []string{"⚡", "❄️", "💥", "💤"}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("scene particles = %v, want %v", got, want)
	}
}

// タップ領域は画面内に収まり、互いに重ならない。
// 重なっていると、どのactionが反応したのかで入力のずれを判断できなくなる。
func TestActionRectsFitOnScreenAndDoNotOverlap(t *testing.T) {
	t.Parallel()

	screen := image.Rect(0, 0, LogicalWidth, LogicalHeight)
	for i, action := range actions {
		if !action.rect.In(screen) {
			t.Errorf("%s: rect %v is outside the logical screen %v", action.label, action.rect, screen)
		}
		for _, other := range actions[i+1:] {
			if action.rect.Overlaps(other.rect) {
				t.Errorf("%s and %s overlap: %v, %v", action.label, other.label, action.rect, other.rect)
			}
		}
	}
}

// タップ領域は実機の指で押せる大きさを確保する。
func TestActionRectsAreLargeEnoughToTap(t *testing.T) {
	t.Parallel()

	// 論理座標での最小の一辺。実機では画面幅へ引き伸ばされるため、
	// これを下回らなければ指で押せる。
	const minSide = 44

	for _, action := range actions {
		if action.rect.Dx() < minSide || action.rect.Dy() < minSide {
			t.Errorf("%s: rect %v is smaller than %dx%d", action.label, action.rect, minSide, minSide)
		}
	}
}

// タップ位置の判定は領域の中だけで当たる。
func TestActionAtMatchesOnlyItsOwnRect(t *testing.T) {
	t.Parallel()

	for want, action := range actions {
		center := image.Pt(
			action.rect.Min.X+action.rect.Dx()/2,
			action.rect.Min.Y+action.rect.Dy()/2,
		)
		got, ok := actionAt(center)
		if !ok || got != want {
			t.Errorf("actionAt(center of %s) = (%d, %v), want (%d, true)", action.label, got, ok, want)
		}
	}

	// キャラクターの位置はどのタップ領域にも当たらない。
	if _, ok := actionAt(image.Pt(characterX, characterY)); ok {
		t.Error("the character position falls inside an action rect")
	}
}

// 操作しなくても待機のparticleが出て、寿命が尽きた分は消える。
// 実機で「止まっているのか動いているのか」を判断する手掛かりになる。
func TestIdleParticlesAppearWithoutInput(t *testing.T) {
	t.Parallel()

	g := newGame(t)

	var spawned bool
	for i := 0; i < idleParticlePeriod*3; i++ {
		if err := g.Update(); err != nil {
			t.Fatalf("Update() returned error: %v", err)
		}
		if g.particles.Len() > 0 {
			spawned = true
		}
		if got, limit := g.particles.Len(), 2; got > limit {
			t.Fatalf("tick %d: %d particles alive, want at most %d; they are not being retired", i, got, limit)
		}
	}

	if !spawned {
		t.Error("no idle particle was ever spawned")
	}
}

// 画面のラベルはebitenutil.DebugPrintAtの組み込みASCIIフォントで描くため、
// 非ASCII文字が混ざると表示できない。
func TestActionLabelsAreASCIIOnly(t *testing.T) {
	t.Parallel()

	for _, action := range actions {
		for _, r := range action.label {
			if r > unicode.MaxASCII {
				t.Errorf("label %q contains non-ASCII rune %q", action.label, r)
			}
		}
	}
}
