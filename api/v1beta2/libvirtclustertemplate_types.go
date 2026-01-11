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

// LibvirtClusterTemplateSpec defines the desired state of LibvirtClusterTemplate.
type LibvirtClusterTemplateSpec struct {
	Template LibvirtClusterTemplateResource `json:"template"`
}

// +kubebuilder:resource:path=libvirtclustertemplates,scope=Namespaced,categories=cluster-api
// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// LibvirtClusterTemplate is the Schema for the libvirtclustertemplates API.
type LibvirtClusterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec LibvirtClusterTemplateSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// LibvirtClusterTemplateList contains a list of LibvirtClusterTemplate.
type LibvirtClusterTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LibvirtClusterTemplate `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &LibvirtClusterTemplate{}, &LibvirtClusterTemplateList{})
}

// LibvirtClusterTemplateResource describes the data needed to create a LibvirtCluster from a template.
type LibvirtClusterTemplateResource struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	ObjectMeta clusterv1.ObjectMeta `json:"metadata,omitempty,omitzero"`
}
