package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GithubIssueSpec struct {
	Repo        string `json:"repo"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type GithubIssueStatus struct {
	Conditons []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// GithubIssue is the Schema for the githubissues API
type GithubIssue struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GithubIssueSpec   `json:"spec,omitempty"`
	Status GithubIssueStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GithubIssueList contains a list of GithubIssue
type GithubIssueList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GithubIssue `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GithubIssue{}, &GithubIssueList{})
}
