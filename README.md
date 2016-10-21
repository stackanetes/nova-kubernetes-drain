# nova-kubernetes-drain

[![Build Status](https://api.travis-ci.org/stackanetes/nova-kubernetes-drain.svg?branch=master "Build Status")](https://travis-ci.org/stackanetes/nova-kubernetes-drain)
[![Container Repository on Quay](https://quay.io/repository/stackanetes/nova-kubernetes-drain/status "Container Repository on Quay")](https://quay.io/repository/stackanetes/stackanetes-nova-drain)
[![Go Report Card](https://goreportcard.com/badge/stackanetes/nova-kubernetes-drain "Go Report Card")](https://goreportcard.com/report/stackanetes/nova-kubernetes-drain)

The main goal of Nova-kubernetes-drain is to perform live-evacuation of [OpenStack] compute node when [Kubernetes] node is being drained.
Nova-kubernetes-drain should be deployed as a [Daemonset] via [Stackanetes].

Nova-kubernetes-drain can be run as a daemon or as one-off task. Those two modes are simple configure by command line flag.

## Requirements

Nova-kubernetes-drain is based on the following clients:
  1. [Kubernetes] client: https://godoc.org/k8s.io/kubernetes/pkg/client/unversioned
  2. [OpenStack] client*: http://gophercloud.io/docs/compute/

  \* Rackspace client is deprecated, but the new [client] currently does not support [live-migration]. The Client should be switched when the [client] supports [live-migration]. 

## Configuration file

Nova-kubernetes-drain requires a configuration file, by default named config.yaml. Configuration file should contain all variables necessary to establish connection with openstack.
config.yaml example:

```
IdentityEndpoint: "http://keystone-api:5000/v3/"
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
[daemonset]: http://kubernetes.io/docs/admin/daemons/

## Run once

Run once is the default mode. To perform an evacuation, simply run the application without any additional flags. Flag `-config-path` is optional. Once the evacuation is successful, the application will exit without any error code. 

To run it:

`./drainer -config-path=<configuration-file-name>`

The application will execute following actions:
 1. Get authorization data from configuration file.
 2. Determine the name of the running hypervisor in [OpenStack].
 3. Disable scheduling of VMs on this node in [OpenStack].
 4. Identify all VMs on this node.
 5. Trigger a [live-migration].
 6. Exit the application if all VMs are migrated or timeout is reached.

## Daemon

To run Nova-kubernetes-drain as a daemon. One has to pass additional `-daemon` flag.

`./drainer -daemon -config-path=<configuration-file-name>`

In this mode, the application will wait for specific Kubernetes events to take actions.
A [Kubernetes] drain operation will disable scheduling of new VMs and perform live-evacuation of the currently running VMs. On the other hand, a [Kubernetes] uncordon operation will re-enable the scheduling.

Lifecycle of the application:
 1. Load [Openstack] authorization variables from file.
 2. Hook to [Kubernetes] event stream and wait for proper events.
 3. According to the event message, trigger the appropriate operation:
    1. Unschedulable event received:
       1. Disable nova-compute in [Openstack].
       2. Identify all VMs on this nova-compute node.
       3. Trigger [live-migration] for each of those VMs.
    2. Schedulable event received:
       1. Enable nova-compute in [Openstack].
