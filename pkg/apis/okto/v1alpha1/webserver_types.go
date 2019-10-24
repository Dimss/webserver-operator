package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WebServerSpec defines the desired state of WebServer
// +k8s:openapi-gen=true
type WebServerSpec struct {
	Size         int    `json:"size"`
	Image        string `json:"image"`
	WebSitesList string `json:"webSitesList"`
}

// +k8s:openapi-gen=true
type WebServerStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WebServer is the Schema for the webservers API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type WebServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WebServerSpec   `json:"spec,omitempty"`
	Status WebServerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WebServerList contains a list of WebServer
type WebServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WebServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WebServer{}, &WebServerList{})
}
