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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	updatev1alpha1 "github.com/fluxcd/git-job-controller/api/v1alpha1"
)

const debug = 1

// GitUpdateJobReconciler reconciles a GitUpdateJob object
type GitUpdateJobReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=update.toolkit.fluxcd.io,resources=gitupdatejobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=update.toolkit.fluxcd.io,resources=gitupdatejobs/status,verbs=get;update;patch

func (r *GitUpdateJobReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()

	var updateJob updatev1alpha1.GitUpdateJob
	if err := r.Get(ctx, req.NamespacedName, &updateJob); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log := r.Log.WithValues("gitupdatejob", req.NamespacedName)

	// For each git update job, there's a batch Job that actually does
	// the work. First task when reconciling: find that Job and see
	// what _its_ status is.

	// the name of the pod is the same as the name of the git update
	// job TODO: should uniquify it (deterministically!); or, index by
	// owner reference
	podName := req.NamespacedName
	var pod corev1.Pod
	err := r.Get(ctx, podName, &pod)
	if err == nil {
		log.V(debug).Info("found Pod", "name", podName)
		updateJob.Status = r.makeStatusFromPod(pod)
		return ctrl.Result{}, r.Status().Update(ctx, &updateJob)
	} else if client.IgnoreNotFound(err) == nil {
		log.V(debug).Info("Pod does not exist; will be created", "name", podName)
		pod, err = r.createPod(updateJob)
		if err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, &pod); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("created Pod", "name", podName)
		return ctrl.Result{}, nil
	} else {
		return ctrl.Result{}, err
	}
}

func (r *GitUpdateJobReconciler) createPod(update updatev1alpha1.GitUpdateJob) (corev1.Pod, error) {
	pod := corev1.Pod{}
	err := ctrl.SetControllerReference(&update, &pod, r.Scheme)
	return pod, err
}

func (r *GitUpdateJobReconciler) makeStatusFromPod(pod corev1.Pod) updatev1alpha1.GitUpdateJobStatus {
	var status updatev1alpha1.GitUpdateJobStatus
	status.StartTime = pod.Status.StartTime
	switch pod.Status.Phase {
	case corev1.PodSucceeded:
		status.Result = updatev1alpha1.GitJobSucceeded
	case corev1.PodFailed:
		status.Result = updatev1alpha1.GitJobFailed
	case corev1.PodUnknown:
		status.Result = updatev1alpha1.GitJobUnknown
	default:
		// leave it blank
	}
	// TODO:
	// - completion time (which will have to be when the pod completion was noticed)
	// - error -- analyse the initContainer/container logs
	// + other info from stdout
	return status
}

func (r *GitUpdateJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&updatev1alpha1.GitUpdateJob{}).
		// any time a job changes, reconcile its GitUpdateJob owner
		// (if it has one)
		Owns(&corev1.Pod{}).
		Complete(r)
}
