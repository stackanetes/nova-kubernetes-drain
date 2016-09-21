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

	"github.com/stackanetes/nova-kubernetes-drain/kube_watcher"
	"github.com/stackanetes/nova-kubernetes-drain/nova"
	"github.com/stackanetes/kubernetes-entrypoint/logger"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/watch"
	"fmt"
)

type drainer struct {
	hypervisor *nova.Hypervisor
	myIP       string
	hostname   string
}

func newDrainer(hyper *nova.Hypervisor) (drainer, error){
	ip, err := nova.GetMyIPAddress()
	if err != nil {
		return drainer{}, err
	}

	hostname, err := nova.GetMyHostname()
	if err != nil {
		return drainer{}, err
	}

	return drainer{
		hypervisor: hyper,
		myIP:       ip,
		hostname:   hostname,
	}, nil
}


func (d drainer) RunNode(node *api.Node, event watch.EventType) (err error) {
	if event != watch.Modified { return nil }

	// Check if event belongs to node. Host can use hostname or IP address.
	if node.Name != d.myIP && node.Name != d.hostname { return nil }

	if node.Spec.Unschedulable {
		if err = d.hypervisor.Disable(); err != nil {
			return err
		}
	} else if !node.Spec.Unschedulable {
		if err = d.hypervisor.Enable(); err != nil {
			return err
		}
	}
	return nil
}

func (d drainer) RunReplicationController(*api.ReplicationController, watch.EventType) error{
	return nil
}

func (d drainer) RunService(*api.Service, watch.EventType) error{
	return nil
}

func (d drainer) RunPod(*api.Pod, watch.EventType) error{
	return nil
}

func main() {
	daemon := flag.Bool("daemon", false, "run as a daemon")
	configPath := flag.String("config-path", "config.yaml", "path to configuration file")
	flag.Parse()
	hyper, err := nova.New(*configPath)
	if err != nil {
		logger.Error.Printf("Cannot create Hypervisor: %v\n", err)
		os.Exit(1)
	}

	if !*daemon {
		if err = hyper.Disable(); err != nil {
			logger.Error.Printf("Cannot disable node: %v\n", err)
			os.Exit(1)
		}
	} else {
		d, err := newDrainer(hyper)
		fmt.Printf("d: %v\n", d)
		if err != nil {
			logger.Error.Printf("I cannot create drainer: %v", err)
			os.Exit(1)
		}

		kw, err := kubewatcher.New()
		if err != nil {
			logger.Error.Printf("I cannot create eventWatcher: %v", err)
			os.Exit(1)
		}

		if err = kw.Watch(d); err != nil {
			logger.Error.Printf("Error druing watching: %v", err)
			os.Exit(1)
		}
	}
}
