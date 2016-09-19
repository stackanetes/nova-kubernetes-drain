# nova-kubernetes-drain

[![Build Status](https://api.travis-ci.org/stackanetes/nova-kubernetes-drain.svg?branch=master "Build Status")](https://travis-ci.org/stackanetes/nova-kubernetes-drain)
[![Container Repository on Quay](https://quay.io/repository/stackanetes/nova-kubernetes-drain/status "Container Repository on Quay")](https://quay.io/repository/stackanetes/nova-kubernetes-drain)
[![Go Report Card](https://goreportcard.com/badge/stackanetes/nova-kubernetes-drain "Go Report Card")](https://goreportcard.com/report/stackanetes/nova-kubernetes-drain)

The main goal of Nova-kubernetes-drain is to perform evacuation of [Openstack] compute node when [Kubernetes] node is being drained.
Nova-kubernetes-drain is one of container of the compute-node pod deployed via [Stackanetes].

Nova-kubernetes-drain can be run as a daemon or can perform single evacuation. Those two modes are simple configure by command line flag.

## Requirements

Nova-kubernetes-drain based on two clients:
  1. [Kubernetes] client: https://godoc.org/k8s.io/kubernetes/pkg/client/unversioned
  2. [Openstack] client*: http://gophercloud.io/docs/compute/

  \* Rackscale client is deprecated, but new [client] currently does not support [live-migration]. Client should be switched when new client supports [live-migration]. 

## Configuration file

Nova-kubernetes-drain requires config.yaml. Configuration file should contain all variables necessary to establish connection with openstack.
config.yaml example:

```
IdentityEndpoint: "http://keystone-api:35357/v3/"
Username: "admin"
Password: "mysupersecretpassword"
TenantName: "admin"
DomainID: "default"
```

[live-migration]: http://docs.openstack.org/admin-guide/compute-live-migration-usage.html
[Openstack]: https://www.openstack.org/
[Stackanetes]: https://github.com/stackanetes/stackanetes
[Kubernetes]: http://kubernetes.io/
[uncordon]: http://kubernetes.io/docs/user-guide/kubectl/kubectl_uncordon/
[drain]: http://kubernetes.io/docs/user-guide/kubectl/kubectl_drain/
[client]: https://github.com/gophercloud/gophercloud

## Run once

Run once is a default mode. So to performe a single evacuation you need to just run application without any additional flag. When evacuation is successful the application will exit without any error code. 

To run it:

`./drainer`

The application will execute following actions:
 1. Load [Openstack] Authorisation variables from file.
 2. Determine nova-compute name in [Openstack] of the pod.
 3. Disable nova-compute in [Openstack].
 4. Identify all VMs on this nova-compute node.
 5. Trigger [live-migration] for each of those VMs.
 6. Exit application.

## Daemon

To run Nova-kubernetes-drain as a daemon. One have to pass additional `-daemon` flag.

`./drainer -daemon`

In this mode specific kubernetes events are triggering openstack actions.
[Kubernetes] drain node command will disable [Openstack] compute-node and perform evacuation of VMs.
[Kubernetes] uncrodon node command will enable [Openstack] compute-node.

Lifecycle of the application:
 1. Load [Openstack] Authorisation variables from file.
 2. Hook to [Kubernetes] event stream and wait for proper events.
 3. According to event message, trigger proper command:
    1. [Drain] event received:
       1. Disable nova-compute in [Openstack].
       2. Identify all VMs on this nova-compute node.
       3. Trigger [live-migration] for each of those VMs.
    2. [Uncordon] event received:
       1. Enable nova-compute in [Openstack].
