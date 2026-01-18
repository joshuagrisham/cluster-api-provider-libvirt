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

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// LibvirtMachineTemplateResource describes the data needed to create a LibvirtMachine from a template.
type LibvirtMachineTemplateResource struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	ObjectMeta clusterv1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// Spec is the specification of the desired behavior of the machine.
	Spec LibvirtMachineSpec `json:"spec"`
}

// LibvirtMachineTemplateSpec defines the desired state of LibvirtMachineTemplate.
type LibvirtMachineTemplateSpec struct {
	Template LibvirtMachineTemplateResource `json:"template"`
}

// Architecture represents the CPU architecture of the node.
// Its underlying type is a string and its value can be any of amd64, arm64, s390x, ppc64le.
// +kubebuilder:validation:Enum=amd64;arm64;s390x;ppc64le
// +enum
type Architecture string

// Example architecture constants defined for better readability and maintainability.
const (
	ArchitectureAmd64   Architecture = "amd64"
	ArchitectureArm64   Architecture = "arm64"
	ArchitectureS390x   Architecture = "s390x"
	ArchitecturePpc64le Architecture = "ppc64le"
)

// NodeInfo contains information about the node's architecture and operating system.
// +kubebuilder:validation:MinProperties=1
type NodeInfo struct {
	// architecture is the CPU architecture of the node.
	// Its underlying type is a string and its value can be any of amd64, arm64, s390x, ppc64le.
	// +optional
	Architecture Architecture `json:"architecture,omitempty"`
	// operatingSystem is a string representing the operating system of the node.
	// This may be a string like 'linux' or 'windows'.
	// +optional
	OperatingSystem string `json:"operatingSystem,omitempty"`
}

// LibvirtMachineTemplateStatus defines the observed state of LibvirtMachineTemplate.
type LibvirtMachineTemplateStatus struct {
	// Capacity defines the resource capacity for this machine.
	// This value is used for autoscaling from zero operations as defined in:
	// https://github.com/kubernetes-sigs/cluster-api/blob/main/docs/proposals/20210310-opt-in-autoscaling-from-zero.md
	// +optional
	Capacity corev1.ResourceList `json:"capacity,omitempty"`

	// +optional
	NodeInfo NodeInfo `json:"nodeInfo,omitempty,omitzero"`

	/*
		  # Example:
			status:
				capacity:
					memory: 500mb
					cpu: "1"
					nvidia.com/gpu: "1"
				nodeInfo:
					architecture: amd64
					operatingSystem: linux
	*/

}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=libvirtmachinetemplates,scope=Namespaced,categories=cluster-api
// +kubebuilder:storageversion

// LibvirtMachineTemplate is the Schema for the libvirtmachinetemplates API.
type LibvirtMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the specification of the LibvirtMachineTemplate.
	Spec LibvirtMachineTemplateSpec `json:"spec"`

	// Status is the status of the LibvirtMachineTemplate.
	// +optional
	Status LibvirtMachineTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LibvirtMachineTemplateList contains a list of LibvirtMachineTemplate.
type LibvirtMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LibvirtMachineTemplate `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &LibvirtMachineTemplate{}, &LibvirtMachineTemplateList{})
}
