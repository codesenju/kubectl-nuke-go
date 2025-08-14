package kube

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// CRDDiscoveryResult contains information about CRDs causing namespace termination issues
type CRDDiscoveryResult struct {
	ProblematicCRDs []ProblematicCRD
	NamespaceStatus NamespaceConditionInfo
}

// ProblematicCRD represents a CRD that has resources with finalizers preventing namespace deletion
type ProblematicCRD struct {
	Name              string
	Group             string
	Version           string
	Kind              string
	ResourcesWithFinalizers []ResourceWithFinalizers
	TotalResources    int
}

// ResourceWithFinalizers represents a resource instance that has finalizers
type ResourceWithFinalizers struct {
	Name       string
	Finalizers []string
}

// NamespaceConditionInfo contains parsed information from namespace conditions
type NamespaceConditionInfo struct {
	HasFinalizersRemaining bool
	HasResourcesRemaining  bool
	FinalizersMessage      string
	ResourcesMessage       string
	RawConditions          []corev1.NamespaceCondition
}

// DiscoverProblematicCRDs analyzes a namespace to find CRDs causing termination issues
func DiscoverProblematicCRDs(ctx context.Context, clientset kubernetes.Interface, namespace string) (*CRDDiscoveryResult, error) {
	fmt.Printf("🔍 Discovering CRDs causing namespace termination issues for: %s\n", namespace)

	// Get REST config for dynamic operations
	config, err := GetRESTConfig(clientset)
	if err != nil {
		return nil, fmt.Errorf("failed to get REST config: %w", err)
	}

	// Create dynamic and discovery clients
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	// Analyze namespace conditions
	nsConditions, err := analyzeNamespaceConditions(ctx, clientset, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze namespace conditions: %w", err)
	}

	// Discover all CRDs with resources that have finalizers
	problematicCRDs, err := findCRDsWithFinalizers(ctx, dynamicClient, discoveryClient, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to discover problematic CRDs: %w", err)
	}

	result := &CRDDiscoveryResult{
		ProblematicCRDs: problematicCRDs,
		NamespaceStatus: *nsConditions,
	}

	// Display results
	displayDiscoveryResults(result, namespace)

	return result, nil
}

// analyzeNamespaceConditions parses namespace conditions to extract finalizer and resource information
func analyzeNamespaceConditions(ctx context.Context, clientset kubernetes.Interface, namespace string) (*NamespaceConditionInfo, error) {
	ns, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	info := &NamespaceConditionInfo{
		RawConditions: ns.Status.Conditions,
	}

	// Parse condition messages for specific patterns
	finalizersPattern := regexp.MustCompile(`(?i).*finalizers?\s+remaining.*`)
	resourcesPattern := regexp.MustCompile(`(?i).*resources?\s+(?:are\s+)?remaining.*`)

	for _, condition := range ns.Status.Conditions {
		message := condition.Message
		
		if finalizersPattern.MatchString(message) {
			info.HasFinalizersRemaining = true
			info.FinalizersMessage = message
		}
		
		if resourcesPattern.MatchString(message) {
			info.HasResourcesRemaining = true
			info.ResourcesMessage = message
		}
	}

	return info, nil
}

