package voxel

import "math"

// Генераторы тестовых сцен первого прототипа.
// Каждый генератор возвращает новый заполненный чанк.

// NewEmpty возвращает пустой чанк (весь воздух).
func NewEmpty() *Chunk {
	chunk := Chunk{}
	chunk.Fill(0)
	return &chunk
}

// NewSolid возвращает полностью заполненный чанк материалом m.
func NewSolid(m uint32) *Chunk {
	chunk := Chunk{}
	chunk.Fill(m)
	return &chunk
}

// NewSingleVoxel возвращает чанк с одним твёрдым вокселем
// в заданных локальных координатах.
func NewSingleVoxel(x, y, z int, m uint32) *Chunk {
	chunk := NewEmpty()
	chunk.Set(x, y, z, m)
	return chunk
}

// NewTwoAdjacent возвращает чанк с двумя твёрдыми вокселями,
// соседними по оси X (общая внутренняя грань).
func NewTwoAdjacent(m uint32) *Chunk {
	x := 13
	y := 13
	z := 13

	chunk := NewSingleVoxel(x, y, z, m)
	chunk.Set(x+1, y, z, m)
	return chunk
}

// NewPlaneY возвращает горизонтальную плоскость толщиной один воксель
// на высоте y0: все воксели (x, y0, z), 0 ≤ x,z < ChunkSize.
func NewPlaneY(y0 int, m uint32) *Chunk {
	chunk := NewEmpty()
	for x := range ChunkSize {
		for z := range ChunkSize {
			chunk.Set(x, y0, z, m)
		}
	}

	return chunk
}

// NewSphere возвращает чанк с воксельной сферой радиуса r
// вокруг центра (cx, cy, cz). Центр вокселя — (x+0.5, y+0.5, z+0.5),
// чтобы форма была симметричной относительно центра чанка.
func NewSphere(cx, cy, cz, r float64, m uint32) *Chunk {
	chunk := NewEmpty()
	if r <= 0 {
		return chunk
	}

	for bboxX := max(math.Ceil(cx-r-0.5)-1, 0); bboxX < min(math.Floor(cx+r-0.5)+2, ChunkSize); bboxX++ {
		for bboxY := max(math.Ceil(cy-r-0.5)-1, 0); bboxY < min(math.Floor(cy+r-0.5)+2, ChunkSize); bboxY++ {
			for bboxZ := max(math.Ceil(cz-r-0.5)-1, 0); bboxZ < min(math.Floor(cz+r-0.5)+2, ChunkSize); bboxZ++ {
				distX := cx - float64(bboxX) - 0.5
				distY := cy - float64(bboxY) - 0.5
				distZ := cz - float64(bboxZ) - 0.5
				if math.Sqrt(distX*distX+distY*distY+distZ*distZ) <= r {
					chunk.Set(int(bboxX), int(bboxY), int(bboxZ), m)
				}
			}
		}
	}

	return chunk
}

// NewCheckerboard возвращает чанк с шахматным узором
// из материалов m1/m2 по чётности x+y+z (твёрдые воксели не соприкасаются).
func NewCheckerboard(m1, m2 uint32) *Chunk {
	chunk := NewEmpty()
	for x := range ChunkSize {
		for y := range ChunkSize {
			for z := range ChunkSize {
				if (x+y+z)%2 == 0 {
					chunk.Set(x, y, z, m1)
				} else {
					chunk.Set(x, y, z, m2)
				}
			}
		}
	}


	return chunk
}

// NewNoise возвращает чанк с детерминированным шумом:
// каждый воксель твёрдый с вероятностью density (0..1), иначе воздух.
// При одинаковом seed результат обязан быть побайтово идентичным —
// на этом будет сверяться GPU-версия.
func NewNoise(seed uint64, density float64, m uint32) *Chunk {
	chunk := NewEmpty()
	for x := range ChunkSize {
		for y := range ChunkSize {
			for z := range ChunkSize {
				hash := mix64(mix64(mix64(mix64(uint64(x))+uint64(y))+uint64(z)) + seed)
				normilisedHash := float64(hash>>11) * (1.0 / (1 << 53))
				if normilisedHash < density {
					chunk.Set(x, y, z, m)
				}
			}
		}
	}

	return chunk
}

func mix64(x uint64) uint64 {
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	x = x ^ (x >> 31)

	return x
}
