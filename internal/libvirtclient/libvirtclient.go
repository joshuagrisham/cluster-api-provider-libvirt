package libvirtclient

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/url"
	"os"

	"github.com/digitalocean/go-libvirt"
	"github.com/kdomanski/iso9660"
)

type LibvirtClientMachine struct {
	Name               string
	NetworkName        string
	StoragePoolName    string
	CPU                int32
	Memory             int32  // in MiB
	DiskSize           int32  // in GiB
	BackingImagePath   string // path on the libvirt target where the base cloud image is located
	BackingImageFormat string // format of the BackingImagePath image; defaults to 'qcow2'
	UserData           string // cloud-init user data

	client              *libvirt.Libvirt // Libvirt client
	diskVolumeName      string           // name of the disk volume created
	cloudInitVolumeName string           // name of the cloud-init ISO volume created
}

func (vm *LibvirtClientMachine) openClient() error {
	uri := os.Getenv("LIBVIRT_URI")
	if uri == "" {
		uri = os.Getenv("LIBVIRT_DEFAULT_URI")
	}
	if uri == "" {
		return fmt.Errorf("LIBVIRT_URI or LIBVIRT_DEFAULT_URI environment variable must be set in order to connect to libvirt")
	}

	// TODO: Add support for LIBVIRT_SASL_USERNAME and LIBVIRT_SASL_PASSWORD ??

	var err error
	urlParsed, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("failed to parse libvirt URI: %v", err)
	}

	vm.client, err = libvirt.ConnectToURI(urlParsed)
	if err != nil {
		return fmt.Errorf("failed to connect to libvirt: %v", err)
	}

	vm.diskVolumeName = fmt.Sprintf("%s.qcow2", vm.Name)
	if vm.BackingImageFormat == "" {
		vm.BackingImageFormat = "qcow2"
	}
	vm.cloudInitVolumeName = fmt.Sprintf("%s-cloudinit.iso", vm.Name)
	return nil
}

func (vm *LibvirtClientMachine) closeClient() error {
	err := vm.client.Disconnect()
	if err != nil {
		return fmt.Errorf("failed closing connection to libvirt: %v", err)
	}
	return nil
}

func (vm *LibvirtClientMachine) createDisk() (string, error) {
	pool, err := vm.client.StoragePoolLookupByName(vm.StoragePoolName)
	if err != nil {
		return "", fmt.Errorf("failed to get storage pool '%s': %v", vm.StoragePoolName, err)
	}

	// Create volume with backing store via libvirt XML
	volumeXML := fmt.Sprintf(`<volume>
  <name>%s</name>
  <capacity unit='GiB'>%d</capacity>
  <target>
    <format type='qcow2'/>
  </target>
  <backingStore>
    <path>%s</path>
    <format type='%s'/>
  </backingStore>
</volume>`, vm.diskVolumeName, vm.DiskSize, vm.BackingImagePath, vm.BackingImageFormat)

	// TODO: Instead of requiring the backing image to already exist on the target libvirt host, we could create a new storage volume and then download the image and upload it to the new volume?

	vol, err := vm.client.StorageVolCreateXML(pool, volumeXML, 0)
	if err != nil {
		return "", fmt.Errorf("failed to create storage volume: %v", err)
	}

	slog.Debug("storage volume created successfully", "volume", vm.diskVolumeName, "pool", vm.StoragePoolName)

	path, err := vm.client.StorageVolGetPath(vol)
	if err != nil {
		return "", fmt.Errorf("failed to get storage volume path: %v", err)
	}

	return path, nil
}

