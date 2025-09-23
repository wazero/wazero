package spectest

import (
	"context"
	"embed"
	"testing"

	"github.com/wazero/wazero"
	"github.com/wazero/wazero/api"
	"github.com/wazero/wazero/experimental"
	"github.com/wazero/wazero/internal/integration_test/spectest"
	"github.com/wazero/wazero/internal/platform"
)

//go:embed testdata/*.wasm
//go:embed testdata/*.json
var testcases embed.FS

const enabledFeatures = api.CoreFeaturesV2 | experimental.CoreFeaturesTailCall

func TestCompiler(t *testing.T) {
	if !platform.CompilerSupported() {
		t.Skip()
	}
	spectest.Run(t, testcases, context.Background(), wazero.NewRuntimeConfigCompiler().WithCoreFeatures(enabledFeatures))
}

func TestInterpreter(t *testing.T) {
	spectest.Run(t, testcases, context.Background(), wazero.NewRuntimeConfigInterpreter().WithCoreFeatures(enabledFeatures))
}
