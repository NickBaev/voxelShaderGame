package voxel

// Stats — CPU-статистика чанка. Только для аналитических тестов
// и последующей сверки с GPU-результатами; не является частью
// production pipeline.

type Stats struct {
	// SolidVoxels — число непустых вокселей.
	SolidVoxels int
	// VisibleFaces — число граней твёрдых вокселей, граничащих
	// с воздухом (включая границы чанка — за ними считается воздух).
	VisibleFaces int
}

// Stats считает статистику наивным проходом по всем вокселям.
// Для каждого твёрдого вокселя проверяются 6 соседей.
// Get за границей чанка возвращает Air, поэтому внешние
// границы учитываются автоматически.
func (c *Chunk) Stats() Stats {
	var s Stats
	for z := 0; z < ChunkSize; z++ {
		for y := 0; y < ChunkSize; y++ {
			for x := 0; x < ChunkSize; x++ {
				if c.Get(x, y, z) == Air {
					continue
				}
				s.SolidVoxels++
				if c.Get(x+1, y, z) == Air {
					s.VisibleFaces++
				}
				if c.Get(x-1, y, z) == Air {
					s.VisibleFaces++
				}
				if c.Get(x, y+1, z) == Air {
					s.VisibleFaces++
				}
				if c.Get(x, y-1, z) == Air {
					s.VisibleFaces++
				}
				if c.Get(x, y, z+1) == Air {
					s.VisibleFaces++
				}
				if c.Get(x, y, z-1) == Air {
					s.VisibleFaces++
				}
			}
		}
	}
	return s
}