func (vm *LibvirtClientMachine) createCloudInitISO() (string, error) {
	pool, err := vm.client.StoragePoolLookupByName(vm.StoragePoolName)
	if err != nil {
		return "", fmt.Errorf("failed to get storage pool '%s': %v", vm.StoragePoolName, err)
	}

	// Create the ISO content in memory first
	writer, err := iso9660.NewWriter()
	if err != nil {
		return "", fmt.Errorf("failed to create cloud-init ISO writer: %v", err)
	}
	defer writer.Cleanup()

	slog.Debug("cloud-init user-data:\n", "user-data", vm.UserData)

	// Add user-data file
	if err := writer.AddFile(bytes.NewReader([]byte(vm.UserData)), "user-data"); err != nil {
		return "", fmt.Errorf("failed to add cloud-init user-data: %v", err)
	}

	// Add barebones meta-data file
	metadata := fmt.Sprintf("instance-id: %s\nlocal-hostname: %s\n", vm.Name, vm.Name)
	if err := writer.AddFile(bytes.NewReader([]byte(metadata)), "meta-data"); err != nil {
		return "", fmt.Errorf("failed to add cloud-init meta-data: %v", err)
	}

	// Write ISO to temporary buffer
	var buf bytes.Buffer
	if err := writer.WriteTo(&buf, "cidata"); err != nil {
		return "", fmt.Errorf("failed to write cloud-init ISO to buffer: %v", err)
	}

	// Create volume for the ISO via libvirt XML
	volumeXML := fmt.Sprintf(`<volume>
  <name>%s</name>
  <capacity unit='bytes'>%d</capacity>
  <target>
    <format type='raw'/>
  </target>
</volume>`, vm.cloudInitVolumeName, buf.Len())

	vol, err := vm.client.StorageVolCreateXML(pool, volumeXML, 0)
	if err != nil {
		return "", fmt.Errorf("failed to create cloud-init storage volume: %v", err)
	}

	// Upload the ISO content to the volume
	err = vm.client.StorageVolUpload(vol, bytes.NewBuffer(buf.Bytes()), 0, uint64(buf.Len()), 0)
	if err != nil {
		return "", fmt.Errorf("failed to upload cloud-init ISO to storage volume: %v", err)
	}

	var test bytes.Buffer
	err = vm.client.StorageVolDownload(vol, &test, 0, 0, 0)
	if err != nil {
		return "", fmt.Errorf("failed to download cloud-init ISO from storage volume: %v", err)
	}
	if !bytes.Equal(test.Bytes(), buf.Bytes()) {
		return "", fmt.Errorf("storage volume %v content does not match uploaded data", vm.cloudInitVolumeName)
	}

	slog.Debug("cloud-init storage volume created successfully", "volume", vm.cloudInitVolumeName, "pool", vm.StoragePoolName)

	path, err := vm.client.StorageVolGetPath(vol)
	if err != nil {
		return "", fmt.Errorf("failed to get cloud-init ISO volume path: %v", err)
	}

	return path, nil
}

func (vm *LibvirtClientMachine) Create() error {

	if len(vm.Name) > 63 {
		return fmt.Errorf("VM name '%s' is too long; must be 63 characters or less", vm.Name)
	}

	slog.Debug("creating VM", "name", vm.Name)

	err := vm.openClient()
	if err != nil {
		return err
	}
	defer vm.closeClient()

	isoPath, err := vm.createCloudInitISO()
	if err != nil {
		return fmt.Errorf("failed to create cloud-init ISO: %v", err)
	}

	diskPath, err := vm.createDisk()
	if err != nil {
		return fmt.Errorf("failed to create disk: %v", err)
	}

	// Create the VM via libvirt XML
	domainXML := fmt.Sprintf(`<domain type='kvm'>
  <name>%s</name>
  <memory unit='MiB'>%d</memory>
  <vcpu>%d</vcpu>
  <os>
    <type arch='x86_64'>hvm</type>
    <boot dev='hd'/>
  </os>
  <devices>
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2'/>
      <source file='%s'/>
      <target dev='vda' bus='virtio'/>
    </disk>
    <disk type='file' device='cdrom'>
      <driver name='qemu' type='raw'/>
      <source file='%s'/>
      <target dev='hda' bus='ide'/>
      <readonly/>
    </disk>
    <interface type='network'>
      <source network='%s'/>
      <model type='virtio'/>
    </interface>
    <serial type='pty'>
      <target type='isa-serial' port='0'>
        <model name='isa-serial'/>
      </target>
    </serial>
    <console type='pty'>
      <target type='serial' port='0'/>
    </console>
  </devices>
</domain>`, vm.Name, vm.Memory, vm.CPU, diskPath, isoPath, vm.NetworkName)

	// Define and start domain
	domain, err := vm.client.DomainDefineXML(domainXML)
	if err != nil {
		return err
	}

	if err := vm.client.DomainCreate(domain); err != nil {
		return err
	}

	return nil
}

