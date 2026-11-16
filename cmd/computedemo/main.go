// Демо Этапа 2.1: запускает echo-шейдер на GPU и проверяет readback.
// Каждый инвокейшен пишет свой gl_GlobalInvocationID в свой слот буфера;
// успех = data[i] == i для всех i.
package main

import (
	"fmt"
	"os"
	"unsafe"

	vk "github.com/goki/vulkan"

	"voxelgame/compute"
	"voxelgame/compute/shaders"
)

const n = 1024 // число инвокейшенов = число элементов буфера

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "ошибка:", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. Контекст: instance + device + compute-очередь.
	ctx, err := compute.Init()
	if err != nil {
		return err
	}
	defer ctx.Destroy()

	// 2. Буфер: n элементов uint32, host-visible.
	buf, err := compute.CreateHostVisibleBuffer(ctx, n*4)
	if err != nil {
		return err
	}
	defer buf.Destroy(ctx)

	// 3. Descriptor: layout → set с привязанным буфером.
	setLayout, err := compute.DescriptorLayout(ctx)
	if err != nil {
		return err
	}
	defer compute.DestroyDescriptorSetLayout(ctx, setLayout)

	set, pool, err := compute.AllocateDescriptorSet(ctx, setLayout, buf)
	if err != nil {
		return err
	}
	defer compute.DestroyDescriptorPool(ctx, pool)

	// 4. Pipeline: layout + compute pipeline из echo SPIR-V.
	pipeLayout, err := compute.PipelineLayout(ctx, []vk.DescriptorSetLayout{setLayout})
	if err != nil {
		return err
	}
	defer compute.DestroyPipelineLayout(ctx, pipeLayout)

	pipe, err := compute.CreateComputePipeline(ctx, shaders.EchoSPIRV, pipeLayout)
	if err != nil {
		return err
	}
	defer pipe.Destroy(ctx)

	// 5. Command buffer: пул → аллокация → запись.
	poolInfo := vk.CommandPoolCreateInfo{
		SType:            vk.StructureTypeCommandPoolCreateInfo,
		Flags:            vk.CommandPoolCreateFlags(vk.CommandPoolCreateResetCommandBufferBit),
		QueueFamilyIndex: ctx.QueueFamily,
	}
	var cmdPool vk.CommandPool
	if err := vk.Error(vk.CreateCommandPool(ctx.Device, &poolInfo, nil, &cmdPool)); err != nil {
		return fmt.Errorf("vk.CreateCommandPool: %w", err)
	}
	defer vk.DestroyCommandPool(ctx.Device, cmdPool, nil)

	allocInfo := vk.CommandBufferAllocateInfo{
		SType:              vk.StructureTypeCommandBufferAllocateInfo,
		CommandPool:        cmdPool,
		Level:              vk.CommandBufferLevelPrimary,
		CommandBufferCount: 1,
	}
	cmdBuffers := make([]vk.CommandBuffer, 1)
	if err := vk.Error(vk.AllocateCommandBuffers(ctx.Device, &allocInfo, cmdBuffers)); err != nil {
		return fmt.Errorf("vk.AllocateCommandBuffers: %w", err)
	}
	cmd := cmdBuffers[0]

	beginInfo := vk.CommandBufferBeginInfo{
		SType: vk.StructureTypeCommandBufferBeginInfo,
	}
	if err := vk.Error(vk.BeginCommandBuffer(cmd, &beginInfo)); err != nil {
		return fmt.Errorf("vk.BeginCommandBuffer: %w", err)
	}
	vk.CmdBindPipeline(cmd, vk.PipelineBindPointCompute, pipe.Pipeline)
	vk.CmdBindDescriptorSets(cmd, vk.PipelineBindPointCompute, pipeLayout,
		0, 1, []vk.DescriptorSet{set}, 0, nil)
	// 1024 потока / 256 на группу = 4 группы.
	vk.CmdDispatch(cmd, n/256, 1, 1)
	if err := vk.Error(vk.EndCommandBuffer(cmd)); err != nil {
		return fmt.Errorf("vk.EndCommandBuffer: %w", err)
	}

	// 6. Fence + submit + ожидание завершения GPU.
	fenceInfo := vk.FenceCreateInfo{
		SType: vk.StructureTypeFenceCreateInfo,
	}
	var fence vk.Fence
	if err := vk.Error(vk.CreateFence(ctx.Device, &fenceInfo, nil, &fence)); err != nil {
		return fmt.Errorf("vk.CreateFence: %w", err)
	}
	defer vk.DestroyFence(ctx.Device, fence, nil)

	submitInfo := vk.SubmitInfo{
		SType:              vk.StructureTypeSubmitInfo,
		CommandBufferCount: 1,
		PCommandBuffers:    cmdBuffers,
	}
	if err := vk.Error(vk.QueueSubmit(ctx.Queue, 1, []vk.SubmitInfo{submitInfo}, fence)); err != nil {
		return fmt.Errorf("vk.QueueSubmit: %w", err)
	}
	const timeoutNs = 10_000_000_000 // 10 с — с запасом на первый запуск
	if err := vk.Error(vk.WaitForFences(ctx.Device, 1, []vk.Fence{fence}, vk.True, timeoutNs)); err != nil {
		return fmt.Errorf("vk.WaitForFences: %w", err)
	}

	// 7. Readback и проверка: data[i] == i.
	data := make([]uint32, n)
	if err := buf.Read(uint32sToBytes(data)); err != nil {
		return err
	}
	for i, v := range data {
		if v != uint32(i) {
			return fmt.Errorf("data[%d] = %d, хотим %d — compute pipeline не совпал", i, v, i)
		}
	}
	fmt.Printf("OK: %d инвокейшенов, data[i] == i для всех i — Этап 2.1 пройден\n", n)
	return nil
}

// uint32sToBytes переинтерпретирует слайс без копирования
// (тот же приём, что с SPIR-V, но в обе стороны).
func uint32sToBytes(s []uint32) []byte {
	if len(s) == 0 {
		return nil
	}
	n := len(s) * 4
	return unsafe.Slice((*byte)(unsafe.Pointer(&s[0])), n)
}
