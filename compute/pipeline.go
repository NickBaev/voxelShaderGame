package compute

import (
	"fmt"
	"unsafe"

	vk "github.com/goki/vulkan"
)

// Pipeline — compute pipeline и его layout.
type Pipeline struct {
	Pipeline vk.Pipeline
	Layout   vk.PipelineLayout
}

// CreateComputePipeline строит compute pipeline из SPIR-V.
// spirv — сырые байты .spv (длина кратна 4). layout — из PipelineLayout.
func CreateComputePipeline(ctx *Context, spirv []byte, layout vk.PipelineLayout) (*Pipeline, error) {
	if len(spirv) == 0 || len(spirv)%4 != 0 {
		return nil, fmt.Errorf("SPIR-V: длина %d не кратна 4", len(spirv))
	}
	// SPIR-V — поток 32-битных слов little-endian; переинтерпретируем байты.
	code := unsafe.Slice((*uint32)(unsafe.Pointer(&spirv[0])), len(spirv)/4)

	moduleInfo := vk.ShaderModuleCreateInfo{
		SType:    vk.StructureTypeShaderModuleCreateInfo,
		CodeSize: uint64(len(spirv)),
		PCode:    code,
	}
	var module vk.ShaderModule
	if err := vk.Error(vk.CreateShaderModule(ctx.Device, &moduleInfo, nil, &module)); err != nil {
		return nil, fmt.Errorf("vk.CreateShaderModule: %w", err)
	}
	// Модуль нужен только при создании pipeline — уничтожаем сразу после.
	defer vk.DestroyShaderModule(ctx.Device, module, nil)

	stage := vk.PipelineShaderStageCreateInfo{
		SType:  vk.StructureTypePipelineShaderStageCreateInfo,
		Stage:  vk.ShaderStageComputeBit,
		Module: module,
		// goki не копирует строку в C и не добавляет NUL: Go-строки не
		// null-terminated, драйвер читал бы мусор за пределами. Явный \0
		// делает байты "main\0" валидной C-строкой.
		PName: "main\x00",
	}
	pipelineInfo := vk.ComputePipelineCreateInfo{
		SType:              vk.StructureTypeComputePipelineCreateInfo,
		Stage:              stage,
		Layout:             layout,
		BasePipelineHandle: nil,
		BasePipelineIndex:  -1, // нет базового pipeline — NVIDIA требует явно
	}
	pipelines := make([]vk.Pipeline, 1)
	if err := vk.Error(vk.CreateComputePipelines(ctx.Device, nil, 1, []vk.ComputePipelineCreateInfo{pipelineInfo}, nil, pipelines)); err != nil {
		return nil, fmt.Errorf("vk.CreateComputePipelines: %w", err)
	}
	return &Pipeline{Pipeline: pipelines[0], Layout: layout}, nil
}

// Destroy освобождает pipeline (layout — отдельно, см. DestroyPipelineLayout).
func (p *Pipeline) Destroy(ctx *Context) {
	vk.DestroyPipeline(ctx.Device, p.Pipeline, nil)
}

// PipelineLayout создаёт layout из набора descriptor set layouts.
// Для Этапа 2 — один layout из одного storage buffer.
func PipelineLayout(ctx *Context, setLayouts []vk.DescriptorSetLayout) (vk.PipelineLayout, error) {
	info := vk.PipelineLayoutCreateInfo{
		SType:          vk.StructureTypePipelineLayoutCreateInfo,
		SetLayoutCount: uint32(len(setLayouts)),
		PSetLayouts:    setLayouts,
	}
	var layout vk.PipelineLayout
	if err := vk.Error(vk.CreatePipelineLayout(ctx.Device, &info, nil, &layout)); err != nil {
		return nil, fmt.Errorf("vk.CreatePipelineLayout: %w", err)
	}
	return layout, nil
}

// DestroyPipelineLayout освобождает pipeline layout.
func DestroyPipelineLayout(ctx *Context, layout vk.PipelineLayout) {
	vk.DestroyPipelineLayout(ctx.Device, layout, nil)
}

