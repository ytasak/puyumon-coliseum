package battle

import "testing"

// Generation Iのタイプはちょうど15種類。Dark / Steel / Fairyは後の世代のもので、
// この世代には存在しない。
func TestAllTypesAreTheFifteenOfGenerationI(t *testing.T) {
	t.Parallel()

	all := AllTypes()
	if len(all) != 15 {
		t.Fatalf("AllTypes() returned %d types, want 15", len(all))
	}

	want := []string{
		"normal", "fighting", "flying", "poison", "ground", "rock", "bug", "ghost",
		"fire", "water", "grass", "electric", "psychic", "ice", "dragon",
	}
	for i, typ := range all {
		if got := typ.String(); got != want[i] {
			t.Errorf("AllTypes()[%d] = %q, want %q", i, got, want[i])
		}
	}
}

// 物理・特殊はGeneration Iでは技ごとではなくタイプで決まる。
func TestTypeCategory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		typ  Type
		want Category
	}{
		{TypeNormal, Physical},
		{TypeFighting, Physical},
		{TypeFlying, Physical},
		{TypePoison, Physical},
		{TypeGround, Physical},
		{TypeRock, Physical},
		{TypeBug, Physical},
		{TypeGhost, Physical},
		{TypeFire, Special},
		{TypeWater, Special},
		{TypeGrass, Special},
		{TypeElectric, Special},
		{TypePsychic, Special},
		{TypeIce, Special},
		{TypeDragon, Special},
	}

	if len(tests) != 15 {
		t.Fatalf("table covers %d types, want 15", len(tests))
	}
	for _, tt := range tests {
		if got := tt.typ.Category(); got != tt.want {
			t.Errorf("%v.Category() = %v, want %v", tt.typ, got, tt.want)
		}
	}
}

// Typingは単タイプと複合タイプのどちらも表せる。
func TestTypingList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		typing Typing
		want   []Type
	}{
		{"single", SingleType(TypeElectric), []Type{TypeElectric}},
		{"dual", DualType(TypeWater, TypePsychic), []Type{TypeWater, TypePsychic}},
		{"secondary left empty", Typing{Primary: TypeFire}, []Type{TypeFire}},
		{"same type twice", DualType(TypeFire, TypeFire), []Type{TypeFire}},
		{"empty", Typing{}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.typing.List()
			if len(got) != len(tt.want) {
				t.Fatalf("List() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("List() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// STABは技のタイプが使用者自身のタイプと一致したときに乗る。
func TestHasSTAB(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		move Type
		user Typing
		want bool
	}{
		{"single type matches", TypeElectric, SingleType(TypeElectric), true},
		{"single type differs", TypeIce, SingleType(TypeElectric), false},
		{"dual type matches primary", TypeWater, DualType(TypeWater, TypePsychic), true},
		{"dual type matches secondary", TypePsychic, DualType(TypeWater, TypePsychic), true},
		{"dual type matches neither", TypeFire, DualType(TypeWater, TypePsychic), false},
		{"no type never matches", NoType, DualType(TypeWater, TypePsychic), false},
		{"empty typing", TypeNormal, Typing{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := HasSTAB(tt.move, tt.user); got != tt.want {
				t.Errorf("HasSTAB(%v, %+v) = %v, want %v", tt.move, tt.user, got, tt.want)
			}
		})
	}
}

// Hasは持っているタイプだけを真とする。
func TestTypingHas(t *testing.T) {
	t.Parallel()

	typing := DualType(TypeGrass, TypePsychic)
	for _, tt := range []struct {
		target Type
		want   bool
	}{
		{TypeGrass, true},
		{TypePsychic, true},
		{TypeFire, false},
		{NoType, false},
	} {
		if got := typing.Has(tt.target); got != tt.want {
			t.Errorf("Has(%v) = %v, want %v", tt.target, got, tt.want)
		}
	}
}
