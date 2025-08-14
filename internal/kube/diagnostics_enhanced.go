package kube

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// EnhancedDiagnoseNamespace provides detailed diagnostics with ArgoCD awareness
// This function is kept for backward compatibility but now delegates to the CRD-aware version
func EnhancedDiagnoseNamespace(
	ctx context.Context, 
	clientset kubernetes.Interface, 
	dynamicClient dynamic.Interface,
	namespace string,
	argoCDApps []unstructured.Unstructured,
) error {
	fmt.Printf("🔍 Running enhanced diagnostics on namespace: %s\n", namespace)

	// Discover problematic CRDs
	crdDiscoveryResult, err := DiscoverProblematicCRDs(ctx, clientset, namespace)
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to discover problematic CRDs: %v\n", err)
		crdDiscoveryResult = &CRDDiscoveryResult{} // Continue with empty result
	}

	// Use the new CRD-aware diagnostics function
	return EnhancedDiagnoseNamespaceWithCRDs(ctx, clientset, dynamicClient, namespace, argoCDApps, crdDiscoveryResult)
}
