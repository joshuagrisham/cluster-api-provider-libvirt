package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LibvirtClusterStatus defines the observed state of LibvirtCluster.
type LibvirtClusterStatus struct {
	// conditions represent the current state of the LibvirtCluster resource.
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

	// +optional
	Initialization LibvirtClusterInitializationStatus `json:"initialization,omitempty,omitzero"`
}

// LibvirtClusterInitializationStatus defines the initialization state of the LibvirtClusterStatus.
type LibvirtClusterInitializationStatus struct {
	// +optional
	Provisioned bool `json:"provisioned,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=libvirtclusters,scope=Namespaced,categories=cluster-api
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels['cluster\\.x-k8s\\.io/cluster-name']",description="Cluster"

// LibvirtCluster is the Schema for the libvirtclusters API
type LibvirtCluster struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// status defines the observed state of LibvirtCluster
	// +optional
	Status LibvirtClusterStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// LibvirtClusterList contains a list of LibvirtCluster
type LibvirtClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []LibvirtCluster `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &LibvirtCluster{}, &LibvirtClusterList{})
}
