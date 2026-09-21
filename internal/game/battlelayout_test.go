package game

import (
	"image"
	"testing"
	"unicode"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/battleui"
	"github.com/ytasak/puyumon-coliseum/internal/singleplayer"
)

// ボタンは画面に収まり、重ならない。
func TestCommandRectsFitOnScreenAndDoNotOverlap(t *testing.T) {
	t.Parallel()

	screen := image.Rect(0, 0, LogicalWidth, LogicalHeight)
	for i, rect := range commandRects {
		if !rect.In(screen) {
			t.Errorf("枠 %d %v が画面からはみ出している", i, rect)
		}
		if rect.Min.Y < commandTop {
			t.Errorf("枠 %d がメッセージ行へかぶっている", i)
		}
		for j := i + 1; j < len(commandRects); j++ {
			if rect.Overlaps(commandRects[j]) {
				t.Errorf("枠 %d と %d が重なっている", i, j)
			}
		}
	}
}

// 盤面・メッセージ・コマンドが上から順に並び、重ならない。
func TestScreenBandsAreOrdered(t *testing.T) {
	t.Parallel()

	if !(fieldBottom < messageTop && messageTop < messageBottom && messageBottom < commandTop) {
		t.Errorf("帯の順序が壊れている: field %d / message %d-%d / command %d",
			fieldBottom, messageTop, messageBottom, commandTop)
	}
	if commandRects[len(commandRects)-1].Max.Y > LogicalHeight {
		t.Error("コマンドが画面の下からはみ出している")
	}
}

// lead選択では3体が別々の枠に出る。
func TestLeadButtonsOfferEveryDealtPokemon(t *testing.T) {
	t.Parallel()

	scene := newSceneOrFatal(t, 1)
	buttons := scene.buttons()

	if len(buttons) != 3 {
		t.Fatalf("ボタンが %d 個（3個のはず）", len(buttons))
	}
	cells := map[int]bool{}
	for _, b := range buttons {
		if cells[b.cell] {
			t.Errorf("枠 %d が重複している", b.cell)
		}
		cells[b.cell] = true
		if b.label == "" {
			t.Error("ラベルが空")
		}
	}
}

// Fightを押すと技の一覧が開き、BACKで戻る。
func TestFightMenuOpensAndCloses(t *testing.T) {
	t.Parallel()

	scene := startedScene(t, 1)

	fight, ok := buttonWithLabel(scene, "FIGHT")
	if !ok {
		t.Fatal("FIGHTが出ていない")
	}
	tapOrFatal(t, scene, center(fight.rect()))
	if scene.menu != menuFight {
		t.Fatalf("menu = %d, want %d", scene.menu, menuFight)
	}

	// 4つの技が並ぶ。
	moves := 0
	for _, b := range scene.buttons() {
		if b.command.kind == commandMove {
			moves++
		}
	}
	if moves != 4 {
		t.Errorf("技が %d 個（4個のはず）", moves)
	}

	back, ok := buttonWithLabel(scene, "BACK")
	if !ok {
		t.Fatal("BACKが出ていない")
	}
	tapOrFatal(t, scene, center(back.rect()))
	if scene.menu != menuRoot {
		t.Error("BACKで戻らない")
	}
}

// 技のボタンを押すとturnが進む。
func TestTappingAMoveAdvancesTheTurn(t *testing.T) {
	t.Parallel()

	scene := startedScene(t, 1)
	openFight(t, scene)

	before, _ := scene.session.State()
	move, ok := enabledMoveButton(scene)
	if !ok {
		t.Fatal("選べる技が無い")
	}
	tapOrFatal(t, scene, center(move.rect()))
	drainCues(t, scene)

	after, _ := scene.session.State()
	if after.Turn == before.Turn && after.Status == battle.Ongoing {
		t.Error("技を押してもturnが進まない")
	}
	// 送ったあとは根へ戻っている。
	if scene.menu != menuRoot {
		t.Error("送信後も技の一覧が開いたまま")
	}
}

// 押せない技は押しても何も起きず、一覧も閉じない。
func TestTappingADisabledMoveDoesNothing(t *testing.T) {
	t.Parallel()

	scene, slot, ok := sceneWithDisabledMove(t)
	if !ok {
		t.Fatal("PPを使い切る局面を作れなかった")
	}
	openFight(t, scene)

	var target button
	for _, b := range scene.buttons() {
		if b.command.kind == commandMove && b.command.index == slot {
			target = b
		}
	}
	if !target.disabled {
		t.Fatalf("slot %d がdisabledになっていない", slot)
	}

	accepted, err := scene.tap(center(target.rect()))
	if accepted || err != nil {
		t.Errorf("押せない技が通った（accepted=%v err=%v）", accepted, err)
	}
	if scene.menu != menuFight {
		t.Error("弾かれたのに技の一覧が閉じた")
	}
}

// どのボタンにも当たらない場所を押しても何も起きない。
func TestTappingOutsideAnyButtonDoesNothing(t *testing.T) {
	t.Parallel()

	scene := startedScene(t, 1)
	if accepted, err := scene.tap(image.Pt(LogicalWidth/2, 8)); accepted || err != nil {
		t.Errorf("盤面のタップが通った（accepted=%v err=%v）", accepted, err)
	}
}

