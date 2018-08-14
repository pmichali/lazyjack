# Lazyjack
Lazyjacks are rigging used to assist in sail handling during reefing and
furling, making the process easier.

In keeping with the nautical theme of Kubernetes, the lazyjack application
is used to make it easier to provision bare-metal systems that they can be
used with Kubernetes/Istio in an IPv6 (only) environment.

The goal is to reduce as many manual steps as possible, so that provisioning
of systems can occur quickly. This is geared to a lab environment, where the
user is using KubeAdm or similar tool to bring up Kubernetes.

## Quick Start Guide
For the impatient, you can do the following to bring up cluster of two (or more)
nodes. See below for details on each step.

1. Provision the hardware with OS and the pre-requisite tools.
2. Install Lazyjack on all systems.
3. Modify the sample config file using the hosts/interfaces for your topology.
4. Run `init` command on master.
5. Copy updated config file to minions.
6. Run `prepare` on each node.
7. Run `up` on master, and then on minions.


## A bit about IPv6 and Kubernetes...
Starting with Kubernetes 1.9, there is alpha support for IPv6 only (not dual stack) mode of
operation for pods and services. There are various plugins that have or are
adding support for IPv6. The reference Bridge plugin, has support and will be
used by Lazyjack.

Currently, there are some external sites, like github.com, which do not support
IPv6 yet. As a result, the Kubernetes installation in 1.9+ uses DNS64 and NAT64
to access the outside world. With this solution, a DNS64 and NAT64 server will
be employed via containers, rather than relying on external H/W or S/W.


## How does this all work?
Once the bare-metals systems have met the **prerequisites** shown below, you can
create a configuration file for your topology, and then run Lazyjack commands on
each node to prepare and run Kubernetes.

When done, you can use Lazyjack commands to clean up the systems, effectively
undoing the setup made and restoring the system to original state.


## Prerequisites
The following needs to be done, prior to using this tool:
* One or more bare-metal systems running Linux (tested with Ubuntu 16.04), each with:
  * Two interfaces, one for access to box, one for management network for cluster.
  * Make sure management interface doesn't have a conflicting IPv6 address.
  * Internet access via IPv4 on the node being used for DNS64/NAT64.
  * Docker (17.03.2) installed.
  * Version 1.9+ of kubeadm, kubectl (on master), and kubelet.
  * Go 1.9+ (1.10.3+ for Kuberentes 1.11+) installed on the system and environment set up.
  * CNI 0.7.1+ installed.
  * openssl installed on system (I used 1.0.2g).
  * (optional) Internet access via IPv6 for direct IPv6 access to external sites.
    * IPv6 enabled on node.
    * IPv6 address on main interface with Internet connectivity.
    * Default route for IPv6 traffic using main interface of nodes.
    * Setting sysctl accept_ra=2 on main I/F (e.g. `net.ipv6.conf.eth0.accept_ra = 2`) of nodes.
* Install Lazyjack on each system (see below)


## Preparing Lazyjack
The easiest way to install lazyjack is to pull down the latest release. For example:
```
mkdir ~/bare-metal
cd ~/bare-metal
wget https://github.com/pmichali/lazyjack/releases/download/v1.0.0/lazyjack_1.0.0_linux_amd64.tar.gz
tar -xzf lazyjack_1.0.0_linux_amd64.tar.gz
sudo cp lazyjack /usr/local/bin
```

This will provide the executable and a sample configuration YAML file.

Alternately, you can get the code:
```
go get github.com/pmichali/lazyjack
cd $GOPATH/src/github.com/pmichali/lazyjack
```

Build an executable, and place it where desired for easy access. For example:

```
go build cmd/lazyjack.go
sudo cp lazyjack /usr/local/bin/
```

Copy this executable, and the associated configuration YAML file (see next section)
to each system to be provisioned.


## Configuration Setup
Lazyjack is driven by a YAML configuration file that specifies the topology for the
entire cluster, the roles that nodes play, and the values to use for subnets,
CIDRs, and IPs.

You can use the default (config.yaml) or specify the configuration file on the
command line using the `--config` option.

