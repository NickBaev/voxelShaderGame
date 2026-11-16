package voxel

import "testing"

// bruteForceSphere — справочная реализация через полный перебор чанка:
// никакого bounding box, минимальная вероятность ошибки.
func bruteForceSphere(cx, cy, cz, r float64, m uint32) *Chunk {
	c := NewEmpty()
	for z := 0; z < ChunkSize; z++ {
		for y := 0; y < ChunkSize; y++ {
			for x := 0; x < ChunkSize; x++ {
				dx := float64(x) + 0.5 - cx
				dy := float64(y) + 0.5 - cy
				dz := float64(z) + 0.5 - cz
				if dx*dx+dy*dy+dz*dz <= r*r {
					c.Set(x, y, z, m)
				}
			}
		}
	}
	return c
}

func equalChunks(a, b *Chunk) bool {
	for z := 0; z < ChunkSize; z++ {
		for y := 0; y < ChunkSize; y++ {
			for x := 0; x < ChunkSize; x++ {
				if a.Get(x, y, z) != b.Get(x, y, z) {
					return false
				}
			}
		}
	}
	return true
}

func TestSphereMatchesBruteForce(t *testing.T) {
	cases := []struct {
		cx, cy, cz, r float64
	}{
		{16, 16, 16, 10},
		{16, 16, 16, 10.5},   // дробный радиус — граница bbox
		{16, 16, 16, 9.7},    // дробный радиус, другая фаза
		{16.5, 16.5, 16.5, 8}, // смещённый центр на полвокселя
		{10, 22, 7, 6.3},     // неравные центры по осям
		{16, 16, 16, 0.4},    // маленький радиус
		{16, 16, 16, 40},     // больше чанка — должна обрезаться краями
		{2, 2, 2, 4},         // центр у угла
		{16, 16, 16, 0},      // вырожденный
		{16, 16, 16, -3},     // отрицательный
	}
	for _, tc := range cases {
		got := NewSphere(tc.cx, tc.cy, tc.cz, tc.r, 1)
		var want *Chunk
		if tc.r <= 0 {
			want = NewEmpty() // договорённая семантика: r <= 0 → пусто
		} else {
			want = bruteForceSphere(tc.cx, tc.cy, tc.cz, tc.r, 1)
		}
		if !equalChunks(got, want) {
			t.Errorf("NewSphere(%v,%v,%v, r=%v) не совпадает с brute force",
				tc.cx, tc.cy, tc.cz, tc.r)
		}
	}
}

func TestSphereSymmetry(t *testing.T) {
	// Сфера с центром (16,16,16) радиуса R должна быть симметрична
	// относительно центра чанка по всем осям.
	c := NewSphere(16, 16, 16, 10, 1)
	for z := 0; z < ChunkSize; z++ {
		for y := 0; y < ChunkSize; y++ {
			for x := 0; x < ChunkSize; x++ {
				if c.Get(x, y, z) != c.Get(31-x, 31-y, 31-z) {
					t.Fatalf("асимметрия в (%d,%d,%d)", x, y, z)
				}
			}
		}
	}
}