// findCRDsWithFinalizers discovers all CRDs that have resources with finalizers in the namespace
func findCRDsWithFinalizers(ctx context.Context, dynamicClient dynamic.Interface, discoveryClient discovery.DiscoveryInterface, namespace string) ([]ProblematicCRD, error) {
	var problematicCRDs []ProblematicCRD

	// Get all API resources
	apiResourceLists, err := discoveryClient.ServerPreferredNamespacedResources()
	if err != nil {
		// Continue with partial results if some APIs are unavailable
		fmt.Printf("⚠️  Warning: Some API resources may not be accessible: %v\n", err)
	}

	fmt.Printf("🔍 Scanning custom resources for finalizers...\n")

	for _, apiResourceList := range apiResourceLists {
		// Skip core Kubernetes APIs - focus on custom resources
		if strings.Contains(apiResourceList.GroupVersion, "/v1") && !strings.Contains(apiResourceList.GroupVersion, ".") {
			continue
		}

		gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			continue
		}

		for _, apiResource := range apiResourceList.APIResources {
			// Skip subresources and non-namespaced resources
			if strings.Contains(apiResource.Name, "/") || !apiResource.Namespaced {
				continue
			}

			// Only check resources that support list and delete operations
			if !supportsVerb(apiResource.Verbs, "list") || !supportsVerb(apiResource.Verbs, "delete") {
				continue
			}

			gvr := schema.GroupVersionResource{
				Group:    gv.Group,
				Version:  gv.Version,
				Resource: apiResource.Name,
			}

			// Check this CRD for resources with finalizers
			problematicCRD, err := checkCRDForFinalizers(ctx, dynamicClient, gvr, apiResource, namespace)
			if err != nil {
				fmt.Printf("⚠️  Warning: Failed to check %s: %v\n", apiResource.Name, err)
				continue
			}

			if problematicCRD != nil {
				problematicCRDs = append(problematicCRDs, *problematicCRD)
			}
		}
	}

	return problematicCRDs, nil
}

// checkCRDForFinalizers checks a specific CRD for resources with finalizers
func checkCRDForFinalizers(ctx context.Context, dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, apiResource metav1.APIResource, namespace string) (*ProblematicCRD, error) {
	// List all resources of this type in the namespace
	resourceList, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(resourceList.Items) == 0 {
		return nil, nil
	}

	var resourcesWithFinalizers []ResourceWithFinalizers
	
	for _, resource := range resourceList.Items {
		finalizers := resource.GetFinalizers()
		if len(finalizers) > 0 {
			resourcesWithFinalizers = append(resourcesWithFinalizers, ResourceWithFinalizers{
				Name:       resource.GetName(),
				Finalizers: finalizers,
			})
		}
	}

	// Only return if there are resources with finalizers
	if len(resourcesWithFinalizers) == 0 {
		return nil, nil
	}

	return &ProblematicCRD{
		Name:                    apiResource.Name,
		Group:                   gvr.Group,
		Version:                 gvr.Version,
		Kind:                    apiResource.Kind,
		ResourcesWithFinalizers: resourcesWithFinalizers,
		TotalResources:          len(resourceList.Items),
	}, nil
}

// displayDiscoveryResults shows the discovery results in a user-friendly format
func displayDiscoveryResults(result *CRDDiscoveryResult, namespace string) {
	fmt.Printf("\n🔍 CRD DISCOVERY RESULTS FOR NAMESPACE: %s\n", namespace)
	fmt.Printf("================================================\n")

	// Display namespace condition analysis
	fmt.Printf("\n📊 NAMESPACE CONDITION ANALYSIS:\n")
	if result.NamespaceStatus.HasFinalizersRemaining {
		fmt.Printf("⚠️  Finalizers Remaining: %s\n", result.NamespaceStatus.FinalizersMessage)
	}
	if result.NamespaceStatus.HasResourcesRemaining {
		fmt.Printf("⚠️  Resources Remaining: %s\n", result.NamespaceStatus.ResourcesMessage)
	}

	if !result.NamespaceStatus.HasFinalizersRemaining && !result.NamespaceStatus.HasResourcesRemaining {
		fmt.Printf("✅ No specific finalizer or resource issues detected in namespace conditions\n")
	}

	// Display problematic CRDs
	fmt.Printf("\n🎯 PROBLEMATIC CRDS DISCOVERED:\n")
	if len(result.ProblematicCRDs) == 0 {
		fmt.Printf("✅ No CRDs with finalizers found in namespace %s\n", namespace)
		return
	}

	for i, crd := range result.ProblematicCRDs {
		fmt.Printf("\n%d. CRD: %s (Group: %s, Version: %s)\n", i+1, crd.Name, crd.Group, crd.Version)
		fmt.Printf("   Kind: %s\n", crd.Kind)
		fmt.Printf("   Total Resources: %d\n", crd.TotalResources)
		fmt.Printf("   Resources with Finalizers: %d\n", len(crd.ResourcesWithFinalizers))
		
		fmt.Printf("   📋 Resources with finalizers:\n")
		for _, resource := range crd.ResourcesWithFinalizers {
			fmt.Printf("     - %s: %v\n", resource.Name, resource.Finalizers)
		}
	}

	// Display recommendations
	fmt.Printf("\n💡 RECOMMENDATIONS:\n")
	fmt.Printf("==================\n")
	for i, crd := range result.ProblematicCRDs {
		fmt.Printf("\n%d. For CRD %s:\n", i+1, crd.Name)
		fmt.Printf("   a) Try to delete resources normally:\n")
		for _, resource := range crd.ResourcesWithFinalizers {
			fmt.Printf("      kubectl delete %s %s -n %s\n", crd.Name, resource.Name, namespace)
		}
		
		fmt.Printf("   b) If deletion fails, remove finalizers:\n")
		for _, resource := range crd.ResourcesWithFinalizers {
			fmt.Printf("      kubectl patch %s %s -n %s --type json -p '[{\"op\":\"remove\",\"path\":\"/metadata/finalizers\"}]'\n", 
				crd.Name, resource.Name, namespace)
		}
	}
	
	fmt.Printf("\n%d. Use kubectl-nuke with intelligent CRD handling:\n", len(result.ProblematicCRDs)+1)
	if result.NamespaceStatus.HasFinalizersRemaining || result.NamespaceStatus.HasResourcesRemaining {
		fmt.Printf("   # Standard mode (will auto-cleanup CRDs causing termination issues)\n")
		fmt.Printf("   kubectl-nuke ns %s\n", namespace)
		fmt.Printf("   \n")
	}
	fmt.Printf("   # Force mode (aggressively cleans up all CRDs with finalizers)\n")
	fmt.Printf("   kubectl-nuke ns %s --force\n", namespace)
}

