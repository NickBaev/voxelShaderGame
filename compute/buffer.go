package compute

import (
	"errors"
	"fmt"
	"unsafe"

	vk "github.com/goki/vulkan"
)

// Buffer — GPU-буфер + привязанная память.
// Память HostVisible | HostCoherent: хост читает/пишет через map
// без vkFlushMappedMemoryRanges и vkInvalidateMappedMemoryRanges.
type Buffer struct {
	Buffer vk.Buffer
	Memory vk.DeviceMemory
	Size   vk.DeviceSize

	device vk.Device // нужен методам Write/Read/Destroy
}

// CreateHostVisibleBuffer создаёт буфер размера size байт,
// доступный и устройству (storage), и хосту (map).
func CreateHostVisibleBuffer(ctx *Context, size int) (*Buffer, error) {
	if size <= 0 {
		return nil, errors.New("размер буфера должен быть > 0")
	}

	info := vk.BufferCreateInfo{
		SType:       vk.StructureTypeBufferCreateInfo,
		Size:        vk.DeviceSize(size),
		Usage:       vk.BufferUsageFlags(vk.BufferUsageStorageBufferBit | vk.BufferUsageTransferSrcBit),
		SharingMode: vk.SharingModeExclusive,
	}
	var buf vk.Buffer
	if err := vk.Error(vk.CreateBuffer(ctx.Device, &info, nil, &buf)); err != nil {
		return nil, fmt.Errorf("vk.CreateBuffer: %w", err)
	}

	// Требования буфера: alignment и биты допустимых memory types.
	var memReqs vk.MemoryRequirements
	vk.GetBufferMemoryRequirements(ctx.Device, buf, &memReqs)
	memReqs.Deref()

	// Свойства памяти GPU: ищем тип, который (а) разрешён буфером
	// (бит i в MemoryTypeBits) и (б) имеет HostVisible | HostCoherent.
	var memProps vk.PhysicalDeviceMemoryProperties
	vk.GetPhysicalDeviceMemoryProperties(ctx.PhysicalDevice, &memProps)
	memProps.Deref()
	// Вложенные элементы массивов тоже дерефаем вручную (c-for-go):
	// после Deref внешней структуры у них заполнены только ref-поля.
	for i := uint32(0); i < memProps.MemoryTypeCount; i++ {
		memProps.MemoryTypes[i].Deref()
	}

	const required = vk.MemoryPropertyFlags(vk.MemoryPropertyHostVisibleBit | vk.MemoryPropertyHostCoherentBit)
	typeIndex := -1
	for i := uint32(0); i < memProps.MemoryTypeCount; i++ {
		if memReqs.MemoryTypeBits&(1<<i) == 0 {
			continue
		}
		if memProps.MemoryTypes[i].PropertyFlags&required != required {
			continue
		}
		typeIndex = int(i)
		break
	}
	if typeIndex < 0 {
		vk.DestroyBuffer(ctx.Device, buf, nil)
		return nil, errors.New("нет memory type с HostVisible|HostCoherent, подходящего буферу")
	}

	allocInfo := vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReqs.Size, // берём требуемый размер, не наш
		MemoryTypeIndex: uint32(typeIndex),
	}
	var mem vk.DeviceMemory
	if err := vk.Error(vk.AllocateMemory(ctx.Device, &allocInfo, nil, &mem)); err != nil {
		vk.DestroyBuffer(ctx.Device, buf, nil)
		return nil, fmt.Errorf("vk.AllocateMemory: %w", err)
	}
	if err := vk.Error(vk.BindBufferMemory(ctx.Device, buf, mem, 0)); err != nil {
		vk.DestroyBuffer(ctx.Device, buf, nil)
		vk.FreeMemory(ctx.Device, mem, nil)
		return nil, fmt.Errorf("vk.BindBufferMemory: %w", err)
	}

	return &Buffer{Buffer: buf, Memory: mem, Size: vk.DeviceSize(size), device: ctx.Device}, nil
}

// Write копирует данные из CPU в буфер через mapped memory.
func (b *Buffer) Write(p []byte) error {
	if vk.DeviceSize(len(p)) > b.Size {
		return fmt.Errorf("Write: %d байт не влезает в буфер размера %d", len(p), b.Size)
	}
	var ptr unsafe.Pointer
	if err := vk.Error(vk.MapMemory(b.device, b.Memory, 0, b.Size, 0, &ptr)); err != nil {
		return fmt.Errorf("vk.MapMemory: %w", err)
	}
	copy(unsafe.Slice((*byte)(ptr), len(p)), p)
	vk.UnmapMemory(b.device, b.Memory)
	return nil
}

// Read копирует данные из буфера в CPU.
// Память HostCoherent, поэтому invalidate не нужен — GPU-записи
// видны хосту автоматически после завершения выполнения.
func (b *Buffer) Read(p []byte) error {
	if vk.DeviceSize(len(p)) > b.Size {
		return fmt.Errorf("Read: %d байт больше буфера размера %d", len(p), b.Size)
	}
	var ptr unsafe.Pointer
	if err := vk.Error(vk.MapMemory(b.device, b.Memory, 0, b.Size, 0, &ptr)); err != nil {
		return fmt.Errorf("vk.MapMemory: %w", err)
	}
	copy(p, unsafe.Slice((*byte)(ptr), len(p)))
	vk.UnmapMemory(b.device, b.Memory)
	return nil
}

// Destroy освобождает память и буфер
func (b *Buffer) Destroy(ctx *Context) {
	vk.DestroyBuffer(ctx.Device, b.Buffer, nil)
	vk.FreeMemory(ctx.Device, b.Memory, nil)
}
