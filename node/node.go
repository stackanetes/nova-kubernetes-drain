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
package node

import (
	"fmt"
	"sync"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/adminactions"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
	"github.com/rackspace/gophercloud/pagination"
	"github.com/stackanetes/kubernetes-entrypoint/util/command"
	"github.com/stackanetes/kubernetes-entrypoint/logger"
	"time"
)

const (
	interval = 2
)

// Node is the implementation of a openstack hypervisor.\
type Node struct {
	client   *gophercloud.ServiceClient
	body     map[string]string
	confPath string
	hostname string
	vms      *[]servers.Server
	Enabled  bool
}

// New creates a Openstack node.
func New(confPath string) (*Node, error) {
	var n Node
	var err error

	n.Enabled = true
	n.confPath = confPath
	n.hostname, err = GetMyHostname()
	if err != nil {
		return &n, fmt.Errorf("Cannot retrieve hostname: %v", err)
	}

	n.body = map[string]string{"binary": "nova-compute", "host": n.hostname}

	n.client, err = createOpenstackClient(n.confPath)
	if err != nil {
		return &n, fmt.Errorf("Cannot create openstack client: %v", err)
	}

	return &n, err
}

// Disable live-migrating all VMs for node and change node state to disable
func (n *Node) Disable() (err error) {
	url := n.client.ServiceURL("os-services", "disable")
	resp, err := n.client.Request("PUT", url, gophercloud.RequestOpts{
		JSONBody: n.body,
		OkCodes:  []int{200, 204},
	})
	if err != nil {
		return fmt.Errorf("Cannot change node state. Recieved code: %s.\nError: %v", resp.StatusCode, err)
	}
	logger.Info.Println("Node disabled.")
	n.Enabled = false

	err = n.migrateVMs()
	if err != nil {
		logger.Warning.Println("Cannot migrate VMs: %v", err)
	}

	return
}

// Enable change node state to enable
func (n *Node) Enable() error {
	url := n.client.ServiceURL("os-services", "enable")
	resp, err := n.client.Request("PUT", url, gophercloud.RequestOpts{
		JSONBody: n.body,
		OkCodes:  []int{200, 204},
	})
	if err != nil {
		logger.Error.Println("Cannot change node state.")
		return fmt.Errorf("Recieved code: %s.\nError: %v", resp.StatusCode, err)
	}
	logger.Info.Println("Node enabled.")
	n.Enabled = true

	return nil
}

func (n *Node) migrateVMs() (err error) {
	var wg sync.WaitGroup
	err = n.updateVMList()
	if err != nil {
		return fmt.Errorf("Cannot update server list: ", err)
	}

	for _, vm := range *n.vms {
		wg.Add(1)

		go func(c *gophercloud.ServiceClient, vmID string) {
			defer wg.Done()
			for a := 1; a < 4; a++ {
				er := adminactions.LiveMigrate(n.client, vm.ID, adminactions.LiveMigrateOpts{
					BlockMigration: true,
				})
				if er.Result.Err == nil {
					logger.Info.Printf("Attempt: %i. VM %s migrated\n", a, vm.ID)
					break
				}
				logger.Warning.Printf("Attempt: %i. Cannot migrate VM %s: %v.\n", a, vm.ID, er.Result.Err)
				time.Sleep(interval * time.Second)
			}

		}(n.client, vm.ID)
	}
	wg.Wait()

	return
}

func (n *Node) updateVMList() (err error) {
	pager := servers.List(n.client, servers.ListOpts{
		Host: n.hostname,
	})
	vms := []servers.Server{}

	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		vms, err = servers.ExtractServers(page)
		if err != nil {
			return false, err
		}
		return true, nil
	})
	n.vms = &vms

	if err != nil {
		return fmt.Errorf("Cannot retrieve server list from Pager: %v", err)
	}
	logger.Info.Printf("Retrive list of %d VMs for this host.\n", len(vms))

	return nil
}
