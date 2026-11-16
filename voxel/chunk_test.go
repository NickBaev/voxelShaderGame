package voxel

import (
	"errors"
	"testing"
)

func TestIndexOrder(t *testing.T) {
	c := &Chunk{}

	cases := []struct {
		x, y, z int
		want    int
	}{
		{0, 0, 0, 0},
		{1, 0, 0, 1},                          // x — самая быстрая ось
		{0, 1, 0, ChunkSize},                  // y идёт целым слоем
		{0, 0, 1, ChunkSize * ChunkSize},      // z — целая плоскость
		{31, 31, 31, VoxelCount - 1},
		{31, 0, 0, 31},
		{0, 31, 0, 31 * ChunkSize},
		{0, 0, 31, 31 * ChunkSize * ChunkSize},
	}
	for _, tc := range cases {
		got, err := c.Index(tc.x, tc.y, tc.z)
		if err != nil {
			t.Fatalf("Index(%d,%d,%d): неожиданная ошибка: %v", tc.x, tc.y, tc.z, err)
		}
		if got != tc.want {
			t.Errorf("Index(%d,%d,%d) = %d, хотим %d", tc.x, tc.y, tc.z, got, tc.want)
		}
	}

	// x и y не должны перемешиваться: разные индексы у разных координат.
	ix, _ := c.Index(1, 0, 0)
	iy, _ := c.Index(0, 1, 0)
	if ix == iy {
		t.Errorf("Index(1,0,0) == Index(0,1,0) — порядок осей сломан")
	}
}

func TestIndexOutOfBounds(t *testing.T) {
	c := &Chunk{}
	bad := [][3]int{{-1, 0, 0}, {0, -1, 0}, {0, 0, -1}, {ChunkSize, 0, 0}, {0, ChunkSize, 0}, {0, 0, ChunkSize}, {100, 100, 100}}
	for _, b := range bad {
		if _, err := c.Index(b[0], b[1], b[2]); !errors.Is(err, ErrOutOfBounds) {
			t.Errorf("Index(%d,%d,%d): err = %v, хотим ErrOutOfBounds", b[0], b[1], b[2], err)
		}
	}
}

func TestInBounds(t *testing.T) {
	c := &Chunk{}
	inside := [][3]int{{0, 0, 0}, {31, 31, 31}, {16, 16, 16}, {31, 0, 0}}
	for _, p := range inside {
		if !c.InBounds(p[0], p[1], p[2]) {
			t.Errorf("InBounds(%d,%d,%d) = false, хотим true", p[0], p[1], p[2])
		}
	}
	outside := [][3]int{{-1, 0, 0}, {0, 0, -1}, {ChunkSize, 0, 0}, {32, 32, 32}, {-100, 50, 50}}
	for _, p := range outside {
		if c.InBounds(p[0], p[1], p[2]) {
			t.Errorf("InBounds(%d,%d,%d) = true, хотим false", p[0], p[1], p[2])
		}
	}
}

func TestGetOutOfBoundsIsAir(t *testing.T) {
	c := &Chunk{}
	c.Fill(7) // чанк заполнен, чтобы граница точно возвращала 0, а не значение
	outside := [][3]int{{-1, 0, 0}, {0, -1, 0}, {0, 0, -1}, {ChunkSize, 0, 0}, {0, ChunkSize, 0}, {0, 0, ChunkSize}, {100, 100, 100}}
	for _, p := range outside {
		if got := c.Get(p[0], p[1], p[2]); got != Air {
			t.Errorf("Get(%d,%d,%d) = %d, хотим Air(0)", p[0], p[1], p[2], got)
		}
	}
}

func TestSetGetRoundtrip(t *testing.T) {
	c := &Chunk{}

	// Углы, рёбра, грани, центр — всё, где легко ошибиться с индексом.
	points := [][3]int{
		{0, 0, 0}, {31, 0, 0}, {0, 31, 0}, {0, 0, 31},
		{31, 31, 0}, {31, 0, 31}, {0, 31, 31}, {31, 31, 31},
		{1, 0, 0}, {0, 1, 0}, {0, 0, 1},
		{15, 16, 17},
	}
	for i, p := range points {
		m := uint32(100 + i)
		if err := c.Set(p[0], p[1], p[2], m); err != nil {
			t.Fatalf("Set(%d,%d,%d): %v", p[0], p[1], p[2], err)
		}
		if got := c.Get(p[0], p[1], p[2]); got != m {
			t.Errorf("Get(%d,%d,%d) = %d, хотим %d", p[0], p[1], p[2], got, m)
		}
	}

	// Соседи записанных точек не должны измениться.
	for _, p := range points {
		for d := 0; d < 3; d++ {
			q := p
			q[d]++
			if q[d] >= ChunkSize {
				continue
			}
			isPoint := false
			for _, r := range points {
				if r == q {
					isPoint = true
					break
				}
			}
			if !isPoint && c.Get(q[0], q[1], q[2]) != Air {
				t.Errorf("воксель %v соседен с записанной точкой %v, но не равен Air", q, p)
			}
		}
	}
}

func TestSetOutOfBounds(t *testing.T) {
	c := &Chunk{}
	bad := [][3]int{{-1, 0, 0}, {0, 0, -1}, {ChunkSize, 0, 0}, {0, 0, ChunkSize}, {32, 32, 32}}
	for _, b := range bad {
		if err := c.Set(b[0], b[1], b[2], 9); !errors.Is(err, ErrOutOfBounds) {
			t.Errorf("Set(%d,%d,%d): err = %v, хотим ErrOutOfBounds", b[0], b[1], b[2], err)
		}
	}
	// Чанк остался пустым: ни одна ошибочная запись не прошла.
	for z := 0; z < ChunkSize; z++ {
		for y := 0; y < ChunkSize; y++ {
			for x := 0; x < ChunkSize; x++ {
				if c.Get(x, y, z) != Air {
					t.Fatalf("чанк изменён после Set вне границ (%d,%d,%d)", x, y, z)
				}
			}
		}
	}
}

func TestFill(t *testing.T) {
	c := &Chunk{}
	c.Fill(5)
	for z := 0; z < ChunkSize; z++ {
		for y := 0; y < ChunkSize; y++ {
			for x := 0; x < ChunkSize; x++ {
				if got := c.Get(x, y, z); got != 5 {
					t.Fatalf("Fill(5): Get(%d,%d,%d) = %d, хотим 5", x, y, z, got)
				}
			}
		}
	}

	c.Fill(0) // очистка в воздух
	for z := 0; z < ChunkSize; z++ {
		for y := 0; y < ChunkSize; y++ {
			for x := 0; x < ChunkSize; x++ {
				if got := c.Get(x, y, z); got != Air {
					t.Fatalf("Fill(0): Get(%d,%d,%d) = %d, хотим Air", x, y, z, got)
				}
			}
		}
	}
}
