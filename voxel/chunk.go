// Package voxel реализует ядро воксельных данных первого прототипа:
// один чанк 32×32×32, один material ID на воксель (uint32), Air = 0.
package voxel

import "errors"

// ChunkSize — размер чанка по каждой оси.
const ChunkSize = 32

// VoxelCount — общее число вокселей в чанке.
const VoxelCount = ChunkSize * ChunkSize * ChunkSize

// Air — пустой воксель. Все значения != 0 — твёрдые материалы.
const Air = 0

// ErrOutOfBounds возвращается при обращении к координатам вне 0..ChunkSize-1.
var ErrOutOfBounds = errors.New("voxel: координаты вне границ чанка")

// Chunk — один воксельный чанк.
// Хранение: voxels[x + ChunkSize*(y + ChunkSize*z)], x — самая быстрая ось.
// Воксель (x,y,z) занимает [x,x+1) × [y,y+1) × [z,z+1).
type Chunk struct {
	voxels [VoxelCount]uint32
}

// InBounds сообщает, лежат ли локальные координаты внутри чанка.
func (c *Chunk) InBounds(x, y, z int) bool {
	return x < ChunkSize && y < ChunkSize && z < ChunkSize && x >= 0 && y >= 0 && z >= 0

}

// Index вычисляет линейный индекс вокселя по локальным координатам.
// Порядок: x + ChunkSize*(y + ChunkSize*z).
func (c *Chunk) Index(x, y, z int) (int, error) {
	if !c.InBounds(x, y, z) {
		return 0, ErrOutOfBounds
	}

	return x + ChunkSize*(y+z*ChunkSize), nil
}

// Get возвращает material ID вокселя.
// Все координаты за границами чанка считаются воздухом (0).
func (c *Chunk) Get(x, y, z int) uint32{
	index, err := c.Index(x, y, z)
	if err != nil {
		return 0
	}

	return c.voxels[index]
}

// Set записывает material ID вокселя по локальным координатам.
// Возвращает ErrOutOfBounds, если координаты вне чанка.
func (c *Chunk) Set(x, y, z int, m uint32) error {
	index, err := c.Index(x, y, z)
	if err != nil {
		return err
	}
	c.voxels[index] = m

	return nil
}

// Fill заполняет весь чанк материалом m (0 = очистка в воздух).
func (c *Chunk) Fill(m uint32) {
	for k := range c.voxels {
		c.voxels[k] = m
	}
}
