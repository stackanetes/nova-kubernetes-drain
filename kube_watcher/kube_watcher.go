package kubewatcher

import (
	"fmt"

	"github.com/stackanetes/kubernetes-entrypoint/logger"
	"k8s.io/kubernetes/pkg/api"
	cl "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/watch"
)

// EventWatcher is the implementation of kubernetes event watcher
type EventWatcher struct {
	Client    *cl.Client
}

type Selector interface {
	RunNode(*api.Node, watch.EventType) error
	RunPod(*api.Pod, watch.EventType) error
	RunReplicationController(*api.ReplicationController, watch.EventType) error
	RunService(*api.Service, watch.EventType) error
}

// New creates new kubernetes event watcher.
func New() (ew EventWatcher, err error) {
	ew.Client, err = cl.NewInCluster()
	if err != nil {
		return ew, fmt.Errorf("Cannot create client because of %v", err)
	}
	logger.Info.Println("EventWatcher successfully created.")

	return
}

// Watch starts listening kubernetes event stream.
// During watching if particular events found, execute proper method.
func (ew EventWatcher) Watch(sel Selector) error {
	var err error
	// Prepare events watcher.
	watcher, err := ew.Client.Nodes().Watch(api.ListOptions{})
	if err != nil {
		return fmt.Errorf("Cannot create watcher over no selector: %v", err)
	}
	logger.Info.Println("Watcher created.")

	for event := range watcher.ResultChan() {

		n, ok := event.Object.(*api.Node)
		if ok {
			err = sel.RunNode(n, event.Type)
			if err != nil {
				return err
			}
			continue
		}

		p, ok := event.Object.(*api.Pod)
		if ok {
			err = sel.RunPod(p, event.Type)
			if err != nil {
				return err
			}
			continue
		}

		rc, ok := event.Object.(*api.ReplicationController)
		if ok {
			err = sel.RunReplicationController(rc, event.Type)
			if err != nil {
				return err
			}
			continue
		}

		s, ok := event.Object.(*api.Service)
		if ok {
			err = sel.RunService(s, event.Type)
			if err != nil {
				return err
			}
			continue
		}
	}

	return err
}