We'll take a look at an example file and disect each section.

### Example
```
general:
    token: "<provide-token>"
    token-cert-hash: "<provide-cert-hash>"
    plugin: bridge
    work-area: "/tmp/lazyjack"
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
    cidr: "fd00:10::/64"
    v4cidr: "172.18.0.0/16"
mgmt_net:
    cidr: "fd00:20::/64"
pod_net:
    cidr: "fd00:40::/72"
    mtu: 9000
service_net:
    cidr: "fd00:30::/110"
nat64:
    v4_cidr: "172.18.0.128/25"
    v4_ip: "172.18.0.200"
    ip: "fd00:10::200"
dns64:
    remote_server: "64.102.6.247"
    cidr: "fd00:10:64:ff9b::/96"
    ip: "fd00:10::100"
    allow_aaaa_use: true
```

### Token (token) and Token CA Certificate Hash (token-cert-hash)
KubeAdm uses a token and CA certificate for nodes to communicate. These two
fields are filled out automatically by the `init` command, which needs to
be run on the master node, before copying the configuration file over to
minion nodes for use in the `up` command. You don't need to set these.

### Plugin (plugin)
Lazyjack will support both the Bridge and PTP plugins. Use either "bridge",
or "ptp", respectively.

### Work Area (work-area)
By default, the `/tmp/lazyjack` area is used to place configuration files,
certificates, etc. that are used by `lazyjack`. For security purposes, you
should select a secure location, by overriding the value in this field.

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
CIDR, and the IPv4 CIDR:
```
    cidr: "fd00:10::/64"
    v4cidr: "172.18.0.0/16"
```
The IPv4 subnet should be large enough to contain the V4 subnet that will be created
for NAT64 mapping of V6 to V4 addresses. A /16 net is usually fine (Lazyjack doesn't
validate this requirement, currently).

### Management Network (mgmt_net)
The network that is used by Kubernetes for each cluster node, is called out in this
section. 
```
    cidr: "fd00:20::/64"
```

### Pod Network (pod_net)
A second network that is used by Kubernetes for the pods. This network should be
distint from the support and management networks. Here, we specify the CIDR for
the pod cluster. Inside this network, each node will carve out a subnet, using the
node "ID" as the last part of the address.
```
    cidr: "fd00:40::/72"
```
In the example configuration, we would have a pod subnet `fd00:40:0:0:2::/80`
on node `my-master` and `fd00:40:0:0:3::/80` on node `a-minion`. If you want to
specify the prefix and size, the prefix must be fully qualified (e.g. in this
case `fd00:40:0:0:`) and the size must be the size allocated to the node (e.g.
80).

Optionally, you can set the MTU used on the interface for the pod and management
networks, on each node. Use the following, under the pod_net section.
```
    mtu: 9000
```

### Service Network (service_net)
Specify the network CIDR to be used for service pods. This should be a smaller
network than the pod subnet?
```
    cidr: "fd00:30::/110"
```