// 再生中はどのボタンも押せない。
func TestTapIsIgnoredWhileCuesPlay(t *testing.T) {
	t.Parallel()

	scene, ok := sceneAtReplacement(t)
	if !ok {
		t.Fatal("replacementへ到達する局面を作れなかった")
	}
	first := scene.buttons()[0]
	tapOrFatal(t, scene, center(first.rect()))
	if !scene.busy() {
		t.Fatal("cueが積まれていない")
	}

	for _, b := range scene.buttons() {
		if accepted, err := scene.tap(center(b.rect())); accepted || err != nil {
			t.Errorf("再生中に %q が通った（accepted=%v err=%v）", b.label, accepted, err)
		}
	}
}

// 再生中はボタンを1つも出さない。
//
// 押せないボタンを見せ続けると、反応しない画面に見える。
func TestNoButtonsAreShownWhileCuesPlay(t *testing.T) {
	t.Parallel()

	scene, ok := sceneAtReplacement(t)
	if !ok {
		t.Fatal("replacementへ到達する局面を作れなかった")
	}
	if len(scene.visibleButtons()) == 0 {
		t.Fatal("再生前からボタンが出ていない")
	}

	tapOrFatal(t, scene, center(scene.buttons()[0].rect()))
	if !scene.busy() {
		t.Fatal("cueが積まれていない")
	}

	if got := scene.visibleButtons(); len(got) != 0 {
		t.Errorf("再生中に %d 個のボタンが出ている", len(got))
	}
	// 選択肢そのものは消さない。消化しきれば戻る。
	if len(scene.buttons()) == 0 {
		t.Error("再生中にviewの選択肢まで消えている")
	}

	drainCues(t, scene)
	if len(scene.visibleButtons()) == 0 {
		t.Error("消化しきってもボタンが戻らない")
	}
}

// 戦闘不能のあとは戻る先が無い。
func TestReplacementHasNoBackButton(t *testing.T) {
	t.Parallel()

	scene, ok := sceneAtReplacement(t)
	if !ok {
		t.Fatal("replacementへ到達する局面を作れなかった")
	}
	for _, b := range scene.buttons() {
		if b.action == actionBack {
			t.Error("replacementでBACKが出ている")
		}
	}
}

// ラベルは組み込みフォントで描けるASCIIだけで作る。
func TestButtonLabelsAreASCIIOnly(t *testing.T) {
	t.Parallel()

	scene := newSceneOrFatal(t, 1)
	labels := collectLabels(t, scene)
	if len(labels) == 0 {
		t.Fatal("ラベルを1つも集められなかった")
	}

	for _, label := range labels {
		for _, r := range label {
			if r > unicode.MaxASCII {
				t.Errorf("%q にASCII外の文字 %q が含まれる", label, r)
			}
		}
		if width := len(label) * debugFontCharWidth; width > cellWidth {
			t.Errorf("%q が枠に収まらない（%d px > %d px）", label, width, cellWidth)
		}
	}
}

// openFight は技の一覧を開く。
func openFight(t *testing.T, s *battleScene) {
	t.Helper()

	fight, ok := buttonWithLabel(s, "FIGHT")
	if !ok {
		t.Fatal("FIGHTが出ていない")
	}
	tapOrFatal(t, s, center(fight.rect()))
}

// tapOrFatal は位置を押す。受理されなければ止める。
func tapOrFatal(t *testing.T, s *battleScene, p image.Point) {
	t.Helper()

	accepted, err := s.tap(p)
	if err != nil {
		t.Fatalf("tap(%v)に失敗: %v", p, err)
	}
	if !accepted {
		t.Fatalf("tap(%v)が受理されなかった", p)
	}
}

// buttonWithLabel はラベルでボタンを探す。
func buttonWithLabel(s *battleScene, label string) (button, bool) {
	for _, b := range s.buttons() {
		if b.label == label {
			return b, true
		}
	}
	return button{}, false
}

// enabledMoveButton は押せる技のボタンを返す。
func enabledMoveButton(s *battleScene) (button, bool) {
	for _, b := range s.buttons() {
		if b.command.kind == commandMove && !b.disabled {
			return b, true
		}
	}
	return button{}, false
}

// center は矩形の中心を返す。
func center(r image.Rectangle) image.Point {
	return image.Pt(r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2)
}

// collectLabels は1試合を通して出てくるラベルを集める。
func collectLabels(t *testing.T, s *battleScene) []string {
	t.Helper()

	var labels []string
	seen := map[string]bool{}
	add := func() {
		for _, b := range s.buttons() {
			if !seen[b.label] {
				seen[b.label] = true
				labels = append(labels, b.label)
			}
		}
	}

	add()
	for step := 0; s.view.Phase != singleplayer.PhaseFinished && step < maxSceneSteps; step++ {
		drainCues(t, s)
		add()
		if s.view.Commands.Kind == battleui.CommandChooseAction {
			s.menu = menuFight
			add()
			s.menu = menuSwitch
			add()
			s.menu = menuRoot
		}

		c, ok := firstChoice(s.view.Commands)
		if !ok {
			if err := s.update(); err != nil {
				t.Fatalf("update()に失敗: %v", err)
			}
			continue
		}
		submitOrFatal(t, s, c)
	}
	drainCues(t, s)
	add()

	return labels
}
