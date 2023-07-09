/*
Copyright 2023.

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

package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	platformv1 "mydev.org/platform-operator/api/platform/v1"

	ref "k8s.io/client-go/tools/reference"
)

//const workloadFinalizer = "platform.mydev.org/finalizer"

// Definitions to manage status conditions
const (
	// typeAvailableWorkload represents the status of the entire Workload reconciliation
	typeAvailableWorkload = "Available"
)

// WorkloadReconciler reconciles a Workload object
type WorkloadReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=platform.mydev.org,resources=workloads,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=platform.mydev.org,resources=workloads/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=platform.mydev.org,resources=workloads/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;get;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Workload object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *WorkloadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.
		FromContext(ctx)

	log.Info("reconciling Workload")

	// GET: fetch object
	var workload platformv1.Workload
	if err := r.Get(ctx, req.NamespacedName, &workload); err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then, it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("workload resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get workload")
		return ctrl.Result{}, err
	}

	// Let's just set the status as Unknown when no status are available
	if workload.Status.Conditions == nil || len(workload.Status.Conditions) == 0 {
		// There are valid condition statuses. "ConditionTrue" means a resource is in the condition.
		// "ConditionFalse" means a resource is not in the condition. "ConditionUnknown" means kubernetes
		// can't decide if a resource is in the condition or not.
		meta.SetStatusCondition(&workload.Status.Conditions, metav1.Condition{Type: typeAvailableWorkload,
			Status: metav1.ConditionUnknown, Reason: "Reconciling",
			Message: "Starting reconciliation",
		})

		if err := r.Status().Update(ctx, &workload); err != nil {
			log.Error(err, "Failed to update Workload status")
			return ctrl.Result{}, err
		}

		// Let's re-fetch the workload Custom Resource after update the status
		// so that we have the latest state of the resource on the cluster and we will avoid
		// raise the issue "the object has been modified, please apply
		// your changes to the latest version and try again" which would re-trigger the reconciliation
		// if we try to update it again in the following operations
		if err := r.Get(ctx, req.NamespacedName, &workload); err != nil {
			log.Error(err, "Failed to re-fetch workload")
			return ctrl.Result{}, err
		}
	}

	// create ServiceAccount object
	log.Info("reconciling ServiceAccount object")
	svcAccount, err := r.desiredServiceAccount(workload)
	if err != nil {
		return ctrl.Result{}, err
	}

	// APPLY: apply changes to objects in the cluster
	applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner("workload-controller")}

	// apply changes for ServiceAccount
	log.Info("applying changes for ServiceAccount")
	err = r.Patch(ctx, &svcAccount, client.Apply, applyOpts...)
	if err != nil {
		// The following implementation will update the status
		meta.SetStatusCondition(&workload.Status.Conditions, metav1.Condition{Type: typeAvailableWorkload,
			Status: metav1.ConditionFalse, Reason: "Reconciling",
			Message: fmt.Sprintf("Failed to create/update the Service Account (%s): (%s)", svcAccount.Name, err),
		})

		if err := r.Status().Update(ctx, &workload); err != nil {
			log.Error(err, "Failed to update Workload status")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, err
	}

	// STATUS: The following implementation will update the status
	svcAccountRef, err := ref.GetReference(r.Scheme, &svcAccount)
	if err != nil {
		log.Error(err, "unable to make reference to serviceAccount", "serviceAccount", svcAccount)
	}
	workload.Status.ServiceAccount = *svcAccountRef

	meta.SetStatusCondition(&workload.Status.Conditions, metav1.Condition{Type: typeAvailableWorkload,
		Status: metav1.ConditionTrue, Reason: "Reconciling",
		Message: fmt.Sprintf("ServiceAccount for custom resource (%s) created successfully", workload.Name),
	})

	if err := r.Status().Update(ctx, &workload); err != nil {
		log.Error(err, "Failed to update Workload status")
		return ctrl.Result{}, err
	}

	// done reconciling
	log.Info("reconciled Workload")

	return ctrl.Result{}, nil
}

func (r *WorkloadReconciler) ReconcileServiceAccount(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.
		FromContext(ctx)

	log.Info("reconciling ServiceAccount")

	// done reconciling
	log.Info("reconciled ServiceAccount")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1.Workload{}).
		Owns(&corev1.ServiceAccount{}).
		Complete(r)
}
