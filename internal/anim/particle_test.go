package anim

import (
	"math"
	"testing"
)

// particleは生成でき、寿命が尽きたら自動で消える。
func TestParticlesAreSpawnedAndExpire(t *testing.T) {
	t.Parallel()

	var ps Particles
	if got := ps.Len(); got != 0 {
		t.Fatalf("Len() on a new Particles = %d, want 0", got)
	}

	ps.Spawn("⚡", 100, 200, 32)
	ps.Spawn("💥", 150, 200, 32)
	if got, want := ps.Len(), 2; got != want {
		t.Fatalf("Len() after spawning = %d, want %d", got, want)
	}

	for i := 0; i < particleTicks-1; i++ {
		ps.Update()
		if ps.Len() != 2 {
			t.Fatalf("tick %d: Len() = %d, want 2; particles expired too early", i, ps.Len())
		}
	}

	ps.Update()
	if got := ps.Len(); got != 0 {
		t.Errorf("Len() after the lifetime elapsed = %d, want 0", got)
	}
}

// particleは生成位置から上へ移動し、終盤で縮んで消える。
func TestParticleRisesAndShrinks(t *testing.T) {
	t.Parallel()

	const (
		spawnX = 100.0
		spawnY = 200.0
		size   = 32.0
	)

	var ps Particles
	ps.Spawn("❄️", spawnX, spawnY, size)

	_, first := ps.At(0)
	if first.X != spawnX || first.Y != spawnY {
		t.Errorf("spawn position = (%v, %v), want (%v, %v)", first.X, first.Y, spawnX, spawnY)
	}
	if first.Scale != size {
		t.Errorf("spawn size = %v, want %v", first.Scale, size)
	}

	previousY, previousScale := first.Y, first.Scale
	for i := 1; i < particleTicks; i++ {
		ps.Update()
		_, got := ps.At(0)

		if got.X != spawnX {
			t.Fatalf("tick %d: X = %v, want it to stay at %v", i, got.X, spawnX)
		}
		if got.Y >= previousY {
			t.Fatalf("tick %d: Y = %v, want it to keep rising above %v", i, got.Y, previousY)
		}
		if got.Scale > previousScale {
			t.Fatalf("tick %d: scale = %v, want it never to grow beyond %v", i, got.Scale, previousScale)
		}
		previousY, previousScale = got.Y, got.Scale
	}

	if previousScale >= size*0.2 {
		t.Errorf("scale just before expiring = %v, want it nearly gone", previousScale)
	}
}

// 縮みは終盤だけで、それまでは元の大きさを保つ。
func TestParticleKeepsItsSizeUntilTheEnd(t *testing.T) {
	t.Parallel()

	if got := particleShrink(0); got != 1 {
		t.Errorf("particleShrink(0) = %v, want 1", got)
	}
	if got := particleShrink(particleShrinkFrom - 0.01); got != 1 {
		t.Errorf("particleShrink just before the shrink point = %v, want 1", got)
	}
	if got := particleShrink(1); math.Abs(got) > 1e-9 {
		t.Errorf("particleShrink(1) = %v, want 0", got)
	}
}

// particleは描画にそのまま渡せるCharacterを持つ。
func TestParticleCarriesItsEmoji(t *testing.T) {
	t.Parallel()

	var ps Particles
	ps.Spawn("💤", 0, 0, 16)

	character, _ := ps.At(0)
	if got, want := len(character.Parts), 1; got != want {
		t.Fatalf("particle character has %d parts, want %d", got, want)
	}
	if got, want := character.Parts[0].Emoji, "💤"; got != want {
		t.Errorf("particle emoji = %q, want %q", got, want)
	}
}

// 定常状態のUpdateでは確保が起きない。
// 毎tick呼ばれるので、ここでの確保はそのままallocation hotspotになる。
func TestUpdateDoesNotAllocate(t *testing.T) {
	var ps Particles
	for i := 0; i < 8; i++ {
		ps.items = append(ps.items, particle{size: 16, lifetime: math.MaxInt32})
	}

	if got := testing.AllocsPerRun(100, ps.Update); got != 0 {
		t.Errorf("Update allocated %v times per run, want 0", got)
	}
}

// particleを繰り返し生成・破棄しても、内部のbufferは使い回される。
func TestSpawningRepeatedlyReusesTheBuffer(t *testing.T) {
	t.Parallel()

	const perWave = 4

	var ps Particles
	for wave := 0; wave < 20; wave++ {
		for i := 0; i < perWave; i++ {
			ps.Spawn("⚡", 0, 0, 16)
		}
		for i := 0; i < particleTicks; i++ {
			ps.Update()
		}
		if ps.Len() != 0 {
			t.Fatalf("wave %d: %d particles survived their lifetime", wave, ps.Len())
		}
	}

	if got := cap(ps.items); got > perWave*2 {
		t.Errorf("buffer capacity grew to %d after 20 waves of %d; it should be reused", got, perWave)
	}
}
