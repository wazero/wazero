package platform

import "golang.org/x/sys/windows"

// CpuFeatures exposes the capabilities for this CPU, queried via the Has method.
var CpuFeatures = loadCpuFeatureFlags()

func loadCpuFeatureFlags() (flags CpuFeatureFlags) {
	// See: go.dev/cl/757482
	// golang.org/x/sys/cpu can't import golang.org/x/sys/windows and can't do this yet.
	if windows.IsProcessorFeaturePresent(windows.PF_ARM_V81_ATOMIC_INSTRUCTIONS_AVAILABLE) {
		flags |= CpuFeatureArm64Atomic
	}
	return
}
