/*
Copyright 2020 The Flux CD contributors.

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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GitUpdateJobSpec defines the desired state of GitUpdateJob
type GitUpdateJobSpec struct {
	// GitRepositoryRef refers to a GitRepository object giving access
	// to the repository on which to run the job.
	// +required
	GitRepositoryRef corev1.LocalObjectReference `json:"gitRepositoryRef"`
	// Image gives the image to run as a function in the git repo.
	// +required
	Image string `json:"image"`
	// Args lists any arguments to be given to the image.
	// +optional
	Args []string `json:"args,omitempty"`
	// FunctionConfig gives a resource to supply as the function's
	// input.
	// +optional
	FunctionConfig string `json:"functionConfig,omitempty"`
}

type GitJobResult string

const (
	GitJobSuccess GitJobResult = "Success"
	GitJobFailure GitJobResult = "Failure"
	GitJobUnknown GitJobResult = "Unknown"
)

// GitUpdateJobStatus defines the observed state of GitUpdateJob
type GitUpdateJobStatus struct {
	// The StartTime and CompletionTime fields are intended to match
	// batch/Job's status fields of the same name.

	// StartTime gives the time the job started, if started.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`
	// FinishedTime gives the time the job finished running.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// batch/Job has a count of successful pods and failed pods, but I
	// don't care about running several pods, so it's {success,
	// failure, unknown}

	// Result gives the outcome of the job, once completed.
	// +optional
	Result GitJobResult `json:"result,omitempty"`

	// Error gives the error message, if the job ran into one.
	// +optional
	Error string `json:"error,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Started",type=string,JSONPath=`.status.startedTime`
// +kubebuilder:printcolumn:name="Finished",type=string,JSONPath=`.status.finishedTime`
// +kubebuilder:printcolumn:name="Result",type=string,JSONPath=`.status.result`

// GitUpdateJob is the Schema for the gitupdatejobs API
type GitUpdateJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitUpdateJobSpec   `json:"spec,omitempty"`
	Status GitUpdateJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GitUpdateJobList contains a list of GitUpdateJob
type GitUpdateJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GitUpdateJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GitUpdateJob{}, &GitUpdateJobList{})
}
