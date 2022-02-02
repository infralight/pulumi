package nodejs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pulumi/pulumi/pkg/v3/codegen"
	"github.com/pulumi/pulumi/pkg/v3/codegen/internal/test"
	"github.com/pulumi/pulumi/pkg/v3/testing/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/executable"
)

func TestGenerateProgram(t *testing.T) {
	test.TestProgramCodegen(t,
		test.ProgramCodegenOptions{
			Language:   "nodejs",
			Extension:  "ts",
			OutputFile: "index.ts",
			Check:      nodejsCheck,
			GenProgram: GenerateProgram,
		})
}

func nodejsCheck(t *testing.T, path string, dependencies codegen.StringSet) {
	ex, err := executable.FindExecutable("yarn")
	require.NoError(t, err, "Could not find yarn executable")
	dir := filepath.Dir(path)
	pkgs := nodejsPackages(t, dependencies)
	// We delete and regenerate package files for each run.
	packageJSONPath := filepath.Join(dir, "package.json")
	if err := os.Remove(packageJSONPath); !os.IsNotExist(err) {
		require.NoError(t, err, "Failed to delete %s", packageJSONPath)
	}
	yarnLock := filepath.Join(dir, "yarn.lock")
	if err := os.Remove(yarnLock); !os.IsNotExist(err) {
		require.NoError(t, err, "Failed to delete %s", yarnLock)
	}

	pkgInfo := npmPackage{
		Dependencies: map[string]string{
			"@pulumi/pulumi": "latest",
		},
		DevDependencies: map[string]string{
			"@types/node": "latest",
			"typescript":  "latest",
		},
	}
	for pkg, v := range pkgs {
		pkgInfo.Dependencies[pkg] = v
	}
	pkgJSON, err := json.MarshalIndent(pkgInfo, "", "    ")
	require.NoError(t, err)
	err = os.WriteFile(packageJSONPath, pkgJSON, 0600)
	require.NoError(t, err)

	err = integration.RunCommand(t, "Install dependencies",
		[]string{ex, "install"},
		dir, &integration.ProgramTestOptions{})
	require.NoError(t, err, "Failed install")

	err = integration.RunCommand(t, "tsc check",
		[]string{ex, "run", "tsc", "--noEmit", filepath.Base(path)}, dir, &integration.ProgramTestOptions{})
	require.NoError(t, err, "Failed to build %q", path)
}

// Returns the nodejs equivalent to the hcl2 package names provided.
func nodejsPackages(t *testing.T, deps codegen.StringSet) map[string]string {
	result := make(map[string]string, len(deps))
	for _, d := range deps.SortedValues() {
		r := fmt.Sprintf("@pulumi/%s", d)
		v := func(s string) {
			result[r] = "^" + s
		}
		switch d {
		case "aws":
			v(test.AwsSchema)
		case "azure-native":
			v(test.AzureNativeSchema)
		case "azure":
			v(test.AzureSchema)
		case "kubernetes":
			v(test.KubernetesSchema)
		case "random":
			v(test.RandomSchema)
		default:
			t.Logf("Unknown package requested: %s", d)
		}

	}
	return result
}
