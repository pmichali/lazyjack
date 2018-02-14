# ORCA

Orca is a (very) simple, provisioning application for bare-metal systems so
that they can be used with Kubernetes/Istio in an IPv6 (only) environment.

The goal is to reduce as many manual steps as possible, so that provisioning
of systems can occur quickly. This is geared to a lab environment, where the
user is using KubeAdm or similar tool to bring up Kubernetes.

A stretch goal is to automate, as much as possible, of the setup of Kubernetes
and Istio.


## A bit about IPv6...

Kubernetes 1.9 has alpha support for IPv6 only (not dual stack) mode of
operation for pods and services. There are various plugins that have or are
adding support for IPv6. The reference Bridge plugin, has support and will be
used by Orca.

Currently, there are some external sites, like github.com, which do not support
IPv6 yet. As a result, the Kubernetes installation in 1.9 uses DNS64 and NAT64
to access the outside world. With this solution, a DNS64 and NAT64 server will
be employed via containers, rather than relying on external H/W or S/W.


## How does this all work?

Once the bare-metals systems have met the *prerequisites* shown below, you can
create a configuration file for your topology, and then run Orca commands on
each node to prepare them for running Kubernetes (and in the future, to bring
up a cluster, I hope!)

When done, you can use Orca commands to clean up the systems, effectively
undoing the setup made and restoring the system to original state.


## Prerequisites

The following needs to be done, prior to using this tool:
* One or more bare-metal systems running Linux (tested with Ubuntu 16.04)
  * Two interfaces, one for access to box, one for management network for cluster
  * Internet access via IPv4 on the node being used for DNS64/NAT64
  * Docker (17.03.2) installed.
  * Version 1.9+ of kubeadm, kubectl (on master), and kubelet.
  * Go 1.9+ installed on the system and environment set up.
* Obtain Orca and build (see below)


## Preparing Orca
Use the following command to obtain Orca and place it into your $GOPATH

```
go get github.com/pmichali/orca
cd $GOPATH/src/github.com/pmichali/orca
```

Build the code into an executable (yeah, I need to figure this out), and place
where desired for easy access. For example:

```
go build -o ~/go/bin/orca cmd/orca.go
```


## Usage
Since Orca needs to perform privileged operations, you'll need to run this
as root (e.g. `sudo ~/go/bin/orca ...`).

TBD


## Limitations/Restrictions

TBD


## TODOs/Futures

TBD
