package battleui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
)

// 技の使用はそのまま1つのcueになる。Struggleも同じcueで表す。
func TestMoveUsedBecomesMoveUsedCue(t *testing.T) {
	t.Parallel()

	before := battleInput(t).State
	cues := Cues(before, []battle.Event{
		battle.MoveUsed{Side: battle.Player1, Slot: 1, Move: roster.MoveOverdrive},
		battle.MoveUsed{Side: battle.Player2, Slot: battle.NoMoveSlot, Move: battle.MoveStruggle},
	})

	used, ok := cues[0].(MoveUsedCue)
	if !ok {
		t.Fatalf("cues[0] = %T", cues[0])
	}
	if used.Move != roster.MoveOverdrive || used.Slot != 1 {
		t.Errorf("MoveUsedCue = %+v", used)
	}
	if used.Actor.Side != battle.Player1 || used.Actor.Index != 0 || used.Actor.Species == "" {
		t.Errorf("Actor = %+v", used.Actor)
	}

	struggle, ok := cues[1].(MoveUsedCue)
	if !ok {
		t.Fatalf("cues[1] = %T", cues[1])
	}
	if struggle.Slot != battle.NoMoveSlot || struggle.Move != battle.MoveStruggle {
		t.Errorf("Struggleのcue = %+v", struggle)
	}
}

// ダメージのcueはその時点のHPを持ち、最終値へ飛ばさない。
func TestDamageCuesKeepEachStep(t *testing.T) {
	t.Parallel()

	before := battleInput(t).State
	maxHP := before.Players[battle.Player2].Team[0].Stats.HP

	cues := Cues(before, []battle.Event{
		battle.Damage{Side: battle.Player2, Amount: 30, RemainingHP: maxHP - 30},
		battle.Damage{Side: battle.Player2, Amount: 25, RemainingHP: maxHP - 55},
	})

	if len(cues) != 2 {
		t.Fatalf("cueが %d 件（2件のはず）", len(cues))
	}
	for i, wantHP := range []int{maxHP - 30, maxHP - 55} {
		damage, ok := cues[i].(DamageCue)
		if !ok {
			t.Fatalf("cues[%d] = %T", i, cues[i])
		}
		if damage.HP != wantHP {
			t.Errorf("cues[%d].HP = %d, want %d", i, damage.HP, wantHP)
		}
		if damage.MaxHP != maxHP {
			t.Errorf("cues[%d].MaxHP = %d, want %d", i, damage.MaxHP, maxHP)
		}
		if damage.Target.Side != battle.Player2 {
			t.Errorf("cues[%d].Target = %+v", i, damage.Target)
		}
	}
}

// 交代を挟むと、そのあとのcueは出てきた個体を指す。
func TestCueTargetsFollowSwitches(t *testing.T) {
	t.Parallel()

	before := battleInput(t).State
	incoming := before.Players[battle.Player1].Team[2]

	cues := Cues(before, []battle.Event{
		battle.Switched{Side: battle.Player1, From: 0, To: 2},
		battle.Damage{Side: battle.Player1, Amount: 10, RemainingHP: incoming.Stats.HP - 10},
	})

	out, ok := cues[0].(SwitchOutCue)
	if !ok || out.Target.Index != 0 {
		t.Fatalf("cues[0] = %T %+v（下がったのは0）", cues[0], cues[0])
	}
	in, ok := cues[1].(SwitchInCue)
	if !ok || in.Target.Index != 2 || in.Target.Species != incoming.Species {
		t.Fatalf("cues[1] = %T %+v（出たのは2）", cues[1], cues[1])
	}

	damage, ok := cues[2].(DamageCue)
	if !ok {
		t.Fatalf("cues[2] = %T", cues[2])
	}
	if damage.Target.Index != 2 || damage.Target.Species != incoming.Species {
		t.Errorf("交代後のDamage対象 = %+v, want index 2 / %s", damage.Target, incoming.Species)
	}
	if damage.MaxHP != incoming.Stats.HP {
		t.Errorf("MaxHP = %d, want %d（出てきた個体の値）", damage.MaxHP, incoming.Stats.HP)
	}
}

