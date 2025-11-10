//go:build !(amd64 || arm64) || !gc

package platform

var CpuFeatures = func() CpuFeatureFlags { return 0 }
