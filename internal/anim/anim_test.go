package anim

import (
	"math"
	"testing"

	"github.com/ytasak/puyumon-coliseum/internal/sprite"
)

func baseTransform() sprite.Transform {
	return sprite.Transform{X: 320, Y: 180, Scale: 100}
}

func assertSameTransform(t *testing.T, got, want sprite.Transform, context string) {
	t.Helper()

	if got != want {
		t.Errorf("%s: transform = %+v, want %+v", context, got, want)
	}
}

// 何も再生していない開始直後は、base transformがそのまま出てくる。
func TestTransformStartsAtBase(t *testing.T) {
	t.Parallel()

	var p Player
	assertSameTransform(t, p.Transform(baseTransform()), baseTransform(), "at rest")
}

// 待機の動きは1周期でちょうどbaseへ戻る。
func TestIdleReturnsToBaseEveryCycle(t *testing.T) {
	t.Parallel()

	var p Player
	for cycle := 1; cycle <= 3; cycle++ {
		for i := 0; i < idlePeriod; i++ {
			p.Update()
		}
		assertSameTransform(t, p.Transform(baseTransform()), baseTransform(), "after idle cycle")
	}
}

// 待機の動きは上下だけで、しかも決めた振れ幅を超えない。
func TestIdleOnlyMovesVerticallyWithinItsAmplitude(t *testing.T) {
	t.Parallel()

	var p Player
	base := baseTransform()
	limit := idleRise * base.Scale

	for i := 0; i < idlePeriod*2; i++ {
		got := p.Transform(base)
		if got.X != base.X || got.Scale != base.Scale || got.Rotation != base.Rotation {
			t.Fatalf("tick %d: idle changed something other than Y: %+v", i, got)
		}
		if offset := base.Y - got.Y; offset < 0 || offset > limit {
			t.Fatalf("tick %d: idle offset = %v, want within [0, %v]", i, offset, limit)
		}
		p.Update()
	}
}

// 単発アニメーションを一通り再生しても、待機の位相が戻る時点でbaseへ完全に戻る。
// これが崩れると再生のたびに位置や大きさがずれて蓄積する。
func TestNoDriftAfterPlayingEveryMotion(t *testing.T) {
	t.Parallel()

	var p Player
	p.Play(Attack)
	p.Play(Hit)
	p.Play(Emphasis)

	total := motions[Attack].ticks + motions[Hit].ticks + motions[Emphasis].ticks
	if total > idlePeriod {
		t.Fatalf("test assumes every motion finishes within one idle cycle: %d > %d", total, idlePeriod)
	}

	// 待機の位相がちょうど一周する長さまで進める。
	for i := 0; i < idlePeriod*2; i++ {
		p.Update()
	}

	if _, playing := p.Playing(); playing {
		t.Fatal("a motion is still playing after every duration elapsed")
	}
	assertSameTransform(t, p.Transform(baseTransform()), baseTransform(), "after all motions")
}

// 同じ単発アニメーションを何度繰り返してもbaseへ戻る。
func TestNoDriftAfterRepeatingTheSameMotion(t *testing.T) {
	t.Parallel()

	for _, m := range []Motion{Attack, Hit, Emphasis} {
		t.Run(m.String(), func(t *testing.T) {
			var p Player
			for i := 0; i < 10; i++ {
				p.Play(m)
				for j := 0; j < idlePeriod; j++ {
					p.Update()
				}
			}
			assertSameTransform(t, p.Transform(baseTransform()), baseTransform(), "after repeats")
		})
	}
}

// 単発アニメーションは同時に再生せず、要求された順に再生する。
func TestMotionsAreQueuedInsteadOfPlayedTogether(t *testing.T) {
	t.Parallel()

	var p Player
	p.Play(Attack)
	p.Play(Hit)
	p.Play(Emphasis)

	if got, ok := p.Playing(); !ok || got != Attack {
		t.Fatalf("playing = (%v, %v), want (attack, true)", got, ok)
	}
	if got, want := p.Queued(), 2; got != want {
		t.Fatalf("queued = %d, want %d", got, want)
	}

	want := []Motion{Attack, Hit, Emphasis}
	for i, m := range want {
		for j := 0; j < motions[m].ticks; j++ {
			if got, ok := p.Playing(); !ok || got != m {
				t.Fatalf("step %d tick %d: playing = (%v, %v), want (%v, true)", i, j, got, ok, m)
			}
			p.Update()
		}
	}

	if _, ok := p.Playing(); ok {
		t.Error("something is still playing after the queue drained")
	}
	if got := p.Queued(); got != 0 {
		t.Errorf("queued = %d, want 0", got)
	}
}

// 各Motionが変えるのは自分の担当する項目だけ。
func TestEachMotionChangesOnlyItsOwnChannel(t *testing.T) {
	t.Parallel()

	base := baseTransform()
	tests := []struct {
		motion     Motion
		wantMovesX bool
		wantScales bool
	}{
		{motion: Attack, wantMovesX: true},
		{motion: Hit, wantMovesX: true},
		{motion: Emphasis, wantScales: true},
	}

	for _, tc := range tests {
		t.Run(tc.motion.String(), func(t *testing.T) {
			m := motions[tc.motion]

			var movedX, scaled bool
			for tick := 1; tick < m.ticks; tick++ {
				got := m.apply(float64(tick)/float64(m.ticks), base)
				if got.X != base.X {
					movedX = true
				}
				if got.Scale != base.Scale {
					scaled = true
				}
				if got.Y != base.Y || got.Rotation != base.Rotation {
					t.Fatalf("tick %d changed Y or Rotation: %+v", tick, got)
				}
			}

			if movedX != tc.wantMovesX {
				t.Errorf("moves X = %v, want %v", movedX, tc.wantMovesX)
			}
			if scaled != tc.wantScales {
				t.Errorf("scales = %v, want %v", scaled, tc.wantScales)
			}
		})
	}
}

// 動きの大きさはキャラクターの大きさに比例する。
// 拡大縮小しても見た目の比率が変わらないことを保証する。
func TestMotionDisplacementScalesWithCharacterSize(t *testing.T) {
	t.Parallel()

	m := motions[Attack]
	progress := 0.5

	small := sprite.Transform{X: 0, Y: 0, Scale: 50}
	large := sprite.Transform{X: 0, Y: 0, Scale: 150}

	smallOffset := m.apply(progress, small).X - small.X
	largeOffset := m.apply(progress, large).X - large.X

	if got, want := largeOffset/smallOffset, 3.0; math.Abs(got-want) > 1e-9 {
		t.Errorf("displacement ratio = %v, want %v", got, want)
	}
}

// 揺れは時間とともに減衰し、終わりに向けて収まる。
func TestHitShakeDecays(t *testing.T) {
	t.Parallel()

	m := motions[Hit]
	base := baseTransform()

	peak := func(from, to float64) float64 {
		var max float64
		for p := from; p < to; p += 0.001 {
			if offset := math.Abs(m.apply(p, base).X - base.X); offset > max {
				max = offset
			}
		}
		return max
	}

	if early, late := peak(0, 0.5), peak(0.5, 1); late >= early {
		t.Errorf("late peak %v is not smaller than early peak %v", late, early)
	}
}

// 未定義のMotionは無視する。再生も順番待ちも増やさない。
func TestPlayIgnoresUnknownMotion(t *testing.T) {
	t.Parallel()

	var p Player
	p.Play(Motion(99))

	if _, ok := p.Playing(); ok {
		t.Error("an unknown motion started playing")
	}
	if got := p.Queued(); got != 0 {
		t.Errorf("queued = %d, want 0", got)
	}
}
