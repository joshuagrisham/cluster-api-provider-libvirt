/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

const (
	// MachineFinalizer allows ReconcileLibvirtMachine to clean up Libvirt resources associated with LibvirtMachine before
	// removing it from the apiserver.
	MachineFinalizer = "libvirtmachine.infrastructure.cluster.x-k8s.io"
)

// LibvirtMachineSpec defines the desired state of LibvirtMachine
type LibvirtMachineSpec struct {
	// Network is the name of the network to which the LibvirtMachine will be connected. Uses the 'default' network if not specified.
	// Assumes that the network already exists and has DHCP enabled. TODO: Support static IPs and specify hostname per LibvirtMachine?
	// +optional
	Network *string `json:"network,omitempty"`

	// StoragePool is the name of the storage pool where the LibvirtMachine's disk will be created. Uses the 'default' storage pool if not specified.
	// Assumes that the storage pool already exists and has been started.
	// +optional
	StoragePool *string `json:"storagePool,omitempty"`

	// CPU is the number of virtual CPUs assigned to the LibvirtMachine.
	CPU int32 `json:"cpu"`

	// Memory is the amount of memory (in MiB) assigned to the LibvirtMachine.
	Memory int32 `json:"memory"`

	// DiskSize is the size (in GiB) allocated to the primary operating system disk mounted to the LibvirtMachine.
	DiskSize int32 `json:"diskSize"`

	// BackingImagePath is a path on the libvirt target host of an image you have already downloaded and wish to use as the base image for the primary operating system disk of the LibvirtMachine.
	BackingImagePath string `json:"backingImagePath"`

	// BackingImageFormat is the format of the backing image (e.g., "qcow2") at BackingImagePath. Uses the 'qcow2' format if not specified.
	// +optional
	BackingImageFormat *string `json:"backingImageFormat,omitempty"`

	// ProviderID is the unique identifier for this machine as exposed by the infrastructure provider.
	// This field is required by Cluster API to link the Machine resource to the infrastructure machine.
	// Format: libvirt:///<machine-name>
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=512
	ProviderID string `json:"providerID,omitempty"`
}

// LibvirtMachineInitializationStatus provides observations of the LibvirtMachine initialization process.
// +kubebuilder:validation:MinProperties=1
type LibvirtMachineInitializationStatus struct {
	// provisioned is true when the infrastructure provider reports that the Machine's infrastructure is fully provisioned.
	// NOTE: this field is part of the Cluster API contract, and it is used to orchestrate initial Machine provisioning.
	// +optional
	Provisioned bool `json:"provisioned,omitempty"`
}

// LibvirtMachineStatus defines the observed state of LibvirtMachine.
type LibvirtMachineStatus struct {
	// conditions represent the current state of the LibvirtMachine resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// addresses contains the associated addresses for the machine.
	// +optional
	Addresses []clusterv1.MachineAddress `json:"addresses,omitempty"`

	// initialization provides observations of the LibvirtMachine initialization process.
	// NOTE: Fields in this struct are part of the Cluster API contract and are used to orchestrate initial Machine provisioning.
	// +optional
	Initialization LibvirtMachineInitializationStatus `json:"initialization,omitempty,omitzero"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=libvirtmachines,scope=Namespaced,categories=cluster-api
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels['cluster\\.x-k8s\\.io/cluster-name']",description="Cluster"
// +kubebuilder:printcolumn:name="ProviderID",type="string",JSONPath=".spec.providerID",description="Provider ID"

// LibvirtMachine is the Schema for the libvirtmachines API
type LibvirtMachine struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of LibvirtMachine
	// +required
	Spec LibvirtMachineSpec `json:"spec"`

	// status defines the observed state of LibvirtMachine
	// +optional
	Status LibvirtMachineStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// LibvirtMachineList contains a list of LibvirtMachine
type LibvirtMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []LibvirtMachine `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &LibvirtMachine{}, &LibvirtMachineList{})
}
