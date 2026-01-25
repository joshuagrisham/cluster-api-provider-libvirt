package v1beta1

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
