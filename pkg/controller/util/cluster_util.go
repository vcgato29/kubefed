/*
Copyright 2016 The Kubernetes Authors.

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

package util

import (
	"context"
	"time"

	"github.com/pkg/errors"

	apiv1 "k8s.io/api/core/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	fedv1a1 "sigs.k8s.io/kubefed/pkg/apis/core/v1alpha1"
	"sigs.k8s.io/kubefed/pkg/client/generic"
)

const (
	DefaultKubefedSystemNamespace  = "kube-federation-system"
	DefaultClusterAvailableDelay   = 20 * time.Second
	DefaultClusterUnavailableDelay = 60 * time.Second

	KubeAPIQPS   = 20.0
	KubeAPIBurst = 30

	DefaultLeaderElectionLeaseDuration = 15 * time.Second
	DefaultLeaderElectionRenewDeadline = 10 * time.Second
	DefaultLeaderElectionRetryPeriod   = 5 * time.Second
	DefaultLeaderElectionResourceLock  = "configmaps"

	DefaultClusterHealthCheckPeriod           = 10
	DefaultClusterHealthCheckFailureThreshold = 3
	DefaultClusterHealthCheckSuccessThreshold = 1
	DefaultClusterHealthCheckTimeout          = 3

	KubefedConfigName = "kubefed"
)

// BuildClusterConfig returns a restclient.Config that can be used to configure
// a client for the given KubefedCluster or an error. The client is used to
// access kubernetes secrets in the kubefed namespace.
func BuildClusterConfig(fedCluster *fedv1a1.KubefedCluster, client generic.Client, fedNamespace string) (*restclient.Config, error) {
	clusterName := fedCluster.Name

	apiEndpoint := fedCluster.Spec.APIEndpoint
	// TODO(marun) Remove when validation ensures a non-empty value.
	if apiEndpoint == "" {
		return nil, errors.Errorf("The api endpoint of cluster %s is empty", clusterName)
	}

	secretName := fedCluster.Spec.SecretRef.Name
	if secretName == "" {
		return nil, errors.Errorf("Cluster %s does not have a secret name", clusterName)
	}
	secret := &apiv1.Secret{}
	err := client.Get(context.TODO(), secret, fedNamespace, secretName)
	if err != nil {
		return nil, err
	}

	tokenKey := "token"
	token, tokenFound := secret.Data[tokenKey]
	if !tokenFound {
		return nil, errors.Errorf("The secret for cluster %s is missing a value for %q", clusterName, tokenKey)
	}
	caKey := "ca.crt"
	ca, caFound := secret.Data[caKey]
	if !caFound {
		return nil, errors.Errorf("The secret for cluster %s is missing a value for %q", clusterName, caKey)
	}

	clusterConfig, err := clientcmd.BuildConfigFromFlags(apiEndpoint, "")
	if err != nil {
		return nil, err
	}
	clusterConfig.CAData = ca
	clusterConfig.BearerToken = string(token)
	clusterConfig.QPS = KubeAPIQPS
	clusterConfig.Burst = KubeAPIBurst

	return clusterConfig, nil
}

// IsPrimaryCluster checks if the caller is working with objects for the
// primary cluster by checking if the UIDs match for both ObjectMetas passed
// in.
// TODO (font): Need to revisit this when cluster ID is available.
func IsPrimaryCluster(obj, clusterObj pkgruntime.Object) bool {
	meta := MetaAccessor(obj)
	clusterMeta := MetaAccessor(clusterObj)
	return meta.GetUID() == clusterMeta.GetUID()
}
