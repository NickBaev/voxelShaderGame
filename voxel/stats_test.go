package voxel

import "testing"

// TestStatsExpectedTable — аналитические значения из CONTEXT.md, раздел 8:
// для каждой сцены точные SolidVoxels и VisibleFaces.
func TestStatsExpectedTable(t *testing.T) {
	cases := []struct {
		name        string
		chunk       *Chunk
		solid       int
		visibleFace int
	}{
		{"empty", NewEmpty(), 0, 0},
		{"single center", NewSingleVoxel(16, 16, 16, 1), 1, 6},
		{"single corner", NewSingleVoxel(0, 0, 0, 1), 1, 6}, // границы чанка = воздух
		{"two adjacent", NewTwoAdjacent(1), 2, 10},
		{"solid", NewSolid(1), VoxelCount, 6 * ChunkSize * ChunkSize}, // 32768 / 6144
		{"plane", NewPlaneY(16, 1), ChunkSize * ChunkSize, 2*ChunkSize*ChunkSize + 4*ChunkSize}, // 1024 / 2176
		{"checkerboard", NewCheckerboard(1, Air), VoxelCount / 2, 6 * VoxelCount / 2},           // 16384 / 98304
	}

	for _, tc := range cases {
		s := tc.chunk.Stats()
		if s.SolidVoxels != tc.solid || s.VisibleFaces != tc.visibleFace {
			t.Errorf("%s: got %+v, хотим {SolidVoxels:%d VisibleFaces:%d}",
				tc.name, s, tc.solid, tc.visibleFace)
		}
	}
}

// TestStatsGenericInvariants — свойства, верные для любой сцены:
// SolidVoxels совпадает с прямым подсчётом, VisibleFaces чётно и не больше 6×Solid.
func TestStatsGenericInvariants(t *testing.T) {
	scenes := []*Chunk{
		NewEmpty(),
		NewSolid(2),
		NewSphere(16, 16, 16, 10, 1),
		NewNoise(42, 0.5, 1),
		NewNoise(43, 0.3, 1),
	}
	for i, c := range scenes {
		s := c.Stats()
		if s.SolidVoxels != countSolid(c) {
			t.Errorf("сцена %d: SolidVoxels=%d, прямой подсчёт=%d", i, s.SolidVoxels, countSolid(c))
		}
		if s.VisibleFaces > 6*s.SolidVoxels {
			t.Errorf("сцена %d: VisibleFaces=%d > 6×%d", i, s.VisibleFaces, s.SolidVoxels)
		}
		// Каждая пара соседних твёрдых убирает ровно 2 грани:
		// VisibleFaces = 6×Solid − 2×(число соседств) — всегда чётно.
		if s.VisibleFaces%2 != 0 {
			t.Errorf("сцена %d: VisibleFaces=%d нечётно", i, s.VisibleFaces)
		}
	}
}
