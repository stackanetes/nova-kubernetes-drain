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
package kubewatcher

import (
	"fmt"

	"github.com/stackanetes/nova-kubernetes-drain/node"
	"github.com/stackanetes/kubernetes-entrypoint/logger"
	"k8s.io/kubernetes/pkg/api"
	cl "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/watch"
)

// EventWatcher is the implementation of kubernetes event watcher
type EventWatcher struct {
	Client    *cl.Client
	MyIP      string
	Hostname  string
	Node      *node.Node
}

// New creates new kubernetes event watcher.
func New(confPath string) (ew EventWatcher, err error) {
	ew.Client, err = cl.NewInCluster()
	if err != nil {
		return ew, fmt.Errorf("Cannot create client because of %v", err)
	}

	ew.MyIP, err = node.GetMyIPAddress()
	if err != nil {
		return ew, fmt.Errorf("Cannot recieve ip address %v", err)
	}

	ew.Hostname, err = node.GetMyHostname()
	if err != nil {
		return ew, fmt.Errorf("Cannot recieve hostname: %v", err)
	}

	ew.Node, err = node.New(confPath)
	if err != nil {
		return ew, fmt.Errorf("Cannot create node object: %v", err)
	}

	logger.Info.Println("EventWatcher successfully created.")
	return
}

// Watch starts listening kubernetes event stream.
// During watching if particular events found, execute proper method.
func (ew EventWatcher) Watch() error {
	var err error
	// Prepare events watcher.
	watcher, err := ew.Client.Nodes().Watch(api.ListOptions{})
	if err != nil {
		return fmt.Errorf("Cannot create watcher over no selector: %v", err)
	}
	logger.Info.Println("Watcher created.")

	for event := range watcher.ResultChan() {
		node, ok := event.Object.(*api.Node)
		if !ok { continue }

		if event.Type != watch.Modified { continue }

		// Check if event belongs to node. Host can use hostname or IP address.
		if node.Name != ew.MyIP && node.Name != ew.Hostname { continue }

		if node.Spec.Unschedulable && ew.Node.Enabled {
			logger.Info.Printf("ew.Node.Enabled: %b", ew.Node.Enabled)
			if err = ew.Node.Disable(); err != nil {
				return err
			}
		} else if !node.Spec.Unschedulable && !ew.Node.Enabled {
			logger.Info.Printf("ew.Node.Enabled: %b", ew.Node.Enabled)
			if err = ew.Node.Enable(); err != nil {
				return err
			}
		}
	}
	return err
}
