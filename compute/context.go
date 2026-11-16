// Package compute — минимальный headless compute-путь Vulkan для Этапа 2.
// Окно, surface и swapchain не нужны: только instance, device, буферы,
// compute pipeline и readback на хост.
package compute

import (
	"errors"
	"fmt"

	vk "github.com/goki/vulkan"
)

// Context держит всё, что живёт на протяжении сессии Vulkan.
type Context struct {
	Instance       vk.Instance
	PhysicalDevice vk.PhysicalDevice
	Device         vk.Device
	Queue          vk.Queue

	// QueueFamily — индекс семейства очередей с COMPUTE битом.
	QueueFamily uint32
}

// Init создаёт headless-контекст: загружает proc address, instance,
// выбирает физическое устройство с compute-очередью, создаёт device.
func Init() (*Context, error) {
	// Шаги 1–2: загрузка Vulkan. Без windowing library —
	// SetDefaultGetInstanceProcAddr берёт libvulkan из системы (Linux).
	if err := vk.SetDefaultGetInstanceProcAddr(); err != nil {
		return nil, fmt.Errorf("SetDefaultGetInstanceProcAddr: %w", err)
	}
	if err := vk.Init(); err != nil {
		return nil, fmt.Errorf("vk.Init: %w", err)
	}

	// Шаг 3: instance. Это "точка входа" Vulkan: через него перечисляются
	// GPU и подключаются validation layers. Пока без extensions и layers.
	appInfo := &vk.ApplicationInfo{
		SType:              vk.StructureTypeApplicationInfo,
		PApplicationName:   "voxelgame-compute",
		ApplicationVersion: vk.MakeVersion(0, 1, 0),
		ApiVersion:         vk.MakeVersion(1, 2, 0), // 1.2 хватает для всего Этапа 2
	}
	createInfo := &vk.InstanceCreateInfo{
		SType:            vk.StructureTypeInstanceCreateInfo,
		PApplicationInfo: appInfo,
		// Валидационный слой подключаем через VK_INSTANCE_LAYERS,
		// а не здесь: c-for-go-биндинги goki искажают []string (см. README goki).
	}
	var instance vk.Instance
	if err := vk.Error(vk.CreateInstance(createInfo, nil, &instance)); err != nil {
		return nil, fmt.Errorf("vk.CreateInstance: %w", err)
	}
	// ВАЖНО: vk.Init() резолвит указатели через vkGetInstanceProcAddr(NULL, …),
	// а спека гарантирует ненулевые указатели с NULL-instance только для
	// глобальных команд. Функции уровня instance (EnumeratePhysicalDevices
	// и все последующие) перерезолвим через реальный instance —
	// без этого segfault на вызове по нулевому указателю.
	if err := vk.InitInstance(instance); err != nil {
		vk.DestroyInstance(instance, nil)
		return nil, fmt.Errorf("vk.InitInstance: %w", err)
	}

	// Шаг 4: перечисление физических устройств — обязательно два прохода.
	// Первый с nil-слайсом возвращает только count, второй заполняет список.
	// Идём в обход невозможности узнать размер заранее.
	var count uint32
	if err := vk.Error(vk.EnumeratePhysicalDevices(instance, &count, nil)); err != nil {
		vk.DestroyInstance(instance, nil)
		return nil, fmt.Errorf("vk.EnumeratePhysicalDevices (count): %w", err)
	}
	if count == 0 {
		vk.DestroyInstance(instance, nil)
		return nil, errors.New("в системе нет Vulkan-устройств")
	}
	gpus := make([]vk.PhysicalDevice, count)
	if err := vk.Error(vk.EnumeratePhysicalDevices(instance, &count, gpus)); err != nil {
		vk.DestroyInstance(instance, nil)
		return nil, fmt.Errorf("vk.EnumeratePhysicalDevices (list): %w", err)
	}
	// gpus — физические устройства; дальше работаешь с ними.

	// Шаг 5: выбор GPU и семейства очередей с COMPUTE битом.
	// Предпочитаем дискретный GPU (тип DiscreteGpu), но берём любой
	// подходящий, чтобы код работал и на встройке.
	var (
		dev          vk.PhysicalDevice
		family       = ^uint32(0) // маркер «не найдено»
		haveDiscrete bool
	)
	for _, gpu := range gpus {
		var props vk.PhysicalDeviceProperties
		vk.GetPhysicalDeviceProperties(gpu, &props)
		props.Deref() // c-for-go не копирует C→Go автоматически
		isDiscrete := props.DeviceType == vk.PhysicalDeviceTypeDiscreteGpu

		var qCount uint32
		vk.GetPhysicalDeviceQueueFamilyProperties(gpu, &qCount, nil)
		families := make([]vk.QueueFamilyProperties, qCount)
		vk.GetPhysicalDeviceQueueFamilyProperties(gpu, &qCount, families)
		for i := range families {
			families[i].Deref() // то же для элементов слайса
			if families[i].QueueFlags&vk.QueueFlags(vk.QueueComputeBit) == 0 {
				continue
			}
			// Первый найденный подходит как запасной; дискретный вытесняет.
			if family == ^uint32(0) || (isDiscrete && !haveDiscrete) {
				dev, family = gpu, uint32(i)
				haveDiscrete = isDiscrete
			}
			break
		}
		if haveDiscrete {
			break // дискретный GPU с compute найден — дальше не ищем
		}
	}
	if family == ^uint32(0) {
		vk.DestroyInstance(instance, nil)
		return nil, errors.New("ни у одного GPU нет семейства очередей с QueueComputeBit")
	}

	// Шаг 6: логическое устройство с одной очередью из найденного семейства.
	queueInfo := vk.DeviceQueueCreateInfo{
		SType:            vk.StructureTypeDeviceQueueCreateInfo,
		QueueFamilyIndex: family,
		QueueCount:       1,
		PQueuePriorities: []float32{1.0},
	}
	deviceInfo := vk.DeviceCreateInfo{
		SType:                 vk.StructureTypeDeviceCreateInfo,
		QueueCreateInfoCount:  1,
		PQueueCreateInfos:     []vk.DeviceQueueCreateInfo{queueInfo},
	}
	var device vk.Device
	if err := vk.Error(vk.CreateDevice(dev, &deviceInfo, nil, &device)); err != nil {
		vk.DestroyInstance(instance, nil)
		return nil, fmt.Errorf("vk.CreateDevice: %w", err)
	}

	ctx := &Context{
		Instance:       instance,
		PhysicalDevice: dev,
		Device:         device,
		QueueFamily:    family,
	}
	// Шаг 7: очередь уже создана вместе с device — забираем её дескриптор.
	vk.GetDeviceQueue(device, family, 0, &ctx.Queue)
	return ctx, nil
}

// Destroy освобождает всё в обратном порядке создания.
func (c *Context) Destroy() {
	// Device уничтожается первым, но только если был создан (шаг 6).
	if c.Device != nil {
		vk.DestroyDevice(c.Device, nil)
	}
	if c.Instance != nil {
		vk.DestroyInstance(c.Instance, nil)
	}
}
