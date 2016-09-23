package nova

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/adminactions"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
	"github.com/rackspace/gophercloud/pagination"
	"github.com/stackanetes/kubernetes-entrypoint/logger"
)

const (
	retryInterval = 2
	novaComputeBinaryName = "nova-compute"
	enabledString = "enabled"
	liveMigrationRetry = 3
)

// Service is a struct which represents single Openstack service
type Service struct {
	Status          string
	Binary          string
	Host            string
	Zone            string
	State           string
	Disabled_reason string
	Id              int
}

// NovaService is struct which represents Nova services returned by OpenStack API
type NovaService struct {
	Services []Service
}

// Node is an implementation of OpenStack hypervisor.
type Hypervisor struct {
	body     map[string]string
	client   *gophercloud.ServiceClient
	confPath string
	hostname string
	timeOut  time.Duration
	vms      *[]servers.Server
	Enabled  bool
}


// NovaServer is struct which represents Nova server returned by OpenStack API
type NovaServer struct {
	Server servers.Server
}

// New is a constructor for Hypervisor.
func New(confPath string, timeOut int) (*Hypervisor, error) {
	to := time.Duration(timeOut) * time.Minute
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("Cannot retrieve hostname: %v", err)
	}
	client, err := createOpenstackClient(confPath)
	if err != nil {
		return nil, fmt.Errorf("Cannot create openstack client: %v", err)
	}
	return &Hypervisor{
		body: map[string]string{
			"binary": "nova-compute",
			"host": hostname,
		},
		client:  client,
		confPath: confPath,
		hostname: hostname,
		timeOut: to,
		Enabled: true,
	}, nil
}

func (n *Hypervisor) novaServices() ([]Service, error) {
	nova := new(NovaService)
	url := n.client.ServiceURL("os-services")
	resp, err := n.client.Request("GET", url, gophercloud.RequestOpts{
		OkCodes: []int{200, 204},
	})
	if err != nil {
		return nil, fmt.Errorf("Cannot gather openstack-service list: %v", err)
	}

	if err = getJson(resp.Body, &nova); err != nil {
		err = fmt.Errorf("Cannot decode JSON: %v", err)
	}

	return nova.Services, err
}
func (n *Hypervisor) hypervisorStatus() (bool, error) {
	services, err := n.novaServices()
	if err != nil {
		return false, fmt.Errorf("Cannot create obtain nova-compute services: %v, err")
	}
	for _, service := range services {
		if service.Host == n.hostname && service.Binary == novaComputeBinaryName {
			if service.Status == enabledString {
				return true, nil
			}
			return false, nil
		}
	}
	return false, fmt.Errorf("Cannot find nova-service with hostname: %s", n.hostname)
}

func (n *Hypervisor) RefreshState() (err error) {
	status, err:= n.hypervisorStatus()
	if err != nil {
		return fmt.Errorf("Cannot update hypervisor state: %v", err)
	}
	if status != n.Enabled {
		logger.Info.Printf("Hypervisior status updated. New status = %v", status)
		n.Enabled = status
	}
	return
}

// Disable disable node and scheduling on it.
func (n *Hypervisor) Disable() (err error) {
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

	return
}

// Enable change node state to enable
func (n *Hypervisor) Enable() error {
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

func (n *Hypervisor) isMigrated(vmID string, hostID string) (bool, error) {
	vm := new(NovaServer)
	url := n.client.ServiceURL("servers", vmID)
	resp, err := n.client.Request("GET", url, gophercloud.RequestOpts{
		OkCodes:  []int{200, 204},
	})
	if err != nil {
		return false, fmt.Errorf("Cannot gather server %v information: %v", vmID, err)
	}

	if err = getJson(resp.Body, &vm); err != nil {
		return false, fmt.Errorf("Cannot decode JSON: %v", err)
	}

	if vm.Server.HostID != hostID {
		return true, nil
	}

	return false, nil
}

// MigrateVMs live migrate all VMs out of node
func (h *Hypervisor) MigrateVMs() (err error) {
	var wg sync.WaitGroup
	if err = h.updateVMList(); err != nil {
		return fmt.Errorf("Cannot update server list: ", err)
	}

	for _, vm := range *h.vms {
		wg.Add(1)

		go func(vmID string, hostID string) {
			migrated := true
			defer wg.Done()
			for a := 1; a < liveMigrationRetry + 1; a++ {
				er := adminactions.LiveMigrate(h.client, vmID, adminactions.LiveMigrateOpts{
					BlockMigration: true,
				})
				if er.Result.Err == nil {
					logger.Info.Printf("Attempt: %d. Request to migrate VM %s accepted\n", a, vmID)
					migrated = false
					break
				}
				logger.Warning.Printf("Attempt: %d. Cannot run migratation of VM %s: %v.\n", a, vmID, er.Result.Err)
				time.Sleep(time.Duration(a * 10) * time.Second)
			}
			for counter := 0; !migrated ; counter++ {
				migrated, err = h.isMigrated(vmID, hostID)
				if err != nil {
					logger.Warning.Printf("Cannot update VM: %v status.", vmID)
				}
				if migrated {
					logger.Info.Printf("VM: %v has been migrated.", vmID)
				} else {
					logger.Info.Printf("VM: %v has not been migrated.", vmID)
					time.Sleep(time.Duration(counter * 10) * time.Second)
				}
			}
			if !migrated {
				logger.Info.Printf("Cannot migrate VM: %v.", vmID)
			}
		}(vm.ID, vm.HostID)
	}

	if waitTimeout(&wg, h.timeOut) {
		logger.Warning.Println("Time out waiting for live-migration.")
	} else {
		logger.Warning.Println("All VMs migrated")
	}

	return
}

func (n *Hypervisor) updateVMList() (err error) {
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
	if err != nil {
		return fmt.Errorf("Cannot retrieve server list from Pager: %v", err)
	}

	logger.Info.Printf("Retrive list of %d VMs for this host.\n", len(vms))
	n.vms = &vms

	return nil
}
