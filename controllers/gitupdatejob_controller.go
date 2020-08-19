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
	batchv1 "k8s.io/api/batch/v1"
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

	// the name of the batch job is the same as the name of the git update job
	// TODO: should uniquify it (deterministically!); or, index by owner reference
	batchJobName := req.NamespacedName
	var batchJob batchv1.Job
	err := r.Get(ctx, batchJobName, &batchJob)
	if err == nil {
		log.V(debug).Info("found batch/v1 Job", "name", batchJobName)
		updateJob.Status = r.makeStatusFromBatchJob(batchJob)
		return ctrl.Result{}, r.Status().Update(ctx, &updateJob)
	} else if client.IgnoreNotFound(err) == nil {
		log.V(debug).Info("batch/v1 Job does not exist; will be created", "name", batchJobName)
		batchJob, err = r.createBatchJob(updateJob)
		if err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, &batchJob); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("created batch/v1 Job", "name", batchJobName)
		updateJob.Status = updatev1alpha1.GitUpdateJobStatus{
			Result: updatev1alpha1.GitJobUnknown,
		}
		// this should be queued again when the job changes, because
		// it is owned
		return ctrl.Result{}, r.Status().Update(ctx, &updateJob)
	} else {
		return ctrl.Result{}, err
	}
}

func (r *GitUpdateJobReconciler) createBatchJob(update updatev1alpha1.GitUpdateJob) (batchv1.Job, error) {
	job := batchv1.Job{}
	err := ctrl.SetControllerReference(&update, &job, r.Scheme)
	return job, err
}

func (r *GitUpdateJobReconciler) makeStatusFromBatchJob(job batchv1.Job) updatev1alpha1.GitUpdateJobStatus {
	var status updatev1alpha1.GitUpdateJobStatus
	status.StartTime = job.Status.StartTime
	status.CompletionTime = job.Status.CompletionTime
	switch {
	case job.Status.Succeeded > 0:
		status.Result = updatev1alpha1.GitJobSuccess
	case job.Status.Failed > 0:
		status.Result = updatev1alpha1.GitJobFailure
	default:
		status.Result = updatev1alpha1.GitJobUnknown
	}
	// TODO not sure how to get an error, if there is one; batchv1.Job
	// doesn't have space for it in its status
	return status
}

func (r *GitUpdateJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&updatev1alpha1.GitUpdateJob{}).
		// any time a job changes, reconcile its GitUpdateJob owner
		// (if it has one)
		Owns(&batchv1.Job{}).
		Complete(r)
}
