# Kubernetes Cluster API Provider for Libvirt (CAPLV)

Kubernetes-native declarative infrastructure for [Libvirt](https://libvirt.org/).

## What is the Cluster API Provider for Libvirt (CAPLV)?

The [Cluster API](https://github.com/kubernetes-sigs/cluster-api) brings declarative Kubernetes-style APIs to cluster creation, configuration and management. Cluster API Provider for Libvirt (CAPLV) is an implementation of an infrastructure provider for the Cluster API using Libvirt.

> [!NOTE]
> The Libvirt provider is **not** designed for production use and is intended for development environments only.

CAPLV only officially supports CAPI version `v1beta2` at this time. Other version support can be added in the future as needed.

## Why did I make this?

I wanted a CAPI Infrastructure provider that I could test locally, but that would more closely mirror a "real" setup with other providers (such as vSphere, Nutanix, AWS, etc) where you get a "real" virtual machine that will look and feel similar to a production environment. Using Libvirt to create KVMs seemed like a great fit, but I could not find any usable Libvirt provider for CAPI already available.

Yes, [there is one for OpenShift](https://github.com/openshift/cluster-api-provider-libvirt), but it requires the OpenShift machine-api and does not work with other Kubernetes clusters. Since I don't plan to use OpenShift for anything right now, that one is out.

There also appear to be [several other ones on GitHub](https://github.com/search?q=cluster-api-provider-libvirt&type=repositories), but they are either outdated, don't really work with CAPI, or are just an empty repository.

## Version compatibility

- CAPLV version `0.1.x` is **_ONLY_** compatible with CAPI `1.10.x` and API version `v1beta1`
- CAPLV version `0.2.x` is **_ONLY_** compatible with CAPI `1.12.x` and API version `v1beta2`

The reason it is like this is so that CAPLV `0.1.x` can be used out-of-the-box with Rancher 2.13.x and Rancher Turtles (otherwise it throws errors that it does not support `v1beta2`).

## Getting started

CAPLV will create KVM-based virtual machines using the Libvirt daemon specified in the `LIBVIRT_URI` environment variable at the time of installing the provider. These virtual machines will then be bootstrapped using [cloud-init](https://cloud-init.io/) and serve as your newly-created Kubernetes cluster's control plane and worker nodes.

Add the `libvirt` infrastructure provider to your `clusterctl.yaml`, setting the CAPLV version based on which version of CAPI you intend to use:

```yaml
# ~/.config/cluster-api/clusterctl.yaml
providers:
  - name: "libvirt"
    url: "https://github.com/joshuagrisham/cluster-api-provider-libvirt/releases/v0.1.0/infrastructure-components.yaml"
    type: InfrastructureProvider
```

Note that you must also only use the version of `clusterctl` that is compatible with that version of CAPI. You can download a separate version of `clusterctl` if you like; here is an example:

```sh
wget -O bin/clusterctl-v1.10.10 https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.10.10/clusterctl-linux-amd64
chmod +x bin/clusterctl-v1.10.10
./bin/clusterctl-v1.10.10 version
```

## Installation instructions

At a high level, the easiest way to get CAPLV working is:

- install the Libvirt daemon and enable Libvirt's [remote TCP socket listener](https://libvirt.org/remote.html) on your host OS
  - ⚠️ **NOTE:** TLS and/or SASL-based username and passwords are not supported by CAPLV at this time, which means that anyone who can remotely access the Libvirt TCP port on your host will be able to freely control anything on it without needing to authenticate--use at your own risk and/or set up a firewall to block remote access! You have been warned!
- (optionally) install `virt-manager` and/or other Libvirt clients just to make your life easier (to manage and/or check the status of your virtual machines, networks, and storage volumes)
- (optionally) create a new Libvirt network with a bridge so you can access the virtual machines from your host OS
  - see [examples/k8s-libvirt-network.xml](./examples/k8s-libvirt-network.xml) for an example `k8s` network definition
- (optionally) create a new Libvirt storage pool so you the disks for your cluster nodes will be easier to manage together
- set `LIBVIRT_URI` to a `qemu+tcp`-based remote URI using the Libvirt network's gateway IP address
  - if using the `k8s` network from the above example, the URI would be `qemu+tcp://192.168.128.1/system` as the gateway address of this newly-created network is `192.168.128.1`
- bootstrap a new Kubernetes cluster (using `kind`, `k3d`, `minikube`, or any other Kubernetes cluster you can easily spin up on your local host OS)
- install the Cluster API to your new bootstrap cluster using [`clusterctl`](https://cluster-api.sigs.k8s.io/clusterctl/overview) with the `libvirt` Infrastructure provider (see above for the necessary `clusterctl.yaml` configuration) and the [bootstrap and control-plane providers](https://cluster-api.sigs.k8s.io/reference/providers) of choice
- prepare the source image for your new VM instances (the Libvirt "backing image")
  - What kind of backing image is needed will depend entirely on the bootstrap provider that you wish to use. For example, with Kubeadm you will probably want to use the CAPI [image-builder](https://github.com/kubernetes-sigs/image-builder), but for other providers (such as K3s and RKE2) you only need to find a cloud-init capable image that meets the provider's requirements and download it to your host OS.
- create a cluster manifest by one of the following methods:
  - using `clusterctl generate cluster` to generate a manifest from the provider
  - taking a copy of one of the [examples](./examples/) provided in this repository
  - creating your own cluster manifest(s)
- apply the cluster manifest to your bootstrap cluster and gleefully watch as your new Libvirt-based cluster spins up automatically!

### Libvirt setup

Enable the Libvirt daemon TCP listener and disable authentication over TCP by modifying the file `/etc/libvirt/libvirtd.conf` and ensuring that the following two values are set:

```ini
listen_tcp = 1
auth_tcp = "none"
```

Then restart `libvirtd` with the TCP listener enabled:

```sh
sudo systemctl stop libvirtd
sudo systemctl enable --now libvirtd-tcp.socket
```

Just to help keep things organized and running smoothly, we can set up a new network and storage pool for Libvirt using the [`virsh`](https://www.libvirt.org/manpages/virsh.html) utility.

Create and start a Libvirt storage pool (this one we will call `k8s` at the path `/k8s`):

```sh
sudo mkdir /k8s
sudo chmod 777 /k8s
virsh pool-create-as --name k8s --type dir --target /k8s
```

Create a Libvirt network (using the example file [examples/k8s-libvirt-network.xml](./examples/k8s-libvirt-network.xml)):

```sh
virsh net-create examples/k8s-libvirt-network.xml
```

This example network has a gateway IP of `192.168.128.1` and will provide DHCP for your VMs in the range `192.168.128.100-192.168.128.254`. The range `192.168.128.2-192.168.128.99` is reserved for static IP assignments, such as for using with `kube-vip` to provide stable control plane endpoints for your clusters (more on this below).

If you restart your host OS, or for any other reason need to restart the network or storage pool, you can just start them again with `virsh` like this:

```sh
virsh pool-start k8s
virsh net-start k8s
```

### Acquire a backing image

#### Kubeadm backing image

If you wish to use the Kubeadm bootstrap and control plane provider, it is easiest to build a new image locally as the backing image by using the [CAPI Image Builder](https://image-builder.sigs.k8s.io/capi/capi). In particular, we will be using the [raw image support](https://image-builder.sigs.k8s.io/capi/providers/raw).

Here is a basic example to help get you started (but please refer to the Image Builder documentation for more information!):

```sh
git clone https://github.com/kubernetes-sigs/image-builder.git capi-image-builder
cd capi-image-builder/images/capi/

# update kubernetes.json with desired versions
nano kubernetes.json

# build dependencies
make deps-qemu

# build image
make build-qemu-ubuntu-2204
```

Once complete, you will have a new `qcow2` (raw) -formatted image created under `./output/` which you can then copy to the path of your Libvirt storage pool (e.g. under `/k8s/` if using the example from above).

```sh
# Copy but add the qcow2 extension
cp output/ubuntu-2204-kube-v1.34.2/ubuntu-2204-kube-v1.34.2 /k8s/ubuntu-2204-kube-v1.34.2.qcow2
# Open permissions to help prevent various issues
sudo chmod 777 /k8s/ubuntu-2204-kube-v1.34.2.qcow2
```

When using this example, you would set the `LibvirtMachine[Template]`'s `spec.backingImagePath` to `/k8s/ubuntu-2204-kube-v1.34.2`.

#### K3s/RKE2 (and possibly other providers?) backing image

For K3s and RKE2 (and possibly other providers?), it is normally sufficient to use a so-called "Cloud Image" from the distribution of your choice.

When testing this myself, I just used one of the [Ubuntu Cloud Images](https://cloud-images.ubuntu.com/).

Example:

```sh
# Download noble-server-cloudimg-amd64.img directly to /k8s/
wget -O /k8s/noble-server-cloudimg-amd64.img https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img
```

When using this example, you would set the `LibvirtMachine[Template]`'s `spec.backingImagePath` to `/k8s/noble-server-cloudimg-amd64.img`.

### Create a bootstrap cluster

My current local Kubernetes provider of choice is [k3d](https://k3d.io/), but you can probably use something else like [kind](https://kind.sigs.k8s.io/) or [minikube](https://minikube.sigs.k8s.io/) instead if you wish.

```sh
# Create a bootstrap cluster
k3d cluster create bootstrap
```

### Create a Libvirt cluster with Kubeadm

Now you can create a downstream cluster hosted on Libvirt virtual machines.

First, you will need to install the Cluster API components and the `libvirt` infrastructure provider, plus your choice of bootstrap and control-plane provider. Below is an example using the Kubeadm controllers:

```sh
# Set the LIBVIRT_URI that CAPLV should use
export LIBVIRT_URI=qemu+tcp://192.168.128.1/system
# Enable the Cluster Topology feature flag
export CLUSTER_TOPOLOGY=true
# Install the Cluster API and providers:
clusterctl init --infrastructure libvirt --bootstrap kubeadm --control-plane kubeadm
```

> [!NOTE]
> Enabling the topology feature flag (as per the example above) is highly recommend if you wish to create `Clusters` using a `ClusterClass`!

In order to work with CAPI, our clusters will need to have a stable control plane endpoint defined before we can create them. You are free to create or use an existing external load balancer for this purpose, but the examples and templates provided by CAPLV use [kube-vip](https://kube-vip.io/) to advertise a pre-determined IP address from inside of the cluster during the startup process.

With our example `k8s` Libvirt network, we can use an address such as `192.168.128.10` for a pre-defined control plane endpoint, since it is outside of the range of DHCP. If using one of the templates provided by the infrastructure provider, you would need to set the following environment variables:

```sh
export LIBVIRT_MACHINE_NETWORK=k8s
export LIBVIRT_MACHINE_STORAGE_POOL=k8s
export LIBVIRT_MACHINE_BACKING_IMAGE=/k8s/ubuntu-2204-kube-v1.34.2.qcow2
export LIBVIRT_CONTROL_PLANE_ENDPOINT_HOST=192.168.128.10
export LIBVIRT_SSH_PUBLIC_KEY="ssh-rsa AAAAB3Nza..." # Set to the value of a public key that you wish to be able to log in with using the LIBVIRT_SSH_USER

## Additional optional default environment variables:
#export LIBVIRT_CONTROL_PLANE_ENDPOINT_PORT=6443
#export LIBVIRT_SSH_USER=clusteradmin
#export LIBVIRT_CONTROL_PLANE_CPU=2
#export LIBVIRT_CONTROL_PLANE_MEMORY=3072
#export LIBVIRT_CONTROL_PLANE_DISK_SIZE=20
#export LIBVIRT_WORKER_MACHINE_CPU=1
#export LIBVIRT_WORKER_MACHINE_MEMORY=1024
#export LIBVIRT_WORKER_MACHINE_DISK_SIZE=20
```

Then create the cluster manifest using `clusterctl`:

```sh
clusterctl generate cluster my-cluster \
  --kubernetes-version=1.34.2 \
  --control-plane-machine-count=1 \
  --worker-machine-count=1 > my-cluster.yaml
```

> [!NOTE]
> With Kubeadm, the `--kubernetes-version` parameter (or env variable `KUBERNETES_VERSION`) is not super important, as what really matters is the version that exists in the backing image that you built. However, the version field of the various resource specs will be used to help steer the rolling update processes of the various CAPI controllers, so it is best if you try to keep the value in sync with the version of your desired backing image!

Now you can apply the newly-generated manifest:

```sh
kubectl apply -f my-cluster.yaml
```

It can take some time for the control plane to come up and for `kube-vip` to start advertising the IP address. You can check if the endpoint is up by polling for it:

```sh
watch curl --verbose --insecure https://192.168.128.10:6443
```

Once the cluster is online and available, you should be able to fetch its `KUBECONFIG` from the bootstrap cluster and connect to it using `kubectl`.

Example:

```sh
# Get my-cluster's KUBECONFIG
kubectl get secret my-cluster-kubeconfig -o jsonpath='{.data.value}' | base64 -d > my-cluster.kubeconfig

# Use the KUBECONFIG with kubectl
KUBECONFIG=./my-cluster.kubeconfig kubectl get nodes
```

### Create a Libvirt cluster with RKE2

Installing a cluster with RKE2 is almost exactly the same, with only a few minor differences (namely, the `KUBERNETES_VERSION` must exist as an RKE2 release tag (see: <https://github.com/rancher/rke2/releases>), and that the backing image should point to a generic cloud image). In RKE2's case, the bootstrap process will actually fetch this version of RKE2 and install it.

```sh
# Fetch the latest noble-server-cloudimg-amd64.img
wget -O /k8s/noble-server-cloudimg-amd64.img https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img

# Set the LIBVIRT_URI that CAPLV should use
export LIBVIRT_URI=qemu+tcp://192.168.128.1/system
# Install the Cluster API and providers:
CLUSTER_TOPOLOGY=true clusterctl init --infrastructure libvirt --bootstrap rke2 --control-plane rke2

export LIBVIRT_MACHINE_NETWORK=k8s
export LIBVIRT_MACHINE_STORAGE_POOL=k8s
export LIBVIRT_MACHINE_BACKING_IMAGE=/k8s/noble-server-cloudimg-amd64.img
export KUBERNETES_VERSION=v1.34.2+rke2r1
export LIBVIRT_CONTROL_PLANE_ENDPOINT_HOST=192.168.128.10
export LIBVIRT_SSH_PUBLIC_KEY="ssh-rsa AAAAB3Nza..."
```

Creation of the manifest is almost exactly the same, except that you need to specify the `rke2` flavor:

```sh
clusterctl generate cluster my-cluster --control-plane-machine-count=1 --worker-machine-count=1 --flavor rke2 > my-cluster.yaml
```

Applying and connecting to it work in the same way:

```sh
kubectl apply -f my-cluster.yaml

kubectl get secret my-cluster-kubeconfig -o jsonpath='{.data.value}' | base64 -d > my-cluster.kubeconfig

KUBECONFIG=./my-cluster.kubeconfig kubectl get nodes
```

### Additional templates

The default `cluster-template.yaml` provided by CAPLV assumes that you are using the Kubeadm bootstrap and control-plane provider. Other template "flavors" are also available:

```sh
# The default template using Kubeadm
clusterctl generate cluster my-cluster > my-cluster.yaml

# "clusterclass": A template which creates a Cluster based on a ClusterClass (also using Kubeadm)
clusterctl generate cluster my-cluster --flavor clusterclass > my-cluster.yaml

# "rke2": The default template using RKE2
clusterctl generate cluster my-cluster --flavor rke2 > my-cluster.yaml

# "clusterclass-rke2": A template which creates a Cluster based on a ClusterClass using RKE2
clusterctl generate cluster my-cluster --flavor clusterclass-rke2 > my-cluster.yaml
```

For more examples, feel free to head on over to the [examples](./examples/) folder!

## Troubleshooting and other considerations

- If you have any problems, you can check the logs of the `caplv-controller-manager` pod or the other various CAPI controller pods to see if they give any clues.
- You can also log in to the virtual machines over SSH using its IP address and the SSH key that you provided. Of particular help is the `cloud-final` journal log (`sudo journalctl -u cloud-final`) but you can also check anything else you wish within the VM.
- If there was an unrecoverable problem creating your virtual machine, or you have some orphaned Libvirt resources, you might need to clean then up manually to help resolve some issues. `virsh` is your friend here (`virsh vol-list`, `virsh vol-delete`, etc).
- Virtual machine names cannot be longer than 63 characters, as this name is used as the hostname within the VM itself. The virtual machine name will be taken directly from the name of the declared or generated `LibvirtMachine` resource. Note that when using additional APIs such as `ClusterClass`, the generated name will be a concatenation of several fields (the `Cluster` name, the Machine Deployment class name, and 3 sets of randomly generated identifiers), so care might need to be taken to ensure that you do not name your resources with names that will not fit within this limit. The CAPLV controller will raise an error in cases where this limit has been reached and not attempt to create any underlying virtual machine until the name is corrected.
- As mentioned before, there is currently no support for using the Libvirt daemon remotely with TLS or SASL usernames or passwords, which can open a security risk of your Libvirt daemon's TCP port is accessible by other hosts on your network.
- There is currently no support for Machine Pools (and no `LibvirtMachinePool` resource defined). I looked into this and tried it out a bit, but did not see any real value in trying to implement logic for this (we get better features by using a `ClusterClass` and no other changes are required, for example).
- All virtual machines will receive a dynamic IP address; there is no support for reserving static IP addresses or using an `IPAddressPool` resources at this time.
- As mentioned above, the desired Libvirt network, storage pool, and disk backing images must be available and managed directly on the host OS before they can be used--CAPLV does not currently have any features to support managing these type of resources!
- Currently, the CAPLV controller does not set any [status conditions](https://github.com/kubernetes-sigs/cluster-api/blob/main/docs/proposals/20240916-improve-status-in-CAPI-resources.md) on the CAPLV resources. Support will ideally be added for this in a coming release!
- Unit tests and e2e tests are not developed or tested.
