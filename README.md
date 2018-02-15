# ORCA
Orca is a (very) simple, provisioning application for bare-metal systems so
that they can be used with Kubernetes/Istio in an IPv6 (only) environment.

The goal is to reduce as many manual steps as possible, so that provisioning
of systems can occur quickly. This is geared to a lab environment, where the
user is using KubeAdm or similar tool to bring up Kubernetes.

A stretch goal is to automate, as much as possible, of the setup of Kubernetes
and Istio.


## A bit about IPv6 and Kubernetes...
Kubernetes 1.9 has alpha support for IPv6 only (not dual stack) mode of
operation for pods and services. There are various plugins that have or are
adding support for IPv6. The reference Bridge plugin, has support and will be
used by Orca.

Currently, there are some external sites, like github.com, which do not support
IPv6 yet. As a result, the Kubernetes installation in 1.9 uses DNS64 and NAT64
to access the outside world. With this solution, a DNS64 and NAT64 server will
be employed via containers, rather than relying on external H/W or S/W.


## How does this all work?
Once the bare-metals systems have met the **prerequisites** shown below, you can
create a configuration file for your topology, and then run Orca commands on
each node to prepare them for running Kubernetes (and in the future, to bring
up a cluster, I hope!)

When done, you can use Orca commands to clean up the systems, effectively
undoing the setup made and restoring the system to original state.


## Prerequisites
The following needs to be done, prior to using this tool:
* One or more bare-metal systems running Linux (tested with Ubuntu 16.04)
  * Two interfaces, one for access to box, one for management network for cluster
  * Make sure management interface doesn't have a conflicting IPv6 address
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

Copy this executable, and the associated configuration YAML file (see next section)
to each system to be provisioned.


## Configuration Setup
Orca is driven by a YAML configuration file that specifies the topology for the
entire cluster, the roles that nodes play, and the values to use for subnets,
CIDRs, and IPs.

You can use the default (config.yaml) or specify the configuration file on the
command line using the `--config` option.

We'll take a look at an example file and disect each section.

### Example
```
plugin: bridge
token: "<provide>"
topology:
  my-master:
    interface: "enp10s0"
    opmodes: "master dns64 nat64"
    id: 2
  a-minion:
    interface: "eth0"
    opmodes: "minion"
    id: 3
support_net:
    subnet: "fd00:10::"
    v4cidr: "172.18.0.0/16"
    size: 64
mgmt_net:
    subnet: "fd00:20::"
    size: 64
pod_net:
    prefix: "fd00:40:0:0"
    size: 80
service_net:
    cidr: "fd00:30::/110"
nat64:
    v4_cidr: "172.18.0.128/25"
    v4_ip: "172.18.0.200"
    ip: "fd00:10::200"
dns64:
    remote_server: "64.102.6.247"
    prefix: "fd00:10:64:ff9b::"
    prefix_size: 96
    ip: "fd00:10::100"
```

### Plugin (plugin)
Currently, the reference Bridge plugin is supported by this script. Looking to add
other plugins.

### Token (token)
KubeAdm uses a bootstrap token for bidirectional trust between nodes. As root, run
the command `kubeadm token generate` and place the output into this entry.
```
token: "7aee33.05f81856d78346bd"
```

### Topology (topology)
This is where you specify each of the systems to be provisioned. Each entry is referred
to by the hostname, and contains three items.