### NAT64 (nat64)
To be able to reach external sites that only support IPv4, we use NAT64 (which
is combined with Docker's NAT44 capabilities) to translate between external
IPv4 addresses and internal IPv6 addresses. To do this, it needs IPv4 access
to the Internet, and uses NAT44 via Docker to translate from it's IPv4 address
to the public IPv4 address for the host. Tayga (http://www.litech.org/tayga/)
is used for this role and runs as a container on the node with **nat64** specified
as an **opmode**.

Tayga uses a pool of local IPv4 addresses that are mapped to IPv6 address. As
such, the IPv4 pool and IPv4 address of Tayga must be specified. Lazyjack checks
to make sure that both of these are inside of the **support_net** subnet, mentioned
above.
```
    v4_cidr: "172.18.0.128/25"
    v4_ip: "172.18.0.200"
```
Also, specify the IPv6 address of Tayga on the **support_net** IPv6 subnet.
```
ip: "fd00:10::200"
```

### DNS64 (dns64)
A companion to NAT64, the DNS64 container using bind9 will provide synthesized IPv6
addresses for external IPv4 addresses (currently, it does so for all addresses).
The CIDR used for this translation is specified in this section.
```
    cidr: "fd00:10:64:ff9b::/96"
```
An external address with a IPv4 address of `172.217.12.238`, would be encoded as
`fd00:10:64:ff9b::acd9:cee/96`.

The DNS64 container will forward DNS requests to a remote DNS server, which is
also specified.
```
    remote_server: "64.102.6.247"
```
The **support_net** IPv6 address of the bind9 server needs to be specified.
```
    ip: "fd00:10::100"
```

If the topology supports accessing the Internet via IPv6, the following can be
set to allow external IPv6 addresses to be used directly, instead of translating
them to IPv4 addresses using NAT64. To do this, requires telling DNS64 to use
the AAAA records for lookups. The default is false, meaning IPv4 addresses will
be used in all lookups.
```
    allow_aaaa_use: true
```

## Usage
As mentioned above, you should have Lazyjack and the YAML file on each system to be
provisioned. Since Lazyjack needs to perform privileged operations, you'll need to run this
as root:
```
   sudo ~/go/bin/lazyjack [options] {init|prepare|up|down|clean|version}
```

The commands do the following:
* **init** - Sets up tokens and certificates needed by Kuberentes. Must be run on the master node, **before** copying the config file to minion nodes. Only needed once.
* **prepare** - Prepares the node so that cluster can be brought up. Do on each node, before proceeded to next step.
* **up** - Brings up Kubernetes cluster on the node. Do master first, and then minions.
* **down** - Tears down the cluster on the node. Do minions first, and then master.
* **clean** - Reverses the prepare steps performed to clear out settings.
* **version** - Shows the version of this app and exits.

Once a cluster is up on the master, you can setup kubectl, as described by the
KubeAdm init command:
```
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
```

### Command Line Options
```
Usage: lazyjack [options] {init|prepare|up|down|clean|version}
  -alsologtostderr
        log to standard error as well as files
  -config string
        Configurations for lazyjack (default "config.yaml")
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
For each command, there are a series of actions performed...

### For the `init` command
* Creates CA certificate and key for KubeAdm.
* Creates token and CA certificate hash.
* Updates the configuration YAML file (needed for `up` command on minions).

### For the `prepare` command
* Creates support network with IPv6 and IPv4.
* Starts DNS64 container, with config file, removes IPv4 address, and adds route to NAT64 server.
* Starts NAT64 container.
* Adds IPv4 route to NAT64 server on node.
* Adds management network IP on specified interface.
* Places management network IP in /etc/hosts, for this hostname.
* Adds DNS64 support network IP as first nameserver in /etc/resolv.conf.
* Creates a drop-in file for kubelet to specify IPv6 DNS64 server IP.
* Creates KubeAdm configuration file, saves old one with .bak suffix, in case file customized.
* Adds route to DNS64 synthesized network via NAT64 server (based on node).
* Adds route to support network for other nodes to access.

### For the `up` command
* Creates CNI config file for bridge/PTP plugin.
* Create routes for each of the pod networks on other nodes.
* Reloaded daemons for services.
* Restarted kubelet service.
* On master: Place CA certificate and Key files into Kubernetes area.
* On master: Perform KubeAdm init command with config file.
* On minion: Perform KubeAdm join command using token information.

### For the `down` command
* Perform KubeAdm reset command.
* Remove routes to other nodes' pod networks.
* Removes bridge/PTP plugin's CNI config file.
* Removes the br0 interface

### For the `clean` command
* Removes drop-in file for kubelet.
* Removes IP from management interface.
* Restores /etc/hosts.
* Restores /etc/resolv.conf.
* Removes route to NAT64 server for DNS64 synthesized net.
* Removes route to support network.
* Stops and removes DNS64 container.
* Stops and removes NAT64 container.
* Removes IPv4 route to NAT64 server.
* Removes support network on DNS64/NAT64 node.


## Limitations/Restrictions
* Some newer versions of docker break the enabling of IPv6 in the containers used for DNS64 and NAT64.
* CNI v0.7.1+ is needed for full IPv6 support by plugins.
* Relies on the tayga and bind6 containers (as provided by other developers).
* The `init` command modifies the specified configuration YAML file. As a result, `init` must be done before copying the config YAML to other nodes.
* Because the config YAML file is modified by the root user, permissions is set to 777, so that the non-root user can still modify the file.


## Troubleshooting
This section has some notes on issues seen and resolutions (if any).

Tip: If for some reason the `prepare` fails after updating /etc/resolv.conf
or /etc/hosts, you can recover the originals from the .bak files created.
However, the tool is designed to allow multiple invocations, so this should
not be required.

If a system is rebooted, the entire process (`init`, copy config, `prepare`,
and `up`) must be performed, because some changes are not persisted, and some
files are created in /tmp.

On a related note, you want to make sure that the node does not already have
incompatible configuration on being interfaces or for routes that will be
defined.

I had one case where I could not ping from a pod on one node, to a pod on
another (but it worked in the reverse direction). Looks like an issue with
some stray IPTABLES rules. Found out that I could tear everything down, do
the following commands to flush IPTABLES rules, and then bring everything
up:
```
sudo iptables -P INPUT ACCEPT
sudo iptables -P FORWARD ACCEPT
sudo iptables -P OUTPUT ACCEPT
sudo iptables -t nat -F
sudo iptables -t mangle -F
sudo iptables -F
sudo iptables -X

sudo ip6tables -P INPUT ACCEPT
sudo ip6tables -P FORWARD ACCEPT
sudo ip6tables -P OUTPUT ACCEPT
sudo ip6tables -t nat -F
sudo ip6tables -t mangle -F
sudo ip6tables -F
sudo ip6tables -X
```

I had another case where kube-dns was not coming up, and kube-proxy log was
showing IPTABLES restore errors saying "iptables-restore v1.6.0: invalid
 mask `128' specified". This should be using the ip6tables-restore operation.
I was unable to find root cause, but did KubeAdm reset, `clean` command,
fllush IPTABLES rules (like above), rebooted, and problem was cleared. May
have been corruption of IPTABLES rules.

Tip: You can customize kubeadm.conf, after the prepare step, to make any
additional changes desired for the configuration (e.g. setting kubernetesVersion
to a specific version).


## TODOs/Futures

### Implementation
* Enhance validation
  * Ensure IP addresses, subnets, and all CIDRs are valid.
  * No overlap on pod, management, and support networks.
  * Make sure pod network prefix and size are compatible (prefix should be size - 16 bits).
  * Ensure NAT64 IP is within NAT64 subnet, and that NAT64 subnet is with support subnet.
  * Node IDs > 0. >1?
  * Docker version.
  * Kubeadm, kubectl, kubelet version 1.9+.
  * Go version.
  * CNI plugin version 0.7.1+.
  * Other tools?
* Support Calico plugin. Cillium? Contiv? Others?

### Details to figure out
* Decide how to handle prepare failures (exits currently). Rollback? Difficulty?
* Create makefile for building/installing. Build executable for immediate use?
* Is there a way to check if management interface already has an (incompatible) IPv6 address?
* Way to timeout on "kubeadm init", if it gets stuck (e.g. kubelet never comes up).

### Enhancements to consider
* Do Istio startup. Useful?  Metal LB startup?
* Running DNS64 and NAT64 on separate nodes. Useful? Routing?
* Is it useful to try with with IPv4 addresses (only) as a vanilla provisioner.
* Support hypervisors other than Docker (have separated out the code)?
* Consider using Kubeadm's DynamicKubeletConfig, instead of drop-in file for kubelet.
* Could skip running kubeadm commands and just display them, for debugging (how to best do that? command line arg?)
* Could copy /etc/kubernetes/admin.conf to ~/.kube/config and change ownership, if can identify user name.
* Using separate go routine for kubeadm commands, and provide a (configurable) timeout.
* Consider including NAT64/DNS64 containers into project t oremove dependencies.
* Consider using different default area for config files, especially for config file for DNS64 container. Could
  use docker volume?
