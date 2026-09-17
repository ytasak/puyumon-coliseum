package battle

import (
	"errors"
	"testing"
)

// 技を選ぶ行動は技枠の範囲だけを取る。
func TestMoveActionValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		slot  int
		valid bool
	}{
		{"first slot", 0, true},
		{"last slot", MoveSlots - 1, true},
		{"negative", -1, false},
		{"past last", MoveSlots, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := MoveAction{Slot: tt.slot}.Validate()
			if tt.valid && err != nil {
				t.Fatalf("Validate() error = %v, want nil", err)
			}
			if !tt.valid && !errors.Is(err, ErrInvalidState) {
				t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidState)
			}
		})
	}
}

// 交代する行動はチームの範囲だけを取る。
func TestSwitchActionValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target int
		valid  bool
	}{
		{"first", 0, true},
		{"last", TeamSize - 1, true},
		{"negative", -1, false},
		{"past last", TeamSize, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := SwitchAction{Target: tt.target}.Validate()
			if tt.valid && err != nil {
				t.Fatalf("Validate() error = %v, want nil", err)
			}
			if !tt.valid && !errors.Is(err, ErrInvalidState) {
				t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidState)
			}
		})
	}
}

// Actionはtype switchで漏れなく振り分けられる。
func TestActionsAreDistinguishable(t *testing.T) {
	t.Parallel()

	actions := []Action{
		MoveAction{Slot: 1},
		SwitchAction{Target: 2},
	}

	var moves, switches int
	for _, action := range actions {
		switch a := action.(type) {
		case MoveAction:
			moves++
			if a.Slot != 1 {
				t.Errorf("MoveAction.Slot = %d, want 1", a.Slot)
			}
		case SwitchAction:
			switches++
			if a.Target != 2 {
				t.Errorf("SwitchAction.Target = %d, want 2", a.Target)
			}
		default:
			t.Fatalf("unhandled action %T", action)
		}
	}

	if moves != 1 || switches != 1 {
		t.Errorf("moves = %d, switches = %d, want 1 and 1", moves, switches)
	}
}
