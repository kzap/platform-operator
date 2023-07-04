/*
Copyright 2022 The Kubernetes Authors.

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
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
)

const (
	DefaultNamespace              = "platform-operator-system"
	DefaultWebhookPort            = 9443
	DefaultHealthProbeBindAddress = ":8081"
	DefaultMetricsBindAddress     = ":8080"
	DefaultLeaderElectionID       = "dcd661b7.mydev.org"
	DefaultClientConnectionQPS    = 20.0
	DefaultClientConnectionBurst  = 30
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	scheme.AddTypeDefaultingFunc(&OperatorConfig{}, func(obj interface{}) {
		SetDefaults_Configuration(obj.(*OperatorConfig))
	})
	return nil
}

func getOperatorNamespace() string {
	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}
	return DefaultNamespace
}

// SetDefaults_Configuration sets default values for ComponentConfig.
func SetDefaults_Configuration(cfg *OperatorConfig) {
	if cfg.Namespace == nil {
		cfg.Namespace = pointer.String(getOperatorNamespace())
	}
	if cfg.Webhook.Port == nil {
		cfg.Webhook.Port = pointer.Int(DefaultWebhookPort)
	}
	if len(cfg.Metrics.BindAddress) == 0 {
		cfg.Metrics.BindAddress = DefaultMetricsBindAddress
	}
	if len(cfg.Health.HealthProbeBindAddress) == 0 {
		cfg.Health.HealthProbeBindAddress = DefaultHealthProbeBindAddress
	}
	if cfg.LeaderElection != nil && cfg.LeaderElection.LeaderElect != nil &&
		*cfg.LeaderElection.LeaderElect && len(cfg.LeaderElection.ResourceName) == 0 {
		cfg.LeaderElection.ResourceName = DefaultLeaderElectionID
	}
	if cfg.ClientConnection == nil {
		cfg.ClientConnection = &ClientConnection{}
	}
	if cfg.ClientConnection.QPS == nil {
		cfg.ClientConnection.QPS = pointer.Float32(DefaultClientConnectionQPS)
	}
	if cfg.ClientConnection.Burst == nil {
		cfg.ClientConnection.Burst = pointer.Int32(DefaultClientConnectionBurst)
	}
}
