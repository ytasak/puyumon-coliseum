package sprite

import (
	"math"
	"testing"
)

const epsilon = 1e-9

func assertPlacement(t *testing.T, got Placement, want Placement) {
	t.Helper()

	if math.Abs(got.CenterX-want.CenterX) > epsilon || math.Abs(got.CenterY-want.CenterY) > epsilon {
		t.Errorf("center = (%v, %v), want (%v, %v)", got.CenterX, got.CenterY, want.CenterX, want.CenterY)
	}
	if math.Abs(got.Size-want.Size) > epsilon {
		t.Errorf("size = %v, want %v", got.Size, want.Size)
	}
	if math.Abs(got.Rotation-want.Rotation) > epsilon {
		t.Errorf("rotation = %v, want %v", got.Rotation, want.Rotation)
	}
}

// 恒等変換ではPartのlocalな定義がそのまま出てくる。
func TestPlaceWithIdentityTransform(t *testing.T) {
	t.Parallel()

	part := Part{Emoji: "🌴", X: 0.25, Y: -0.5, Scale: 0.75, Rotation: 0.3}
	got := part.Place(Transform{Scale: 1})

	assertPlacement(t, got, Placement{CenterX: 0.25, CenterY: -0.5, Size: 0.75, Rotation: 0.3})
}

// Character全体の移動では、全Partが同じだけ平行移動する。
func TestTransformPositionMovesEveryPartTogether(t *testing.T) {
	t.Parallel()

	parts := []Part{
		{Emoji: "🌴", X: 0, Y: 0, Scale: 1},
		{Emoji: "🥺", X: -0.2, Y: -0.3, Scale: 0.3},
	}
	base := Transform{X: 0, Y: 0, Scale: 100}
	moved := Transform{X: 40, Y: -25, Scale: 100}

	for _, p := range parts {
		before, after := p.Place(base), p.Place(moved)
		if dx, dy := after.CenterX-before.CenterX, after.CenterY-before.CenterY; math.Abs(dx-40) > epsilon || math.Abs(dy+25) > epsilon {
			t.Errorf("part %q moved by (%v, %v), want (40, -25)", p.Emoji, dx, dy)
		}
		if math.Abs(after.Size-before.Size) > epsilon {
			t.Errorf("part %q size changed on translation: %v -> %v", p.Emoji, before.Size, after.Size)
		}
	}
}

// Character全体の拡大では、部品の大きさと中心間の距離が同じ倍率で変わる。
// 部品だけが大きくなって位置関係が崩れることがないのを保証する。
func TestTransformScaleAffectsSizeAndOffsetEqually(t *testing.T) {
	t.Parallel()

	body := Part{Emoji: "🌴", X: 0, Y: 0, Scale: 1}
	face := Part{Emoji: "🥺", X: -0.2, Y: -0.3, Scale: 0.3}

	small := Transform{X: 100, Y: 100, Scale: 50}
	large := Transform{X: 100, Y: 100, Scale: 150}

	distance := func(t Transform) float64 {
		b, f := body.Place(t), face.Place(t)
		return math.Hypot(f.CenterX-b.CenterX, f.CenterY-b.CenterY)
	}

	if got, want := distance(large)/distance(small), 3.0; math.Abs(got-want) > epsilon {
		t.Errorf("distance ratio = %v, want %v", got, want)
	}
	if got, want := face.Place(large).Size/face.Place(small).Size, 3.0; math.Abs(got-want) > epsilon {
		t.Errorf("size ratio = %v, want %v", got, want)
	}
}

// Character全体の回転では、部品の位置も回転し、部品自身の角度にも加算される。
func TestTransformRotationRotatesOffsetsAndParts(t *testing.T) {
	t.Parallel()

	// 中心の右隣にある部品を90度（時計回り）回すと、中心の真下へ来る。
	part := Part{Emoji: "🥺", X: 1, Y: 0, Scale: 0.5, Rotation: 0.25}
	got := part.Place(Transform{X: 0, Y: 0, Scale: 1, Rotation: math.Pi / 2})

	assertPlacement(t, got, Placement{CenterX: 0, CenterY: 1, Size: 0.5, Rotation: 0.25 + math.Pi/2})
}

// 回転しても部品どうしの距離は変わらない（一体として回る）。
func TestTransformRotationKeepsPartsRigid(t *testing.T) {
	t.Parallel()

	body := Part{Emoji: "🌴", X: 0, Y: 0, Scale: 1}
	face := Part{Emoji: "🤪", X: 0.24, Y: -0.18, Scale: 0.3}

	distance := func(rotation float64) float64 {
		tr := Transform{X: 320, Y: 180, Scale: 100, Rotation: rotation}
		b, f := body.Place(tr), face.Place(tr)
		return math.Hypot(f.CenterX-b.CenterX, f.CenterY-b.CenterY)
	}

	want := distance(0)
	for _, r := range []float64{0.3, 1, math.Pi, -2.5} {
		if got := distance(r); math.Abs(got-want) > 1e-6 {
			t.Errorf("distance at rotation %v = %v, want %v", r, got, want)
		}
	}
}

// 部品ごとのlocal scale / rotationは、Character全体の変換とは独立に効く。
func TestPartLocalTransformIsIndependent(t *testing.T) {
	t.Parallel()

	tr := Transform{X: 10, Y: 20, Scale: 100, Rotation: 0.5}
	at := Part{Emoji: "🥺", X: 0.1, Y: 0.2, Scale: 1, Rotation: 0}.Place(tr)
	scaled := Part{Emoji: "🥺", X: 0.1, Y: 0.2, Scale: 0.25, Rotation: 0}.Place(tr)
	rotated := Part{Emoji: "🥺", X: 0.1, Y: 0.2, Scale: 1, Rotation: 0.75}.Place(tr)

	// 大きさ・角度だけが変わり、中心位置は動かない。
	if math.Abs(scaled.CenterX-at.CenterX) > epsilon || math.Abs(scaled.CenterY-at.CenterY) > epsilon {
		t.Error("part scale moved the part center")
	}
	if math.Abs(rotated.CenterX-at.CenterX) > epsilon || math.Abs(rotated.CenterY-at.CenterY) > epsilon {
		t.Error("part rotation moved the part center")
	}
	if got, want := scaled.Size/at.Size, 0.25; math.Abs(got-want) > epsilon {
		t.Errorf("part scale ratio = %v, want %v", got, want)
	}
	if got, want := rotated.Rotation-at.Rotation, 0.75; math.Abs(got-want) > epsilon {
		t.Errorf("part rotation delta = %v, want %v", got, want)
	}
}

// NewCharacterはZの昇順へ並べ、同じZでは定義順を保つ。
func TestNewCharacterOrdersPartsByZ(t *testing.T) {
	t.Parallel()

	c := NewCharacter(
		Part{Emoji: "face-a", Z: 1},
		Part{Emoji: "body", Z: 0},
		Part{Emoji: "face-b", Z: 1},
	)

	got := make([]string, len(c.Parts))
	for i, p := range c.Parts {
		got[i] = p.Emoji
	}
	want := []string{"body", "face-a", "face-b"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("draw order = %v, want %v", got, want)
		}
	}
}

// NewCharacterは渡されたsliceを書き換えない。
func TestNewCharacterDoesNotMutateInput(t *testing.T) {
	t.Parallel()

	parts := []Part{{Emoji: "face", Z: 1}, {Emoji: "body", Z: 0}}
	NewCharacter(parts...)

	if parts[0].Emoji != "face" || parts[1].Emoji != "body" {
		t.Errorf("NewCharacter reordered the caller's slice: %v", parts)
	}
}