// 急所は直後のダメージへ畳み、単独のcueにはしない。
func TestCriticalHitFoldsIntoTheNextDamage(t *testing.T) {
	t.Parallel()

	before := battleInput(t).State
	cues := Cues(before, []battle.Event{
		battle.CriticalHit{Side: battle.Player2},
		battle.Damage{Side: battle.Player2, Amount: 60, RemainingHP: 100},
	})

	if len(cues) != 1 {
		t.Fatalf("cueが %d 件（畳んで1件のはず）: %s", len(cues), kindsOf(cues))
	}
	damage, ok := cues[0].(DamageCue)
	if !ok || !damage.Critical {
		t.Errorf("cues[0] = %T %+v", cues[0], cues[0])
	}
}

// 急所のflagは後続の反動や継続ダメージへ漏らさない。
func TestCriticalDoesNotLeakToLaterDamage(t *testing.T) {
	t.Parallel()

	before := battleInput(t).State
	cues := Cues(before, []battle.Event{
		battle.CriticalHit{Side: battle.Player2},
		battle.Damage{Side: battle.Player2, Amount: 60, RemainingHP: 100}, // 急所
		battle.Damage{Side: battle.Player1, Amount: 30, RemainingHP: 120}, // 反動
		battle.Damage{Side: battle.Player2, Amount: 10, RemainingHP: 90},  // 継続ダメージ
	})

	if len(cues) != 3 {
		t.Fatalf("cueが %d 件（3件のはず）: %s", len(cues), kindsOf(cues))
	}
	for i, want := range []bool{true, false, false} {
		damage, ok := cues[i].(DamageCue)
		if !ok {
			t.Fatalf("cues[%d] = %T", i, cues[i])
		}
		if damage.Critical != want {
			t.Errorf("cues[%d].Critical = %v, want %v", i, damage.Critical, want)
		}
	}
}

// 直後がそのダメージでない急所は、cueを増やさずに捨てる。
func TestOrphanCriticalHitIsDropped(t *testing.T) {
	t.Parallel()

	before := battleInput(t).State
	cues := Cues(before, []battle.Event{
		battle.CriticalHit{Side: battle.Player2},
		battle.MoveFailed{Side: battle.Player1},
		battle.Damage{Side: battle.Player2, Amount: 10, RemainingHP: 90},
	})

	if len(cues) != 2 {
		t.Fatalf("cueが %d 件（2件のはず）: %s", len(cues), kindsOf(cues))
	}
	damage, ok := cues[1].(DamageCue)
	if !ok {
		t.Fatalf("cues[1] = %T", cues[1])
	}
	if damage.Critical {
		t.Error("離れたダメージへ急所が付いた")
	}
}

// 外れた・効果がない・行動できないを別のcueとして区別する。
func TestMissUnaffectedAndBlockedAreDistinct(t *testing.T) {
	t.Parallel()

	before := battleInput(t).State
	cues := Cues(before, []battle.Event{
		battle.MoveMissed{Side: battle.Player1},
		battle.Unaffected{Side: battle.Player1},
		battle.MoveFailed{Side: battle.Player1},
		battle.ActionBlocked{Side: battle.Player2, Reason: battle.BlockedByParalysis},
		battle.Recharge{Side: battle.Player2},
	})

	miss, ok := cues[0].(MissCue)
	if !ok {
		t.Fatalf("cues[0] = %T", cues[0])
	}
	if miss.Actor.Side != battle.Player1 || miss.Target.Side != battle.Player2 {
		t.Errorf("MissCue = %+v", miss)
	}

	unaffected, ok := cues[1].(UnaffectedCue)
	if !ok {
		t.Fatalf("cues[1] = %T", cues[1])
	}
	// Eventが持つのは使用者のSideだけ。効果がないのは受け手なので、そちらも解決する。
	if unaffected.Actor.Side != battle.Player1 || unaffected.Target.Side != battle.Player2 {
		t.Errorf("UnaffectedCue = %+v", unaffected)
	}

	if _, ok := cues[2].(FailedCue); !ok {
		t.Fatalf("cues[2] = %T", cues[2])
	}

	blocked, ok := cues[3].(BlockedCue)
	if !ok || blocked.Reason != BlockParalysis || blocked.Target.Side != battle.Player2 {
		t.Fatalf("cues[3] = %T %+v", cues[3], cues[3])
	}

	recharge, ok := cues[4].(BlockedCue)
	if !ok || recharge.Reason != BlockRecharge {
		t.Fatalf("cues[4] = %T %+v", cues[4], cues[4])
	}
}

