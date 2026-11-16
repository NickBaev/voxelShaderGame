package voxel

import (
	"math"
	"testing"
)

// countSolid считает твёрдые воксели напрямую, не завися от Stats.
func countSolid(c *Chunk) int {
	n := 0
	for z := 0; z < ChunkSize; z++ {
		for y := 0; y < ChunkSize; y++ {
			for x := 0; x < ChunkSize; x++ {
				if c.Get(x, y, z) != Air {
					n++
				}
			}
		}
	}
	return n
}

func TestEmpty(t *testing.T) {
	if n := countSolid(NewEmpty()); n != 0 {
		t.Errorf("NewEmpty: твёрдых = %d, хотим 0", n)
	}
}

func TestSolid(t *testing.T) {
	c := NewSolid(3)
	if n := countSolid(c); n != VoxelCount {
		t.Errorf("NewSolid: твёрдых = %d, хотим %d", n, VoxelCount)
	}
	if c.Get(0, 0, 0) != 3 || c.Get(31, 31, 31) != 3 {
		t.Error("NewSolid: материал не везде 3")
	}
}

func TestSingleVoxel(t *testing.T) {
	c := NewSingleVoxel(16, 16, 16, 5)
	if n := countSolid(c); n != 1 {
		t.Fatalf("твёрдых = %d, хотим 1", n)
	}
	for z := 0; z < ChunkSize; z++ {
		for y := 0; y < ChunkSize; y++ {
			for x := 0; x < ChunkSize; x++ {
				want := uint32(Air)
				if x == 16 && y == 16 && z == 16 {
					want = 5
				}
				if got := c.Get(x, y, z); got != want {
					t.Fatalf("Get(%d,%d,%d) = %d, хотим %d", x, y, z, got, want)
				}
			}
		}
	}

	// Воксель в углу — тоже ровно один твёрдый.
	if n := countSolid(NewSingleVoxel(0, 0, 0, 5)); n != 1 {
		t.Errorf("угол: твёрдых = %d, хотим 1", n)
	}
}

func TestTwoAdjacent(t *testing.T) {
	c := NewTwoAdjacent(9)
	if n := countSolid(c); n != 2 {
		t.Fatalf("твёрдых = %d, хотим 2", n)
	}
	// Находим воксели и проверяем, что они соседи по одной оси.
	var pts [][3]int
	for z := 0; z < ChunkSize && len(pts) < 2; z++ {
		for y := 0; y < ChunkSize && len(pts) < 2; y++ {
			for x := 0; x < ChunkSize && len(pts) < 2; x++ {
				if c.Get(x, y, z) != Air {
					pts = append(pts, [3]int{x, y, z})
				}
			}
		}
	}
	if len(pts) != 2 {
		t.Fatalf("нашли %d точек вместо 2", len(pts))
	}
	d := [3]int{pts[1][0] - pts[0][0], pts[1][1] - pts[0][1], pts[1][2] - pts[0][2]}
	manhattan := d[0] + d[1] + d[2]
	if manhattan != 1 && manhattan != -1 {
		t.Errorf("воксели %v и %v не соседи по оси (разность %v)", pts[0], pts[1], d)
	}
}

func TestPlaneY(t *testing.T) {
	c := NewPlaneY(16, 4)
	if n := countSolid(c); n != ChunkSize*ChunkSize {
		t.Fatalf("твёрдых = %d, хотим %d", n, ChunkSize*ChunkSize)
	}
	for z := 0; z < ChunkSize; z++ {
		for x := 0; x < ChunkSize; x++ {
			if c.Get(x, 15, z) != Air || c.Get(x, 17, z) != Air {
				t.Error("воксели вне плоскости y=16 должны быть воздухом")
			}
			if c.Get(x, 16, z) != 4 {
				t.Errorf("Get(%d,16,%d) != 4", x, z)
			}
		}
	}
	// Плоскость на границе чанка — тоже легальна.
	if n := countSolid(NewPlaneY(0, 4)); n != ChunkSize*ChunkSize {
		t.Errorf("плоскость y=0: твёрдых = %d, хотим %d", n, ChunkSize*ChunkSize)
	}
}

func TestCheckerboard(t *testing.T) {
	const m = uint32(7)
	c := NewCheckerboard(m, Air)

	// Ровно половина клеток — твёрдые.
	if n := countSolid(c); n != VoxelCount/2 {
		t.Errorf("твёрдых = %d, хотим %d", n, VoxelCount/2)
	}

	// Ни один твёрдый воксель не имеет твёрдого соседа (по всем 6 направлениям).
	dirs := [][3]int{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}}
	for z := 0; z < ChunkSize; z++ {
		for y := 0; y < ChunkSize; y++ {
			for x := 0; x < ChunkSize; x++ {
				if c.Get(x, y, z) == Air {
					continue
				}
				for _, d := range dirs {
					if c.Get(x+d[0], y+d[1], z+d[2]) != Air {
						t.Fatalf("твёрдые соседи: (%d,%d,%d) и (%d,%d,%d)",
							x, y, z, x+d[0], y+d[1], z+d[2])
					}
				}
			}
		}
	}
}

func TestNoiseDeterministic(t *testing.T) {
	a := NewNoise(42, 0.5, 1)
	b := NewNoise(42, 0.5, 1)
	for z := 0; z < ChunkSize; z++ {
		for y := 0; y < ChunkSize; y++ {
			for x := 0; x < ChunkSize; x++ {
				if a.Get(x, y, z) != b.Get(x, y, z) {
					t.Fatalf("тот же seed: расхождение в (%d,%d,%d)", x, y, z)
				}
			}
		}
	}

	c := NewNoise(43, 0.5, 1)
	diff := 0
	for z := 0; z < ChunkSize; z++ {
		for y := 0; y < ChunkSize; y++ {
			for x := 0; x < ChunkSize; x++ {
				if a.Get(x, y, z) != c.Get(x, y, z) {
					diff++
				}
			}
		}
	}
	if diff == 0 {
		t.Error("разные seed дали идентичные чанки")
	}
}

func TestNoiseDensity(t *testing.T) {
	// Края: 0 → пусто, 1 → полный чанк.
	if n := countSolid(NewNoise(1, 0.0, 1)); n != 0 {
		t.Errorf("density=0: твёрдых = %d, хотим 0", n)
	}
	if n := countSolid(NewNoise(1, 1.0, 1)); n != VoxelCount {
		t.Errorf("density=1: твёрдых = %d, хотим %d", n, VoxelCount)
	}

	// Sanity check распределения: при 0.5 должно быть около половины.
	n := countSolid(NewNoise(7, 0.5, 1))
	dev := math.Abs(float64(n) - VoxelCount/2)
	if dev > 0.05*VoxelCount {
		t.Errorf("density=0.5: твёрдых = %d, отклонение %.1f%% — подозрительно много", n, 100*dev/VoxelCount)
	}
}
