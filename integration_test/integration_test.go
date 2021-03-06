// Copyright 2016 Palantir Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package integration_test

import (
	"testing"

	"github.com/nmiyake/pkg/gofiles"
	"github.com/palantir/godel/v2/framework/pluginapitester"
	"github.com/palantir/godel/v2/pkg/products"
	"github.com/palantir/okgo/okgotester"
	"github.com/stretchr/testify/require"
)

const (
	okgoPluginLocator  = "com.palantir.okgo:check-plugin:1.12.0"
	okgoPluginResolver = "https://github.com/{{index GroupParts 1}}/{{index GroupParts 2}}/releases/download/v{{Version}}/{{Product}}-{{Version}}-{{OS}}-{{Arch}}.tgz"
)

func TestCheck(t *testing.T) {
	const godelYML = `exclude:
  names:
    - "\\..+"
    - "vendor"
  paths:
    - "godel"
`

	assetPath, err := products.Bin("errcheck-asset")
	require.NoError(t, err)

	configFiles := map[string]string{
		"godel/config/godel.yml":        godelYML,
		"godel/config/check-plugin.yml": "",
	}

	pluginProvider, err := pluginapitester.NewPluginProviderFromLocator(okgoPluginLocator, okgoPluginResolver)
	require.NoError(t, err)

	okgotester.RunAssetCheckTest(t,
		pluginProvider,
		pluginapitester.NewAssetProvider(assetPath),
		"errcheck",
		"",
		[]okgotester.AssetTestCase{
			{
				Name: "unchecked error",
				Specs: []gofiles.GoFileSpec{
					{
						RelPath: "go.mod",
						Src:     "module foo",
					},
					{
						RelPath: "foo.go",
						Src: `package foo

func Foo() {
	bar()
}

func bar() error {
	return nil
}
`,
					},
				},
				ConfigFiles: configFiles,
				WantError:   true,
				WantOutput: `Running errcheck...
foo.go:4:5: bar()
Finished errcheck
Check(s) produced output: [errcheck]
`,
			},
			{
				Name: "errcheck uses ignore configuration",
				Specs: []gofiles.GoFileSpec{
					{
						RelPath: "go.mod",
						Src:     "module foo",
					},
					{
						RelPath: "foo.go",
						Src: `package foo

import "os"

func Foo() {
	os.Getwd()
}
`,
					},
				},
				ConfigFiles: map[string]string{
					"godel/config/godel.yml": godelYML,
					"godel/config/check-plugin.yml": `
checks:
 errcheck:
   config:
     ignore:
       - os:Getwd
`,
				},
				WantOutput: `Running errcheck...
Finished errcheck
`,
			},
			{
				Name: "errcheck uses exclude configuration",
				Specs: []gofiles.GoFileSpec{
					{
						RelPath: "go.mod",
						Src:     "module foo",
					},
					{
						RelPath: "foo.go",
						Src: `package foo

import "os"

func Foo() {
	os.Getwd()
}
`,
					},
				},
				ConfigFiles: map[string]string{
					"godel/config/godel.yml": godelYML,
					"godel/config/check-plugin.yml": `
checks:
 errcheck:
   config:
     exclude:
       - os.Getwd
`,
				},
				WantOutput: `Running errcheck...
Finished errcheck
`,
			},
			{
				Name: "unchecked error from inner directory",
				Specs: []gofiles.GoFileSpec{
					{
						RelPath: "go.mod",
						Src:     "module foo",
					},
					{
						RelPath: "foo.go",
						Src: `package foo

func Foo() {
	bar()
}

func bar() error {
	return nil
}
`,
					},
					{
						RelPath: "inner/bar",
					},
				},
				ConfigFiles: configFiles,
				Wd:          "inner",
				WantError:   true,
				WantOutput: `Running errcheck...
../foo.go:4:5: bar()
Finished errcheck
Check(s) produced output: [errcheck]
`,
			},
		},
	)
}

func TestUpgradeConfig(t *testing.T) {
	pluginProvider, err := pluginapitester.NewPluginProviderFromLocator(okgoPluginLocator, okgoPluginResolver)
	require.NoError(t, err)

	assetPath, err := products.Bin("errcheck-asset")
	require.NoError(t, err)
	assetProvider := pluginapitester.NewAssetProvider(assetPath)

	pluginapitester.RunUpgradeConfigTest(t,
		pluginProvider,
		[]pluginapitester.AssetProvider{assetProvider},
		[]pluginapitester.UpgradeConfigTestCase{
			{
				Name: `legacy configuration with empty "args" field is updated`,
				ConfigFiles: map[string]string{
					"godel/config/check.yml": `
checks:
  errcheck:
    filters:
      - value: "should have comment or be unexported"
      - type: name
        value: ".*.pb.go"
`,
				},
				Legacy: true,
				WantOutput: `Upgraded configuration for check-plugin.yml
`,
				WantFiles: map[string]string{
					"godel/config/check-plugin.yml": `checks:
  errcheck:
    filters:
    - value: should have comment or be unexported
    exclude:
      names:
      - .*.pb.go
`,
				},
			},
			{
				Name: `legacy configuration with "ignore" args is upgraded`,
				ConfigFiles: map[string]string{
					"godel/config/check.yml": `
checks:
  errcheck:
    args:
      - "-ignore"
      - "github.com/cihub/seelog:(Info|Warn|Error|Critical)f?"
`,
				},
				Legacy: true,
				WantOutput: `Upgraded configuration for check-plugin.yml
`,
				WantFiles: map[string]string{
					"godel/config/check-plugin.yml": `checks:
  errcheck:
    config:
      ignore:
      - github.com/cihub/seelog:(Info|Warn|Error|Critical)f?
`,
				},
			},
			{
				Name: `legacy configuration with unknown args fails`,
				ConfigFiles: map[string]string{
					"godel/config/check.yml": `
checks:
  errcheck:
    args:
      - "-help"
`,
				},
				Legacy:    true,
				WantError: true,
				WantOutput: `Failed to upgrade configuration:
	godel/config/check-plugin.yml: failed to upgrade configuration: failed to upgrade check "errcheck" legacy configuration: failed to upgrade asset configuration: errcheck-asset only supports legacy configuration if the first element in "args" is "-ignore"
`,
				WantFiles: map[string]string{
					"godel/config/check.yml": `
checks:
  errcheck:
    args:
      - "-help"
`,
				},
			},
			{
				Name: `valid v0 config works`,
				ConfigFiles: map[string]string{
					"godel/config/check-plugin.yml": `
checks:
  errcheck:
    config:
      # comment
      ignore:
        - github.com/cihub/seelog:(Info|Warn|Error|Critical)f?
`,
				},
				WantOutput: ``,
				WantFiles: map[string]string{
					"godel/config/check-plugin.yml": `
checks:
  errcheck:
    config:
      # comment
      ignore:
        - github.com/cihub/seelog:(Info|Warn|Error|Critical)f?
`,
				},
			},
		},
	)
}
