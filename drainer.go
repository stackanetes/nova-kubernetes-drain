// Copyright 2016 Intel Corporation
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
package main

import (
	"flag"
	"os"

	"github.com/stackanetes/evacuator/kube_watcher"
	"github.com/stackanetes/evacuator/node"
	"github.com/stackanetes/kubernetes-entrypoint/logger"
)

func main() {
	daemon := flag.Bool("daemon", false, "without this run once")
	flag.Parse()

	if !*daemon {
		n, err := node.New()
		if err != nil {
			logger.Error.Printf("Cannot create Node: %v\n", err)
			os.Exit(1)
		}
		err = n.Disable()
		if err != nil {
			logger.Error.Printf("Cannot disable node: %v\n", err)
			os.Exit(1)
		}
	} else {
		kw, err := kubewatcher.New()
		if err != nil {
			logger.Error.Printf("I cannot create eventWatcher: %v", err)
			os.Exit(1)
		}
		err = kw.Watch()
		if err != nil {
			logger.Error.Printf("Error druing watching: %v", err)
			os.Exit(1)
		}
	}
}
