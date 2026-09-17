package battle

import "testing"

// 仕様書が挙げるEventがすべて定義されていて、type switchで振り分けられる。
// 描画側はこのswitchだけでturnの結果を解釈できる。
func TestEventsAreDistinguishable(t *testing.T) {
	t.Parallel()

	events := []Event{
		MoveUsed{Side: Player1, Slot: 0, Move: "move-a"},
		Switched{Side: Player1, From: 0, To: 2},
		CriticalHit{Side: Player2},
		Damage{Side: Player2, Amount: 40, RemainingHP: 160},
		StatusApplied{Side: Player2, Status: Paralysis},
		StatusRecovered{Side: Player2, Status: Sleep},
		Fainted{Side: Player2, Index: 1},
		Recharge{Side: Player1},
	}

	seen := make(map[string]bool, len(events))
	for _, event := range events {
		switch e := event.(type) {
		case MoveUsed:
			seen["move_used"] = true
			if e.Move != "move-a" {
				t.Errorf("MoveUsed.Move = %q, want %q", e.Move, "move-a")
			}
		case Switched:
			seen["switched"] = true
			if e.From != 0 || e.To != 2 {
				t.Errorf("Switched = %+v, want from 0 to 2", e)
			}
		case CriticalHit:
			seen["critical_hit"] = true
		case Damage:
			seen["damage"] = true
			if e.Amount != 40 || e.RemainingHP != 160 {
				t.Errorf("Damage = %+v, want 40 damage and 160 left", e)
			}
		case StatusApplied:
			seen["status_applied"] = true
			if e.Status != Paralysis {
				t.Errorf("StatusApplied.Status = %v, want %v", e.Status, Paralysis)
			}
		case StatusRecovered:
			seen["status_recovered"] = true
		case Fainted:
			seen["fainted"] = true
			if e.Index != 1 {
				t.Errorf("Fainted.Index = %d, want 1", e.Index)
			}
		case Recharge:
			seen["recharge"] = true
		default:
			t.Fatalf("unhandled event %T", event)
		}
	}

	for _, name := range []string{
		"move_used", "switched", "critical_hit", "damage",
		"status_applied", "status_recovered", "fainted", "recharge",
	} {
		if !seen[name] {
			t.Errorf("event %s was not covered", name)
		}
	}
}
