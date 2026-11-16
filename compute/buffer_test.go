package compute

import (
	"testing"
	"voxelgame/voxel"
)

// TestHostVisibleBufferRoundtrip: create → write → read без шейдера.
// Проверяет цепочку buffer+memory+map целиком.
func TestHostVisibleBufferRoundtrip(t *testing.T) {
	ctx, err := Init()
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer ctx.Destroy()

	const size = voxel.VoxelCount * 4 // чанк как uint32, как в будущем Этапе 2.2
	buf, err := CreateHostVisibleBuffer(ctx, size)
	if err != nil {
		t.Fatalf("CreateHostVisibleBuffer: %v", err)
	}
	defer buf.Destroy(ctx)

	// Пишем детерминированный паттерн.
	want := make([]byte, size)
	for i := range want {
		want[i] = byte(i*31 + 7)
	}
	if err := buf.Write(want); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got := make([]byte, size)
	if err := buf.Read(got); err != nil {
		t.Fatalf("Read: %v", err)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("байт %d: got %d, хотим %d", i, got[i], want[i])
		}
	}
}