First, the name of the **interface** to be used for the management of the cluster during
operation. I used the second interface on my systems (the first being used to access
the systems for provisioning), and had them on the same VLAN, connected to the same
switch in the lab.
```
    interface: "eth1"
```
Second, an arbitrary ID (**id**) is assigned to each node. Use 2+ as the ID, as this
will be used in IPs and subnets (the app doesn't validate this - yet).
```
    id: 100
```
Third, the operational mode (**opmode**) of the system. This string can have the value
**master** or **minion** (only specify one master per cluster). It can also have the
values **dns64** and **nat64** (again only specify these once).
```
    opmodes: "master dns64 nat64"
```
Currently, the **dns64** and **nat64** settings must be on the same system (will see if
it makes sense to allow them on separate nodes). They can accompany a master or
minion, or can be on a node by themselves.

### Support Network (support_net)
For the NAT64 and DNS64 services, which are running in containers, we need a network
that has both V4 and V6 addresses. This section of the YAML file specifies the IPv6
subnet and size (split out as Orca uses the parts separately), and the IPv4 CIDR:
```
    subnet: "fd00:10::"
    size: 64
    v4cidr: "172.18.0.0/16"
```
The IPv4 subnet should be large enough to contain the V4 subnet that will be created
for NAT64 mapping of V6 to V4 addresses. A /16 net is usually fine (Orca doesn't
validate this dependency, currently).

### Management Network (mgmt_net)
The network that is used by Kubernetes for each cluster node, is called out in this
section. The IPv6 subnet and size are specified.
```
    subnet: "fd00:20::"
    size: 64
```

### Pod Network (pod_net)
A second network that is used by Kubernetes for the pods. This network should be
distint from the support and management networks. Here, we specify all but 16 bits
of the subnet address. During provisioning, Orca will add the node ID to the
address to form distinct subnet on each node.
```
prefix: "fd00:40:0:0"
    size: 80
```
In the example configuration, we would have a pod subnet `fd00:40:0:0:2::/80`
on node `my-master` and `fd00:40:0:0:3::/80` on node `a-minion`.

### Service Network (service_net)
Specify the network CIDR to be used for service pods. This should be a smaller
network than the pod subnet?
```
    cidr: "fd00:30::/110"
```

# NAT64 (nat64)
To be able to reach external sites that only support IPv4, we use NAT64 (which
is combined with Docker's NAT44 capabilities) to translate between external
IPv4 addresses and internal IPv6 addresses. To do this, it needs IPv4 access
to the Internet, and uses NAT44 via Docker to translate from it's IPv4 address
to the public IPv4 address for the host. Tayga (http://www.litech.org/tayga/)
is used for this role and runs as a container on the node with **nat64** specified
as an **opmode**.

Tayga uses a pool of local IPv4 addresses that are mapped to IPv6 address. As
such, the IPv4 pool and IPv4 address of Tayga must be specified. Make sure that
this pool is inside of the **support_net** subnet, mentioned above.
```
    v4_cidr: "172.18.0.128/25"
    v4_ip: "172.18.0.200"
```
Also, specify the IPv6 address of Tayga on the **support_net** IPv6 subnet.
```
ip: "fd00:10::200"
```

# DNS64 (dns64)
A companion to NAT64, the DNS64 container using bind9 will provide synthesized IPv6
addresses for external IPv4 addresses (currently, it does so for all addresses).
The prefix for IPv4 embedded IPv6 addresses is specified, along with the size.
```
    prefix: "fd00:10:64:ff9b::"
    prefix_size: 96
```
An external address with a IPv4 address of `172.217.12.238`, would be encoded as
`fd00:10:64:ff9b::acd9:cee`.

The DNS64 container will forward DNS requests to a remote DNS server, which is
also specified.
```
    remote_server: "64.102.6.247"
```
Lastly, the **support_net** IPv6 address of the bind9 server needs to be specified.
```
ip: "fd00:10::100"
```


## Usage
As mentioned above, you should have Orca and the YAML file on each system to be
provisioned. Since Orca needs to perform privileged operations, you'll need to run this
as root:
```
   sudo ~/go/bin/orca [options] {prepare|up|down|clean}
```

Currently, the `prepare` and `clean` commands are implemented. The former will do all
the actions to prepare a node to run Kubernetes, and the later will undo that setup.

In the future, the `up` and `down` commands will support bringing up the Kubernetes
cluster. Will consider commands for Isto support as well.

### Command Line Options
```
Usage: orca [options] {prepare|up|down|clean}
  -alsologtostderr
        log to standard error as well as files
  -config string
        Configurations for orca (default "config.yaml")
  -host string
        Name of (this) host to apply command (default "my-master")
  -log_backtrace_at value
        when logging hits line file:N, emit a stack trace
  -log_dir string
        If non-empty, write log files in this directory
  -logtostderr
        log to standard error instead of files
  -stderrthreshold value
        logs at or above this threshold go to stderr
  -v value
        log level for V logs
  -vmodule value
        comma-separated list of pattern=N settings for file-filtered logging
```

There are log level "1" and "4" entries in the app, if you want verbose logging.
The log defaults to stderr.

The default hostname is the name of the system you are on.


## Under The Covers
Here are the configuration steps that Orca does for the `prepare` command:
* Creates support network with IPv6 and IPv4.
* Starts DNS64 container, with config file, removes IPv4 address, and adds route to NAT64 server.
* Starts NAT64 container.
* Adds IPv4 route to NAT64 server on node.
* Adds management network IP on specified interface.
* Adds route to DNS64 synthesized network via NAT64 server (based on node).
* Places management network IP in /etc/hosts, for this hostname.
* Adds DNS64 support network IP as first nameserver in /etc/resolv.conf.
* Creates a drop-in file for kubelet to specify IPv6 DNS64 server IP.
* Creates CNI config file for bridge plugin (move to "up" step later).

Here are the actions that Orca does for the `clean` command:
* Removes drop-in file for kubelet.
* Removes IP from management interface.
* Removes route for DNS64 synthesized network.
* Restores /etc/hosts.
* Restores /etc/resolv.conf.
* Removes bridge plugin's CNI config file.
* Stops and removes DNS64 container.
* Stops and removes NAT64 container.
* Removes IPv4 route to NAT64 server.
* Removes support network.


## Limitations/Restrictions
* Some newer versions of docker break the enabling of IPv6 in the containers used for DNS64 and NAT64.
* Relies on the tayga and bind6 containers (as provided by other developers).


## Troubleshooting
This section has some notes on issues seen and resolutions (if any).

Tip: If for some reason the `prepare` fails after updating /etc/resolv.conf or /etc/hosts, you can
recover the originals from the .bak files created. However, the tool is designed to allow multiple
invocations, so this should not be required.

I did have one case where kube-dns was not coming up, and kube-proxy log was showing iptables restore
errors saying "iptables-restore v1.6.0: invalid mask `128' specified". This should be using the
ip6tables-restore operation. Was unable to find root cause, but did KubeAdm reset, `clean` command,
flush iptables rules, and rebooted, and problem was cleared. May have been corruption of iptables
rules.


## TODOs/Futures
### Implementation
* Implement `up` and `down` commands (in progress)
  * kubeadm reset/init
* Enhance validation
  * Ensure IP addresses, subnets, and CIDRs are valid.
  * No overlap on pod, management, and support networks.
  * Make sure pod network prefix and size are compatible (prefix should be size - 16 bits).
  * Ensure NAT64 IP is within NAT64 subnet, and that NAT64 subnet is with support subnet.
  * Node IDs > 0. >1?
  * Docker version.
  * Kubeadm, kubectl, kubelet version 1.9+.
  * Go version.
  * Other tools?
* Support Calico plugin. Cillium? Others?  
* Mocking for UTs to provide better coverage.
* Add version command.
* Add per function documentation.
* Mention on my blog.
* **Need to rename app, so as to not conflict with other project names (e.g. spinnaker/orca).**

### Details to figure out
* Decide how to handle prepare failures (exits currently). Rollback? Difficulty?
* Create makefile for building/installing. Build executable for immediate use?
* Modifying NAT64/DNS64 to support external sytems that support IPv6 only addresses, without translating.
* Is there a way to check if management interface already has an (incompatible) IPv6 address?

### Enhancements to consider
* Do Istio startup. Useful?  Metal LB startup?
* Running DNS64 and NAT64 on separate nodes. Useful? Routing?
* Is it useful to try with with IPv4 addresses (only) as a vanilla provisioner.
* Support hypervisors other than Docker (have separated out the code)?
* Allow configuration file to be specified as URL?
* In config file use CIDRs for subnets and split them out to use the parts in the app, instead of having the user specify a subnet and a size separately.
* Consider using Kubeadm's DynamicKubeletConfig, instead of drop-in file for kubelet.