// 状態異常と能力変化もcueになる。
func TestStatusAndStatStageCues(t *testing.T) {
	t.Parallel()

	before := battleInput(t).State
	cues := Cues(before, []battle.Event{
		battle.StatusApplied{Side: battle.Player2, Status: battle.Paralysis},
		battle.StatusRecovered{Side: battle.Player2, Status: battle.Paralysis},
		battle.StatStageChanged{Side: battle.Player1, Stat: battle.StatSpeed, Delta: 2},
		battle.MultiHit{Side: battle.Player1, Hits: 3},
		battle.Healed{Side: battle.Player1, Amount: 50, RemainingHP: 150},
	})

	applied, ok := cues[0].(StatusCue)
	if !ok || !applied.Applied || applied.Status != battle.Paralysis {
		t.Fatalf("cues[0] = %T %+v", cues[0], cues[0])
	}
	recovered, ok := cues[1].(StatusCue)
	if !ok || recovered.Applied {
		t.Fatalf("cues[1] = %T %+v", cues[1], cues[1])
	}
	if stage, ok := cues[2].(StatStageCue); !ok || stage.Delta != 2 || stage.Stat != battle.StatSpeed {
		t.Fatalf("cues[2] = %T %+v", cues[2], cues[2])
	}
	if multi, ok := cues[3].(MultiHitCue); !ok || multi.Hits != 3 {
		t.Fatalf("cues[3] = %T %+v", cues[3], cues[3])
	}
	heal, ok := cues[4].(HealCue)
	if !ok || heal.HP != 150 || heal.MaxHP == 0 {
		t.Fatalf("cues[4] = %T %+v", cues[4], cues[4])
	}
}

// 戦闘不能はfaintのcueになる。
func TestFaintedBecomesFaintCue(t *testing.T) {
	t.Parallel()

	before := battleInput(t).State
	cues := Cues(before, []battle.Event{
		battle.Damage{Side: battle.Player2, Amount: 200, RemainingHP: 0},
		battle.Fainted{Side: battle.Player2, Index: 0},
	})

	faint, ok := cues[1].(FaintCue)
	if !ok {
		t.Fatalf("cues[1] = %T", cues[1])
	}
	if faint.Target.Side != battle.Player2 || faint.Target.Index != 0 || faint.Target.Species == "" {
		t.Errorf("FaintCue = %+v", faint)
	}
}

