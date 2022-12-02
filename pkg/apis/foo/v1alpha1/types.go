package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/vietanhduong/xcontroller/api/foo/v1alpha1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Bar struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   *v1alpha1.Bar `json:"spec,omitempty"`
	Status BarStatus     `json:"status"`
}

type BarStatus struct {
	Ready   uint32 `json:"ready,omitempty"`
	Desired uint32 `json:"desired,omitempty"`
}

func (in *BarStatus) String() string { return fmt.Sprintf("%d/%d", in.Ready, in.Desired) }

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BarList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Bar `json:"items"`
}
