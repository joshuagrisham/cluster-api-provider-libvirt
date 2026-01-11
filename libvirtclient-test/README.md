# libvirtclient Test Utility

This is a small test utility of the `libvirtclient` used by this controller. It is very basic: just a copy/paste of the file [../internal/libvirtclient/libvirtclient.go](../internal/libvirtclient/libvirtclient.go) with an simple CLI created in [main.go](./main.go).

Using this, you can test and interact with a copy of the controller's `libvirtclient` locally.

## Examples

```sh
# Create a new sample VM:
LIBVIRT_URI=qemu+tcp://192.168.128.1/system go run . -create test

# Check the status of a VM:
LIBVIRT_URI=qemu:///system go run . -status test

# Delete a VM:
LIBVIRT_URI=qemu:///system go run . -delete test

# Yes, you can even interact with VMs originally created by the controller:
LIBVIRT_URI=qemu+tcp://192.168.128.1/system go run . -status demo-cluster-demo-cluster-sd9kb

# Especially useful to delete orphaned VMs
# (like if you deleted the k3d cluster before deleting the LibvirtMachine(s))
LIBVIRT_URI=qemu+tcp://192.168.128.1/system go run . -delete demo-cluster-demo-cluster-sd9kb
```