// AttemptCRDCleanup attempts to clean up the discovered problematic CRDs
func AttemptCRDCleanup(ctx context.Context, clientset kubernetes.Interface, result *CRDDiscoveryResult, namespace string) error {
	if len(result.ProblematicCRDs) == 0 {
		fmt.Printf("✅ No problematic CRDs to clean up\n")
		return nil
	}

	fmt.Printf("🧹 Attempting to clean up %d problematic CRDs...\n", len(result.ProblematicCRDs))

	// Get REST config for dynamic operations
	config, err := GetRESTConfig(clientset)
	if err != nil {
		return fmt.Errorf("failed to get REST config: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	var cleanupErrors []string
	successfulCleanups := 0

	for _, crd := range result.ProblematicCRDs {
		fmt.Printf("\n🔧 Cleaning up CRD: %s\n", crd.Name)
		
		gvr := schema.GroupVersionResource{
			Group:    crd.Group,
			Version:  crd.Version,
			Resource: crd.Name,
		}

		// First attempt: Try to delete resources normally
		if err := attemptNormalDeletion(ctx, dynamicClient, gvr, crd, namespace); err != nil {
			fmt.Printf("⚠️  Normal deletion failed for %s: %v\n", crd.Name, err)
			
			// Second attempt: Remove finalizers and then delete
			if err := attemptFinalizerRemovalAndDeletion(ctx, dynamicClient, gvr, crd, namespace); err != nil {
				cleanupErrors = append(cleanupErrors, fmt.Sprintf("CRD %s: %v", crd.Name, err))
				fmt.Printf("❌ Failed to clean up CRD %s: %v\n", crd.Name, err)
			} else {
				successfulCleanups++
				fmt.Printf("✅ Successfully cleaned up CRD %s\n", crd.Name)
			}
		} else {
			successfulCleanups++
			fmt.Printf("✅ Successfully cleaned up CRD %s\n", crd.Name)
		}
	}

	fmt.Printf("\n📊 Cleanup Summary: %d/%d CRDs cleaned up successfully\n", successfulCleanups, len(result.ProblematicCRDs))

	if len(cleanupErrors) > 0 {
		return fmt.Errorf("some CRD cleanups failed: %v", cleanupErrors)
	}

	// Wait a bit for cleanup to propagate
	fmt.Printf("⏳ Waiting for cleanup to propagate...\n")
	time.Sleep(5 * time.Second)

	return nil
}

// attemptNormalDeletion tries to delete CRD resources normally
func attemptNormalDeletion(ctx context.Context, dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, crd ProblematicCRD, namespace string) error {
	fmt.Printf("🗑️  Attempting normal deletion of %s resources...\n", crd.Name)
	
	gracePeriod := int64(0)
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	for _, resource := range crd.ResourcesWithFinalizers {
		err := dynamicClient.Resource(gvr).Namespace(namespace).Delete(ctx, resource.Name, deleteOptions)
		if err != nil {
			return fmt.Errorf("failed to delete %s: %w", resource.Name, err)
		}
		fmt.Printf("🗑️  Deleted %s: %s\n", crd.Name, resource.Name)
	}

	return nil
}

// attemptFinalizerRemovalAndDeletion removes finalizers and then deletes resources
func attemptFinalizerRemovalAndDeletion(ctx context.Context, dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, crd ProblematicCRD, namespace string) error {
	fmt.Printf("🔧 Attempting finalizer removal and deletion for %s resources...\n", crd.Name)

	for _, resource := range crd.ResourcesWithFinalizers {
		// First, remove finalizers
		if err := removeCRDResourceFinalizers(ctx, dynamicClient, gvr, namespace, resource.Name); err != nil {
			return fmt.Errorf("failed to remove finalizers from %s: %w", resource.Name, err)
		}

		// Then delete the resource
		gracePeriod := int64(0)
		deleteOptions := metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
		}

		err := dynamicClient.Resource(gvr).Namespace(namespace).Delete(ctx, resource.Name, deleteOptions)
		if err != nil {
			return fmt.Errorf("failed to delete %s after finalizer removal: %w", resource.Name, err)
		}

		fmt.Printf("✅ Cleaned up %s: %s (finalizers removed + deleted)\n", crd.Name, resource.Name)
	}

	return nil
}

// removeCRDResourceFinalizers removes finalizers from a specific CRD resource
func removeCRDResourceFinalizers(ctx context.Context, dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, namespace, resourceName string) error {
	// Get the current resource
	resource, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, resourceName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Check if it has finalizers
	finalizers := resource.GetFinalizers()
	if len(finalizers) == 0 {
		return nil // No finalizers to remove
	}

	fmt.Printf("🔧 Removing finalizers from %s: %v\n", resourceName, finalizers)

	// Try patch method first (most reliable)
	patchData := `{"metadata":{"finalizers":null}}`
	_, err = dynamicClient.Resource(gvr).Namespace(namespace).Patch(
		ctx,
		resourceName,
		types.MergePatchType,
		[]byte(patchData),
		metav1.PatchOptions{},
	)

	if err != nil {
		// Fallback to update method
		resource.SetFinalizers([]string{})
		_, err = dynamicClient.Resource(gvr).Namespace(namespace).Update(ctx, resource, metav1.UpdateOptions{})
	}

	return err
}

// supportsVerb checks if an API resource supports a specific verb
func supportsVerb(verbs []string, verb string) bool {
	for _, v := range verbs {
		if v == verb {
			return true
		}
	}
	return false
}

// RetryNamespaceDeletion attempts to delete the namespace again after CRD cleanup
func RetryNamespaceDeletion(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	fmt.Printf("🔄 Retrying namespace deletion after CRD cleanup: %s\n", namespace)

	// First try standard deletion
	deleted, terminating, err := DeleteNamespace(ctx, clientset, namespace)
	if err != nil {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	if deleted && !terminating {
		fmt.Printf("✅ Namespace %s deleted successfully!\n", namespace)
		return nil
	}

	if terminating {
		fmt.Printf("⚠️  Namespace still stuck in Terminating state, attempting finalizer removal...\n")
		removed, err := ForceRemoveFinalizers(ctx, clientset, namespace)
		if err != nil {
			return fmt.Errorf("failed to remove namespace finalizers: %w", err)
		}
		
		if removed {
			fmt.Printf("🔧 Namespace finalizers removed, waiting for deletion...\n")
			// Wait for the namespace to be deleted
			if WaitForNamespaceDeletion(ctx, clientset, namespace, 30) {
				return nil
			}
		}
	}

	return fmt.Errorf("namespace %s is still not deleted after CRD cleanup and finalizer removal", namespace)
}
