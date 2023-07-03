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
	ctrl "sigs.k8s.io/controller-runtime"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	platformv1 "mydev.org/platform-operator/api/v1"
)

func (r *WorkloadReconciler) desiredServiceAccount(workload platformv1.Workload) (corev1.ServiceAccount, error) {
	svcAccountName := workload.Name
	if workload.Spec.ServiceAccountName != "" {
		svcAccountName = workload.Spec.ServiceAccountName
	}

	svcAccount := corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{APIVersion: corev1.SchemeGroupVersion.String(), Kind: "ServiceAccount"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcAccountName,
			Namespace: workload.Namespace,
		},
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: "imagepullsecret-patcher"},
		},
	}

	// always set the controller reference so that we know which object owns this.
	if err := ctrl.SetControllerReference(&workload, &svcAccount, r.Scheme); err != nil {
		return svcAccount, err
	}

	return svcAccount, nil
}
