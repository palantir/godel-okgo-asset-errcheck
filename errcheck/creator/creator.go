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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/palantir/godel-okgo-asset-errcheck/errcheck"
	"github.com/palantir/godel-okgo-asset-errcheck/errcheck/config"
	"github.com/palantir/okgo/checker"
	"github.com/palantir/okgo/okgo"
	"github.com/palantir/pkg/signals"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
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
			var postAction func()

			if len(cfg.Ignore) > 0 {
				args = append(args, "-ignore", strings.Join(cfg.Ignore, ","))
			}

			if len(cfg.Exclude) > 0 {
				excludeFile, err := createExcludeFile(cfg.Exclude)
				if err != nil {
					if excludeFile != "" {
						_ = os.Remove(excludeFile)
					}
					return nil, errors.Wrap(err, "failed to create exclude file")
				}

				ctx, cancel := signals.ContextWithShutdown(context.Background())
				done := make(chan struct{})

				postAction = func() {
					cancel()
					<-done
				}

				go func() {
					<-ctx.Done()
					_ = os.Remove(excludeFile)
					done <- struct{}{}
				}()

				args = append(args, "-exclude", excludeFile)
			}

			c := checker.NewAmalgomatedChecker(errcheck.TypeName, checker.ParamPriority(errcheck.Priority), checker.ParamArgs(args...),
				checker.ParamLineParserWithWd(
					func(line, wd string) okgo.Issue {
						if match := lineRegexp.FindStringSubmatch(line); match != nil {
							// errcheck uses tab rather than space to separate prefix from content: transform to use space instead
							line = fmt.Sprintf("%s:%s:%s: %s", match[1], match[2], match[3], match[4])
						}
						return okgo.NewIssueFromLine(line, wd)
					},
				),
			)
			return &postActionChecker{c, postAction}, nil
		},
	)
}

type postActionChecker struct {
	okgo.Checker
	action func()
}

func (c *postActionChecker) Check(pkgPaths []string, projectDir string, stdout io.Writer) {
	if c.action != nil {
		defer c.action()
	}
	c.Checker.Check(pkgPaths, projectDir, stdout)
}

func createExcludeFile(excludes []string) (file string, err error) {
	tmp, err := ioutil.TempFile("", "")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temporary file")
	}

	defer func() {
		if cerr := tmp.Close(); cerr != nil && err == nil {
			err = errors.Wrap(cerr, "failed to close file")
		}
	}()

	if _, err := tmp.WriteString(strings.Join(excludes, "\n")); err != nil {
		return tmp.Name(), errors.Wrap(err, "failed to write excludes to file")
	}
	return tmp.Name(), nil
}
