package game

import (
	"testing"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
)

// 縦長のときだけ横持ちを促す。
func TestNeedsRotation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		width, height int
		want          bool
	}{
		{name: "iPhone 13 mini portrait", width: 375, height: 610, want: true},
		{name: "iPhone 13 mini landscape", width: 812, height: 375, want: false},
		{name: "default desktop window", width: 1280, height: 720, want: false},
		{name: "square", width: 480, height: 480, want: false},
		{name: "barely portrait", width: 359, height: 360, want: true},
		{name: "barely landscape", width: 361, height: 360, want: false},
		{name: "size unknown", width: 0, height: 0, want: false},
		{name: "height unknown", width: 640, height: 0, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needsRotation(tt.width, tt.height); got != tt.want {
				t.Errorf("needsRotation(%d, %d) = %v, want %v", tt.width, tt.height, got, tt.want)
			}
		})
	}
}

// Layoutは外側のサイズを控えるが、返す論理解像度は変えない。
func TestLayoutRecordsTheOutsideSizeWithoutChangingTheResolution(t *testing.T) {
	t.Parallel()

	g := newGame(t)
	w, h := g.Layout(375, 610)

	if w != LogicalWidth || h != LogicalHeight {
		t.Errorf("Layout(375, 610) = (%d, %d), want (%d, %d)", w, h, LogicalWidth, LogicalHeight)
	}
	if g.outsideWidth != 375 || g.outsideHeight != 610 {
		t.Errorf("recorded outside size = (%d, %d), want (375, 610)", g.outsideWidth, g.outsideHeight)
	}
	if !needsRotation(g.outsideWidth, g.outsideHeight) {
		t.Error("a portrait viewport should ask for rotation")
	}
}

// 縦長でも横長でもDrawはpanicしない。
// ヘッドレスではpixelを読めないため、どちらが描かれたかは検証できない。
func TestDrawDoesNotPanicInEitherOrientation(t *testing.T) {
	t.Parallel()

	g := newGame(t)
	screen := ebiten.NewImage(LogicalWidth, LogicalHeight)

	for _, size := range []struct{ w, h int }{
		{w: 375, h: 610}, // 縦長。回転を促す画面
		{w: 812, h: 375}, // 横長。ゲーム画面
		{w: 0, h: 0},     // サイズ不明
	} {
		g.Layout(size.w, size.h)
		g.Draw(screen)
	}
}

// 促す文面は組み込みのASCIIフォントで描くため、非ASCII文字を含められない。
func TestRotatePromptIsASCIIOnly(t *testing.T) {
	t.Parallel()

	for _, line := range rotatePromptLines {
		for _, r := range line {
			if r > unicode.MaxASCII {
				t.Errorf("rotate prompt line %q contains non-ASCII rune %q", line, r)
			}
		}
	}
}

// 促す画面の図と文面が論理画面へ収まる。
func TestRotatePromptFitsOnScreen(t *testing.T) {
	t.Parallel()

	if right := landscapeIconLeft + landscapeIconWidth; right > LogicalWidth {
		t.Errorf("landscape icon reaches x=%d, beyond the screen width %d", right, LogicalWidth)
	}
	if portraitIconLeft < 0 {
		t.Errorf("portrait icon starts at x=%d, before the screen", portraitIconLeft)
	}
	if bottom := rotatePromptTextTop + len(rotatePromptLines)*rotatePromptLineGap; bottom > LogicalHeight {
		t.Errorf("prompt text reaches y=%d, beyond the screen height %d", bottom, LogicalHeight)
	}
}
