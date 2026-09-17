package game

import "testing"

// Layoutは外側のサイズに関係なく論理解像度を固定で返す。
// ウィンドウリサイズやbrowser viewportの変化でゲーム側の座標系が変わらないことを保証する。
func TestLayoutReturnsFixedLogicalResolution(t *testing.T) {
	t.Parallel()

	outsideSizes := []struct {
		name          string
		width, height int
	}{
		{"default window", LogicalWidth * DefaultWindowScale, LogicalHeight * DefaultWindowScale},
		{"same as logical", LogicalWidth, LogicalHeight},
		{"smaller", 320, 180},
		{"larger", 2560, 1440},
		{"different aspect ratio", 800, 1200},
		{"zero", 0, 0},
	}

	g := newGame(t)
	for _, s := range outsideSizes {
		t.Run(s.name, func(t *testing.T) {
			gotW, gotH := g.Layout(s.width, s.height)
			if gotW != LogicalWidth || gotH != LogicalHeight {
				t.Errorf("Layout(%d, %d) = (%d, %d), want (%d, %d)",
					s.width, s.height, gotW, gotH, LogicalWidth, LogicalHeight)
			}
		})
	}
}

// Layoutの戻り値はtickが進んでも変化しない。
func TestLayoutIsStableAcrossUpdates(t *testing.T) {
	t.Parallel()

	g := newGame(t)
	wantW, wantH := g.Layout(1280, 720)

	for i := 0; i < 100; i++ {
		if err := g.Update(); err != nil {
			t.Fatalf("Update() returned error: %v", err)
		}
	}

	gotW, gotH := g.Layout(1280, 720)
	if gotW != wantW || gotH != wantH {
		t.Errorf("Layout after updates = (%d, %d), want (%d, %d)", gotW, gotH, wantW, wantH)
	}
}

func TestDefaultWindowSizeIsScaledLogicalResolution(t *testing.T) {
	t.Parallel()

	w, h := DefaultWindowSize()
	if w != LogicalWidth*DefaultWindowScale || h != LogicalHeight*DefaultWindowScale {
		t.Errorf("DefaultWindowSize() = (%d, %d), want (%d, %d)",
			w, h, LogicalWidth*DefaultWindowScale, LogicalHeight*DefaultWindowScale)
	}
}
