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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	platformv1 "mydev.org/platform-operator/api/v1"

	ref "k8s.io/client-go/tools/reference"
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

	// fetch object
	var workload platformv1.Workload
	if err := r.Get(ctx, req.NamespacedName, &workload); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// create ServiceAccount object
	log.Info("reconciling ServiceAccount object")
	svcAccount, err := r.desiredServiceAccount(workload)
	if err != nil {
		return ctrl.Result{}, err
	}

	// apply changes to objects in the cluster
	applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner("workload-controller")}

	// apply changes for ServiceAccount
	log.Info("applying changes for ServiceAccount")
	err = r.Patch(ctx, &svcAccount, client.Apply, applyOpts...)
	if err != nil {
		return ctrl.Result{}, err
	}

	svcAccountRef, err := ref.GetReference(r.Scheme, &svcAccount)
	if err != nil {
		log.Error(err, "unable to make reference to serviceAccount", "serviceAccount", svcAccount)
	}
	workload.Status.ServiceAccount = *svcAccountRef

	// update status
	if err := r.Status().Update(ctx, &workload); err != nil {
		log.Error(err, "unable to update Workload status")
		return ctrl.Result{}, err
	}

	// done reconciling
	log.Info("reconciled Workload")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1.Workload{}).
		Owns(&corev1.ServiceAccount{}).
		Complete(r)
}
