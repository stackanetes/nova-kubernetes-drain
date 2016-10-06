package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/stackanetes/kubernetes-entrypoint/logger"
	"github.com/stackanetes/nova-kubernetes-drain/kube_watcher"
	"github.com/stackanetes/nova-kubernetes-drain/nova"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/watch"
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

	hostname, err := os.Hostname()
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
	logger.Info.Println("Node Event found.")
	if event != watch.Modified { return nil }

	// Check if event belongs to node. Host can use hostname or IP address.
	if node.Name != d.myIP && node.Name != d.hostname { return nil }
	logger.Info.Println("Event belongs to node.")

	// Refresh state
	if err = d.hypervisor.RefreshState(); err != nil {
		return fmt.Errorf("Cannot update hypervisior state: %v", err)
	}

	if node.Spec.Unschedulable && d.hypervisor.Enabled {
		logger.Info.Println("Disabling hypervisor.")
		if err = d.hypervisor.Disable(); err != nil {
			return err
		}
		if err = d.hypervisor.MigrateVMs(); err != nil {
			logger.Warning.Println("Cannot migrate VMs: %v", err)
		}
	} else if !node.Spec.Unschedulable && !d.hypervisor.Enabled {
		logger.Info.Println("Enabling hypervisor.")
		if err = d.hypervisor.Enable(); err != nil {
			return err
		}
	} else {
		logger.Info.Println("Hypervisior is in a suitable state.")
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
	runOnce := flag.Bool("run-once", false, "run once")
	timeOut := flag.Int("time-out", 30, "time out for a live-migration")
	configPath := flag.String("config-path", "config.yaml", "path to configuration file")
	flag.Parse()

	hyper, err := nova.New(*configPath, *timeOut)
	if err != nil {
		logger.Error.Printf("Cannot create Hypervisor: %v\n", err)
		os.Exit(1)
	}


	if *runOnce && *daemon {
		logger.Warning.Printf("Both '-daemon' and '-run-once' flags passed. You need to pass only one of them.")
		os.Exit(1)
	} else if *runOnce {
		if err = hyper.Disable(); err != nil {
			logger.Error.Printf("Cannot disable node: %v\n", err)
			os.Exit(1)
		}
	} else if *daemon{
		d, err := newDrainer(hyper)
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
			logger.Error.Printf("Error during watching: %v", err)
			os.Exit(1)
		}
	} else {
		logger.Warning.Printf("You need to pass '-daemon' or '-run-once' flag.")
		os.Exit(1)
	}
}
