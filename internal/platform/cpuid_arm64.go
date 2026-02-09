package platform

import (
	"runtime"

	"golang.org/x/sys/cpu"
)

// CpuFeatures exposes the capabilities for this CPU, queried via the Has method.
var CpuFeatures = loadCpuFeatureFlags()

func loadCpuFeatureFlags() (flags CpuFeatureFlags) {
	if cpu.ARM64.HasATOMICS || runtime.GOOS == "darwin" {
		// macOS does not allow userland to read the instruction set attribute registers,
		// but all M-series SoCs support LSE (atomics).
		flags |= CpuFeatureArm64Atomic
	}
	return
}
