// Package shaders содержит SPIR-V модули, встроенные в бинарь.
// Сборка .spv — через Makefile (glslc); без него этот пакет не соберётся.
package shaders

import _ "embed"

//go:embed echo.comp.spv
var EchoSPIRV []byte