func (vm *LibvirtClientMachine) Destroy() error {

	slog.Debug("destroying VM", "name", vm.Name)

	err := vm.openClient()
	if err != nil {
		return err
	}
	defer vm.closeClient()

	// Look up the domain
	domain, err := vm.client.DomainLookupByName(vm.Name)
	if err != nil {
		return fmt.Errorf("failed to lookup domain: %v", err)
	}

	// Check if domain is running and destroy it
	active, err := vm.client.DomainIsActive(domain)

	//domain.IsActive()
	if err != nil {
		return fmt.Errorf("failed to check domain state: %v", err)
	}

	if active == int32(libvirt.DomainRunning) {
		slog.Debug("stopping VM", "name", vm.Name)
		if err := vm.client.DomainDestroy(domain); err != nil {
			return fmt.Errorf("failed to stop domain: %v", err)
		}
	}

	// Undefine the domain
	slog.Debug("undefining VM", "name", vm.Name)
	if err := vm.client.DomainUndefine(domain); err != nil {
		return fmt.Errorf("failed to undefine domain: %v", err)
	}

	// Delete volumes from storage pool
	pool, err := vm.client.StoragePoolLookupByName(vm.StoragePoolName)
	if err != nil {
		return fmt.Errorf("failed to get storage pool '%s': %v", vm.StoragePoolName, err)
	}

	// Refresh pool to detect all volumes
	if err := vm.client.StoragePoolRefresh(pool, 0); err != nil {
		slog.Warn("failed to refresh pool", "error", err)
	}

	// Delete disk volume
	vol, err := vm.client.StorageVolLookupByName(pool, vm.diskVolumeName)
	if err == nil {
		slog.Debug("deleting volume", "volume", vm.diskVolumeName, "pool", vm.StoragePoolName)
		if err := vm.client.StorageVolDelete(vol, 0); err != nil {
			slog.Warn("failed to delete volume", "error", err)
		}
	}

	// Delete cloud-init ISO volume
	isoVol, err := vm.client.StorageVolLookupByName(pool, vm.cloudInitVolumeName)
	if err == nil {
		slog.Debug("deleting ISO volume", "volume", vm.cloudInitVolumeName, "pool", vm.StoragePoolName)
		if err := vm.client.StorageVolDelete(isoVol, 0); err != nil {
			slog.Warn("failed to delete ISO volume", "error", err)
		}
	}

	// Refresh pool again after deletions
	vm.client.StoragePoolRefresh(pool, 0)

	slog.Debug("VM destroyed and removed successfully", "name", vm.Name)

	return nil
}

func (vm *LibvirtClientMachine) Exists() bool {

	err := vm.openClient()
	if err != nil {
		slog.Error("Error opening client", "error", err)
		return false
	}
	defer vm.closeClient()

	domain, err := vm.client.DomainLookupByName(vm.Name)
	if err != nil {
		slog.Debug("Error looking up domain by name", "error", err)
		return false
	}

	return domain.Name == vm.Name
}

func (vm *LibvirtClientMachine) IsReconciled() bool {

	err := vm.openClient()
	if err != nil {
		slog.Error("Error opening client", "error", err)
		return false
	}
	defer vm.closeClient()

	domain, err := vm.client.DomainLookupByName(vm.Name)
	if err != nil {
		slog.Debug("Error looking up domain by name", "error", err)
		return false
	}

	// Get domain info
	//rState, rMaxMem, rMemory, rNrVirtCPU, rCPUTime, err := vm.client.DomainGetInfo(domain)
	_, rMaxMem, _, rNrVirtCPU, _, err := vm.client.DomainGetInfo(domain)
	if err != nil {
		slog.Debug("failed to get domain info", "error", err)
		return false
	}

	// Compare vCPUs
	if vm.CPU != int32(rNrVirtCPU) {
		slog.Debug("VM is not reconciled; CPU mismatch", "name", vm.Name, "expected", vm.CPU, "actual", int32(rNrVirtCPU))
		return false
	}

	// Compare memory
	if vm.Memory != int32(rMaxMem/1024) {
		slog.Debug("VM is not reconciled; Memory mismatch", "name", vm.Name, "expected", vm.Memory, "actual", int32(rMaxMem/1024))
		return false
	}

	return true

}

func (vm *LibvirtClientMachine) IsReady() bool {

	err := vm.openClient()
	if err != nil {
		slog.Error("Error opening client", "error", err)
		return false
	}
	defer vm.closeClient()

	domain, err := vm.client.DomainLookupByName(vm.Name)
	if err != nil {
		slog.Debug("Error looking up domain by name", "error", err)
		return false
	}

	state, _, err := vm.client.DomainGetState(domain, 0)
	if err != nil {
		slog.Debug("failed to get domain state", "error", err)
		return false
	}

	slog.Debug("domain state", "state", state) // TODO map the enum to a string??
	return state == int32(libvirt.DomainRunning)
}

func (vm *LibvirtClientMachine) GetIPAddresses() ([]string, error) {

	err := vm.openClient()
	if err != nil {
		slog.Error("Error opening client", "error", err)
		return nil, err
	}
	defer vm.closeClient()

	domain, err := vm.client.DomainLookupByName(vm.Name)
	if err != nil {
		slog.Debug("Error looking up domain by name", "error", err)
		return nil, err
	}

	state, _, err := vm.client.DomainGetState(domain, 0)
	if err != nil {
		slog.Debug("failed to get domain state", "error", err)
		return nil, err
	}

	if state != int32(libvirt.DomainRunning) {
		return nil, fmt.Errorf("Error: VM %s is not running", vm.Name)
	}

	ifaces, err := vm.client.DomainInterfaceAddresses(domain, uint32(libvirt.DomainInterfaceAddressesSrcLease), 0)
	var addresses []string
	if err == nil && len(ifaces) > 0 {
		for _, iface := range ifaces {
			if iface.Name != "" {
				for _, addr := range iface.Addrs {
					addresses = append(addresses, addr.Addr)
				}
			}
		}
	}
	return addresses, nil
}
