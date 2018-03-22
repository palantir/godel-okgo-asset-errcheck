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

package creator

import (
	"fmt"
	"regexp"

	"github.com/palantir/okgo/checker"
	"github.com/palantir/okgo/okgo"
	"gopkg.in/yaml.v2"

	"github.com/palantir/godel-okgo-asset-errcheck/errcheck"
	"github.com/palantir/godel-okgo-asset-errcheck/errcheck/config"
)

var lineRegexp = regexp.MustCompile(`(.+):(\d+):(\d+):\t(.+)`)

func Errcheck() checker.Creator {
	return checker.NewCreator(
		errcheck.TypeName,
		errcheck.Priority,
		func(cfgYML []byte) (okgo.Checker, error) {
			var cfg config.Errcheck
			if err := yaml.UnmarshalStrict(cfgYML, &cfg); err != nil {
				return nil, err
			}

			var args []string
			if len(cfg.Ignore) > 0 {
				args = append(args, "-ignore")
				args = append(args, cfg.Ignore...)
			}
			return checker.NewAmalgomatedChecker(errcheck.TypeName, checker.ParamPriority(errcheck.Priority), checker.ParamArgs(args...),
				checker.ParamLineParserWithWd(
					func(line, wd string) okgo.Issue {
						if match := lineRegexp.FindStringSubmatch(line); match != nil {
							// errcheck uses tab rather than space to separate prefix from content: transform to use space instead
							line = fmt.Sprintf("%s:%s:%s: %s", match[1], match[2], match[3], match[4])
						}
						return okgo.NewIssueFromLine(line, wd)
					},
				)), nil
		},
	)
}