// DescriptorLayout создаёт layout из одного storage buffer (binding 0),
// доступного compute-стадии.
func DescriptorLayout(ctx *Context) (vk.DescriptorSetLayout, error) {
	binding := vk.DescriptorSetLayoutBinding{
		Binding:         0,
		DescriptorType:  vk.DescriptorTypeStorageBuffer,
		DescriptorCount: 1,
		StageFlags:      vk.ShaderStageFlags(vk.ShaderStageComputeBit),
	}
	info := vk.DescriptorSetLayoutCreateInfo{
		SType:        vk.StructureTypeDescriptorSetLayoutCreateInfo,
		BindingCount: 1,
		PBindings:    []vk.DescriptorSetLayoutBinding{binding},
	}
	var layout vk.DescriptorSetLayout
	if err := vk.Error(vk.CreateDescriptorSetLayout(ctx.Device, &info, nil, &layout)); err != nil {
		return nil, fmt.Errorf("vk.CreateDescriptorSetLayout: %w", err)
	}
	return layout, nil
}

// DestroyDescriptorSetLayout освобождает layout descriptor set'ов.
func DestroyDescriptorSetLayout(ctx *Context, layout vk.DescriptorSetLayout) {
	vk.DestroyDescriptorSetLayout(ctx.Device, layout, nil)
}

// AllocateDescriptorSet создаёт пул, аллоцирует один set и привязывает
// буфер к binding 0. Пул возвращается для последующего DestroyDescriptorPool
// (set'ы освобождаются вместе с пулом, отдельного destroy у них нет).
func AllocateDescriptorSet(ctx *Context, layout vk.DescriptorSetLayout, b *Buffer) (vk.DescriptorSet, vk.DescriptorPool, error) {
	poolInfo := vk.DescriptorPoolCreateInfo{
		SType:         vk.StructureTypeDescriptorPoolCreateInfo,
		MaxSets:       1,
		PoolSizeCount: 1,
		PPoolSizes: []vk.DescriptorPoolSize{{
			Type:            vk.DescriptorTypeStorageBuffer,
			DescriptorCount: 1,
		}},
	}
	var pool vk.DescriptorPool
	if err := vk.Error(vk.CreateDescriptorPool(ctx.Device, &poolInfo, nil, &pool)); err != nil {
		return nil, nil, fmt.Errorf("vk.CreateDescriptorPool: %w", err)
	}

	allocInfo := vk.DescriptorSetAllocateInfo{
		SType:              vk.StructureTypeDescriptorSetAllocateInfo,
		DescriptorPool:     pool,
		DescriptorSetCount: 1,
		PSetLayouts:        []vk.DescriptorSetLayout{layout},
	}
	var set vk.DescriptorSet
	if err := vk.Error(vk.AllocateDescriptorSets(ctx.Device, &allocInfo, &set)); err != nil {
		vk.DestroyDescriptorPool(ctx.Device, pool, nil)
		return nil, nil, fmt.Errorf("vk.AllocateDescriptorSets: %w", err)
	}

	write := vk.WriteDescriptorSet{
		SType:           vk.StructureTypeWriteDescriptorSet,
		DstSet:          set,
		DstBinding:      0,
		DescriptorCount: 1,
		DescriptorType:  vk.DescriptorTypeStorageBuffer,
		PBufferInfo: []vk.DescriptorBufferInfo{{
			Buffer: b.Buffer,
			Offset: 0,
			Range:  b.Size,
		}},
	}
	vk.UpdateDescriptorSets(ctx.Device, 1, []vk.WriteDescriptorSet{write}, 0, nil)
	return set, pool, nil
}

// DestroyDescriptorPool освобождает пул вместе со всеми set'ами из него.
func DestroyDescriptorPool(ctx *Context, pool vk.DescriptorPool) {
	vk.DestroyDescriptorPool(ctx.Device, pool, nil)
}