// ダメージを伴わない戦闘不能もfaintのcueで表す。
//
// 自爆は外れても使用者が倒れる。UIは技の効果から推測せず、
// このcueだけでHPを0にしてひんしへ切り替えられる。
func TestFaintCueWithoutDamage(t *testing.T) {
	t.Parallel()

	before := battleInput(t).State
	cues := Cues(before, []battle.Event{
		battle.MoveUsed{Side: battle.Player1, Slot: 3, Move: roster.MoveBurst},
		battle.MoveMissed{Side: battle.Player1},
		battle.Fainted{Side: battle.Player1, Index: 0},
	})

	if len(cues) != 3 {
		t.Fatalf("cueが %d 件（3件のはず）: %s", len(cues), kindsOf(cues))
	}
	for _, cue := range cues {
		if _, ok := cue.(DamageCue); ok {
			t.Fatal("ダメージのcueが出ている")
		}
	}
	faint, ok := cues[2].(FaintCue)
	if !ok {
		t.Fatalf("cues[2] = %T", cues[2])
	}
	if faint.Target.Side != battle.Player1 || faint.Target.Index != 0 {
		t.Errorf("FaintCue = %+v（自爆した本人のはず）", faint)
	}
}

// 戦闘不能はEventが持つindexを指す。交代でactiveが動いたあとでも取り違えない。
func TestFaintCueHonoursTheEventIndex(t *testing.T) {
	t.Parallel()

	before := battleInput(t).State
	cues := Cues(before, []battle.Event{
		battle.Switched{Side: battle.Player1, From: 0, To: 2},
		battle.Fainted{Side: battle.Player1, Index: 0},
	})

	faint, ok := cues[len(cues)-1].(FaintCue)
	if !ok {
		t.Fatalf("最後のcue = %T", cues[len(cues)-1])
	}
	if faint.Target.Index != 0 {
		t.Errorf("FaintCueの対象 = %d, want 0（activeは2へ動いている）", faint.Target.Index)
	}
	if want := before.Players[battle.Player1].Team[0].Species; faint.Target.Species != want {
		t.Errorf("FaintCueのSpecies = %q, want %q", faint.Target.Species, want)
	}
}

// Event列の順序をそのまま保つ。
func TestCuesKeepEventOrder(t *testing.T) {
	t.Parallel()

	before := battleInput(t).State
	cues := Cues(before, []battle.Event{
		battle.MoveUsed{Side: battle.Player1, Slot: 0, Move: roster.MoveSlam},
		battle.Damage{Side: battle.Player2, Amount: 40, RemainingHP: 120},
		battle.StatusApplied{Side: battle.Player2, Status: battle.Paralysis},
		battle.MoveUsed{Side: battle.Player2, Slot: 0, Move: roster.MoveSpores},
		battle.MoveMissed{Side: battle.Player2},
	})

	want := "MoveUsedCue DamageCue StatusCue MoveUsedCue MissCue"
	if got := kindsOf(cues); got != want {
		t.Errorf("cueの並び = %q, want %q", got, want)
	}
}

// 決着そのもののcueは返さない。結果はSnapshotのResultで見る。
func TestCuesNeverEmitOutcome(t *testing.T) {
	t.Parallel()

	before := battleInput(t).State
	cues := Cues(before, []battle.Event{
		battle.Damage{Side: battle.Player2, Amount: 200, RemainingHP: 0},
		battle.Fainted{Side: battle.Player2, Index: 0},
		battle.Damage{Side: battle.Player1, Amount: 200, RemainingHP: 0},
		battle.Fainted{Side: battle.Player1, Index: 0},
	})

	want := "DamageCue FaintCue DamageCue FaintCue"
	if got := kindsOf(cues); got != want {
		t.Errorf("cueの並び = %q, want %q（結果のcueは出さない）", got, want)
	}
}

// 空のEvent列からはcueが出ない。
func TestCuesOfNothing(t *testing.T) {
	t.Parallel()

	if cues := Cues(battleInput(t).State, nil); len(cues) != 0 {
		t.Errorf("cueが %d 件出た", len(cues))
	}
}

// kindsOf はcueの型名を並べた文字列を返す。順序の比較に使う。
func kindsOf(cues []Cue) string {
	names := make([]string, 0, len(cues))
	for _, cue := range cues {
		name := fmt.Sprintf("%T", cue)
		names = append(names, name[strings.LastIndex(name, ".")+1:])
	}
	return strings.Join(names, " ")
}
