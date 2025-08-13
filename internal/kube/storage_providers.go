package kube

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// StorageProviderHandler handles specific storage provider resources
func HandleStorageProviderResources(ctx context.Context, clientset kubernetes.Interface, namespace string, config *rest.Config) error {
	fmt.Printf("🔍 Checking for storage provider resources in namespace %s...\n", namespace)

	// Create discovery client
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Check for Longhorn resources
	if err := handleLonghornResources(ctx, discoveryClient, dynamicClient, namespace); err != nil {
		fmt.Printf("⚠️  Error handling Longhorn resources: %v\n", err)
	}

	// Check for Rook-Ceph resources
	if err := handleRookCephResources(ctx, discoveryClient, dynamicClient, namespace); err != nil {
		fmt.Printf("⚠️  Error handling Rook-Ceph resources: %v\n", err)
	}

	// Check for OpenEBS resources
	if err := handleOpenEBSResources(ctx, discoveryClient, dynamicClient, namespace); err != nil {
		fmt.Printf("⚠️  Error handling OpenEBS resources: %v\n", err)
	}

	return nil
}

// handleLonghornResources specifically handles Longhorn resources
func handleLonghornResources(ctx context.Context, discoveryClient discovery.DiscoveryInterface, dynamicClient dynamic.Interface, namespace string) error {
	// Define Longhorn resource types to check
	longhornResources := []struct {
		group    string
		version  string
		resource string
	}{
		{"longhorn.io", "v1beta2", "volumes"},
		{"longhorn.io", "v1beta2", "replicas"},
		{"longhorn.io", "v1beta2", "engines"},
		{"longhorn.io", "v1beta2", "instancemanagers"},
		{"longhorn.io", "v1beta2", "nodes"},
		{"longhorn.io", "v1beta2", "volumeattachments"},
		{"longhorn.io", "v1beta2", "snapshots"},
		{"longhorn.io", "v1beta1", "volumes"},
		{"longhorn.io", "v1beta1", "replicas"},
		{"longhorn.io", "v1beta1", "engines"},
		{"longhorn.io", "v1beta1", "instancemanagers"},
		{"longhorn.io", "v1beta1", "nodes"},
		{"longhorn.io", "v1beta1", "volumeattachments"},
		{"longhorn.io", "v1beta1", "snapshots"},
	}

	longhornFound := false
	resourcesProcessed := 0

	// Check each Longhorn resource type
	for _, res := range longhornResources {
		gvr := schema.GroupVersionResource{
			Group:    res.group,
			Version:  res.version,
			Resource: res.resource,
		}

		// Try to list resources of this type
		list, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			// Skip if resource not found or other error
			continue
		}

		if len(list.Items) > 0 {
			longhornFound = true
			fmt.Printf("🔍 Found %d Longhorn %s resources\n", len(list.Items), res.resource)

			// Process each resource
			for _, item := range list.Items {
				resourcesProcessed++
				fmt.Printf("🔧 Processing Longhorn %s: %s\n", res.resource, item.GetName())

				// First, try to remove finalizers
				if finalizers := item.GetFinalizers(); len(finalizers) > 0 {
					fmt.Printf("🔧 Removing finalizers from %s: %s\n", res.resource, item.GetName())
					
					// Try patch method first (most reliable)
					patchData := map[string]interface{}{
						"metadata": map[string]interface{}{
							"finalizers": nil,
						},
					}
					patchBytes, _ := json.Marshal(patchData)
					
					_, err := dynamicClient.Resource(gvr).Namespace(namespace).Patch(
						ctx,
						item.GetName(),
						types.MergePatchType,
						patchBytes,
						metav1.PatchOptions{},
					)
					
					if err != nil {
						fmt.Printf("⚠️  Failed to patch finalizers for %s: %v\n", item.GetName(), err)
						
						// Try update method as fallback
						item.SetFinalizers([]string{})
						_, err = dynamicClient.Resource(gvr).Namespace(namespace).Update(
							ctx,
							&item,
							metav1.UpdateOptions{},
						)
						
						if err != nil {
							fmt.Printf("⚠️  Failed to update finalizers for %s: %v\n", item.GetName(), err)
						}
					}
				}

				// Then force delete the resource
				gracePeriod := int64(0)
				err = dynamicClient.Resource(gvr).Namespace(namespace).Delete(
					ctx,
					item.GetName(),
					metav1.DeleteOptions{GracePeriodSeconds: &gracePeriod},
				)
				
				if err != nil {
					fmt.Printf("⚠️  Failed to delete %s %s: %v\n", res.resource, item.GetName(), err)
				} else {
					fmt.Printf("✅ Successfully deleted %s: %s\n", res.resource, item.GetName())
				}
			}
		}
	}

	if longhornFound {
		fmt.Printf("📊 Processed %d Longhorn resources\n", resourcesProcessed)
		fmt.Printf("💡 Tip: Longhorn resources often have finalizers that prevent deletion\n")
		fmt.Printf("⏳ Waiting for Longhorn resources to be processed...\n")
		time.Sleep(5 * time.Second)
	}

	return nil
}

// handleRookCephResources specifically handles Rook-Ceph resources
func handleRookCephResources(ctx context.Context, discoveryClient discovery.DiscoveryInterface, dynamicClient dynamic.Interface, namespace string) error {
	// Define Rook-Ceph resource types to check
	rookResources := []struct {
		group    string
		version  string
		resource string
	}{
		{"ceph.rook.io", "v1", "cephclusters"},
		{"ceph.rook.io", "v1", "cephblockpools"},
		{"ceph.rook.io", "v1", "cephfilesystems"},
		{"ceph.rook.io", "v1", "cephobjectstores"},
		{"ceph.rook.io", "v1", "cephobjectstoreusers"},
	}

	rookFound := false
	resourcesProcessed := 0

	// Check each Rook-Ceph resource type
	for _, res := range rookResources {
		gvr := schema.GroupVersionResource{
			Group:    res.group,
			Version:  res.version,
			Resource: res.resource,
		}

		// Try to list resources of this type
		list, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			// Skip if resource not found or other error
			continue
		}

		if len(list.Items) > 0 {
			rookFound = true
			fmt.Printf("🔍 Found %d Rook-Ceph %s resources\n", len(list.Items), res.resource)

			// Process each resource
			for _, item := range list.Items {
				resourcesProcessed++
				fmt.Printf("🔧 Processing Rook-Ceph %s: %s\n", res.resource, item.GetName())

				// Remove finalizers
				if finalizers := item.GetFinalizers(); len(finalizers) > 0 {
					fmt.Printf("🔧 Removing finalizers from %s: %s\n", res.resource, item.GetName())
					
					// Try patch method
					patchData := map[string]interface{}{
						"metadata": map[string]interface{}{
							"finalizers": nil,
						},
					}
					patchBytes, _ := json.Marshal(patchData)
					
					_, err := dynamicClient.Resource(gvr).Namespace(namespace).Patch(
						ctx,
						item.GetName(),
						types.MergePatchType,
						patchBytes,
						metav1.PatchOptions{},
					)
					
					if err != nil {
						fmt.Printf("⚠️  Failed to patch finalizers for %s: %v\n", item.GetName(), err)
						
						// Try update method as fallback
						item.SetFinalizers([]string{})
						_, err = dynamicClient.Resource(gvr).Namespace(namespace).Update(
							ctx,
							&item,
							metav1.UpdateOptions{},
						)
						
						if err != nil {
							fmt.Printf("⚠️  Failed to update finalizers for %s: %v\n", item.GetName(), err)
						}
					}
				}

				// Force delete the resource
				gracePeriod := int64(0)
				err = dynamicClient.Resource(gvr).Namespace(namespace).Delete(
					ctx,
					item.GetName(),
					metav1.DeleteOptions{GracePeriodSeconds: &gracePeriod},
				)
				
				if err != nil {
					fmt.Printf("⚠️  Failed to delete %s %s: %v\n", res.resource, item.GetName(), err)
				} else {
					fmt.Printf("✅ Successfully deleted %s: %s\n", res.resource, item.GetName())
				}
			}
		}
	}

	if rookFound {
		fmt.Printf("📊 Processed %d Rook-Ceph resources\n", resourcesProcessed)
		fmt.Printf("⏳ Waiting for Rook-Ceph resources to be processed...\n")
		time.Sleep(3 * time.Second)
	}

	return nil
}

// handleOpenEBSResources specifically handles OpenEBS resources
func handleOpenEBSResources(ctx context.Context, discoveryClient discovery.DiscoveryInterface, dynamicClient dynamic.Interface, namespace string) error {
	// Define OpenEBS resource types to check
	openebsResources := []struct {
		group    string
		version  string
		resource string
	}{
		{"openebs.io", "v1alpha1", "blockdevices"},
		{"openebs.io", "v1alpha1", "blockdeviceclaims"},
		{"openebs.io", "v1alpha1", "cstorvolumes"},
		{"openebs.io", "v1alpha1", "cstorvolumeclaims"},
		{"openebs.io", "v1alpha1", "cstorvolumereplicas"},
	}

	openebsFound := false
	resourcesProcessed := 0

	// Check each OpenEBS resource type
	for _, res := range openebsResources {
		gvr := schema.GroupVersionResource{
			Group:    res.group,
			Version:  res.version,
			Resource: res.resource,
		}

		// Try to list resources of this type
		list, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			// Skip if resource not found or other error
			continue
		}

		if len(list.Items) > 0 {
			openebsFound = true
			fmt.Printf("🔍 Found %d OpenEBS %s resources\n", len(list.Items), res.resource)

			// Process each resource
			for _, item := range list.Items {
				resourcesProcessed++
				fmt.Printf("🔧 Processing OpenEBS %s: %s\n", res.resource, item.GetName())

				// Remove finalizers
				if finalizers := item.GetFinalizers(); len(finalizers) > 0 {
					fmt.Printf("🔧 Removing finalizers from %s: %s\n", res.resource, item.GetName())
					
					// Try patch method
					patchData := map[string]interface{}{
						"metadata": map[string]interface{}{
							"finalizers": nil,
						},
					}
					patchBytes, _ := json.Marshal(patchData)
					
					_, err := dynamicClient.Resource(gvr).Namespace(namespace).Patch(
						ctx,
						item.GetName(),
						types.MergePatchType,
						patchBytes,
						metav1.PatchOptions{},
					)
					
					if err != nil {
						fmt.Printf("⚠️  Failed to patch finalizers for %s: %v\n", item.GetName(), err)
						
						// Try update method as fallback
						item.SetFinalizers([]string{})
						_, err = dynamicClient.Resource(gvr).Namespace(namespace).Update(
							ctx,
							&item,
							metav1.UpdateOptions{},
						)
						
						if err != nil {
							fmt.Printf("⚠️  Failed to update finalizers for %s: %v\n", item.GetName(), err)
						}
					}
				}

				// Force delete the resource
				gracePeriod := int64(0)
				err = dynamicClient.Resource(gvr).Namespace(namespace).Delete(
					ctx,
					item.GetName(),
					metav1.DeleteOptions{GracePeriodSeconds: &gracePeriod},
				)
				
				if err != nil {
					fmt.Printf("⚠️  Failed to delete %s %s: %v\n", res.resource, item.GetName(), err)
				} else {
					fmt.Printf("✅ Successfully deleted %s: %s\n", res.resource, item.GetName())
				}
			}
		}
	}

	if openebsFound {
		fmt.Printf("📊 Processed %d OpenEBS resources\n", resourcesProcessed)
		fmt.Printf("⏳ Waiting for OpenEBS resources to be processed...\n")
		time.Sleep(3 * time.Second)
	}

	return nil
}

// DetectStorageProviderResources detects common storage provider issues
func DetectStorageProviderResources(ctx context.Context, clientset kubernetes.Interface) error {
	// Check for storage classes that might be causing issues
	storageClasses, err := clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list storage classes: %w", err)
	}

	// Check for known problematic storage providers
	for _, sc := range storageClasses.Items {
		provisioner := sc.Provisioner
		
		if strings.Contains(provisioner, "longhorn.io") {
			fmt.Printf("🔍 Detected Longhorn storage class: %s\n", sc.Name)
			fmt.Printf("💡 Tip: Longhorn resources often have finalizers that prevent deletion\n")
		} else if strings.Contains(provisioner, "rook.io") || strings.Contains(provisioner, "ceph.rook.io") {
			fmt.Printf("🔍 Detected Rook-Ceph storage class: %s\n", sc.Name)
			fmt.Printf("💡 Tip: Rook-Ceph resources often have finalizers that prevent deletion\n")
		} else if strings.Contains(provisioner, "openebs.io") {
			fmt.Printf("🔍 Detected OpenEBS storage class: %s\n", sc.Name)
			fmt.Printf("💡 Tip: OpenEBS resources often have finalizers that prevent deletion\n")
		}
	}

	return nil
}

// RemoveAllCustomResourceFinalizers aggressively removes finalizers from all custom resources in a namespace
func RemoveAllCustomResourceFinalizers(ctx context.Context, config *rest.Config, namespace string) error {
	fmt.Printf("💥 Aggressively removing finalizers from all custom resources in namespace %s...\n", namespace)

	// Create discovery client
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Get all API resources
	apiResourceLists, err := discoveryClient.ServerPreferredNamespacedResources()
	if err != nil {
		return fmt.Errorf("failed to discover API resources: %w", err)
	}

	resourcesProcessed := 0
	finalizersRemoved := 0

	// Process all resource types
	for _, apiResourceList := range apiResourceLists {
		gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			continue
		}

		for _, apiResource := range apiResourceList.APIResources {
			// Skip subresources and non-namespaced resources
			if strings.Contains(apiResource.Name, "/") || !apiResource.Namespaced {
				continue
			}

			// Check if resource supports get and update operations
			canGet := false
			canUpdate := false
			for _, verb := range apiResource.Verbs {
				if verb == "get" {
					canGet = true
				}
				if verb == "update" || verb == "patch" {
					canUpdate = true
				}
			}
			if !canGet || !canUpdate {
				continue
			}

			// Create GVR
			gvr := schema.GroupVersionResource{
				Group:    gv.Group,
				Version:  gv.Version,
				Resource: apiResource.Name,
			}

			// List resources of this type
			list, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				continue
			}

			// Process each resource
			for _, item := range list.Items {
				resourcesProcessed++
				
				// Check for finalizers
				finalizers := item.GetFinalizers()
				if len(finalizers) > 0 {
					fmt.Printf("🔧 Removing finalizers from %s/%s: %v\n", apiResource.Name, item.GetName(), finalizers)
					
					// Try patch method first
					patchData := map[string]interface{}{
						"metadata": map[string]interface{}{
							"finalizers": nil,
						},
					}
					patchBytes, _ := json.Marshal(patchData)
					
					_, err := dynamicClient.Resource(gvr).Namespace(namespace).Patch(
						ctx,
						item.GetName(),
						types.MergePatchType,
						patchBytes,
						metav1.PatchOptions{},
					)
					
					if err != nil {
						// Try update method as fallback
						item.SetFinalizers([]string{})
						_, err = dynamicClient.Resource(gvr).Namespace(namespace).Update(
							ctx,
							&item,
							metav1.UpdateOptions{},
						)
						
						if err != nil {
							fmt.Printf("⚠️  Failed to remove finalizers from %s/%s: %v\n", apiResource.Name, item.GetName(), err)
						} else {
							finalizersRemoved++
						}
					} else {
						finalizersRemoved++
					}
				}
			}
		}
	}

	fmt.Printf("📊 Processed %d resources, removed finalizers from %d resources\n", resourcesProcessed, finalizersRemoved)
	return nil
}

// ForceDeleteAllCustomResources aggressively deletes all custom resources in a namespace
func ForceDeleteAllCustomResources(ctx context.Context, config *rest.Config, namespace string) error {
	fmt.Printf("💥 Aggressively deleting all custom resources in namespace %s...\n", namespace)

	// Create discovery client
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Get all API resources
	apiResourceLists, err := discoveryClient.ServerPreferredNamespacedResources()
	if err != nil {
		return fmt.Errorf("failed to discover API resources: %w", err)
	}

	resourcesProcessed := 0
	resourcesDeleted := 0

	// Process all resource types
	for _, apiResourceList := range apiResourceLists {
		// Skip core Kubernetes APIs
		if strings.Contains(apiResourceList.GroupVersion, "/v1") && 
		   !strings.Contains(apiResourceList.GroupVersion, ".") {
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

			// Check if resource supports delete operation
			canDelete := false
			for _, verb := range apiResource.Verbs {
				if verb == "delete" {
					canDelete = true
					break
				}
			}
			if !canDelete {
				continue
			}

			// Create GVR
			gvr := schema.GroupVersionResource{
				Group:    gv.Group,
				Version:  gv.Version,
				Resource: apiResource.Name,
			}

			// List resources of this type
			list, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				continue
			}

			if len(list.Items) > 0 {
				fmt.Printf("🔍 Found %d %s resources\n", len(list.Items), apiResource.Name)
				
				// Process each resource
				for _, item := range list.Items {
					resourcesProcessed++
					fmt.Printf("🔧 Processing %s: %s\n", apiResource.Name, item.GetName())
					
					// First remove finalizers
					if finalizers := item.GetFinalizers(); len(finalizers) > 0 {
						fmt.Printf("🔧 Removing finalizers from %s: %s\n", apiResource.Name, item.GetName())
						
						// Try patch method
						patchData := map[string]interface{}{
							"metadata": map[string]interface{}{
								"finalizers": nil,
							},
						}
						patchBytes, _ := json.Marshal(patchData)
						
						_, err := dynamicClient.Resource(gvr).Namespace(namespace).Patch(
							ctx,
							item.GetName(),
							types.MergePatchType,
							patchBytes,
							metav1.PatchOptions{},
						)
						
						if err != nil {
							fmt.Printf("⚠️  Failed to patch finalizers for %s: %v\n", item.GetName(), err)
							
							// Try update method as fallback
							item.SetFinalizers([]string{})
							_, err = dynamicClient.Resource(gvr).Namespace(namespace).Update(
								ctx,
								&item,
								metav1.UpdateOptions{},
							)
							
							if err != nil {
								fmt.Printf("⚠️  Failed to update finalizers for %s: %v\n", item.GetName(), err)
							}
						}
					}
					
					// Then force delete
					gracePeriod := int64(0)
					err = dynamicClient.Resource(gvr).Namespace(namespace).Delete(
						ctx,
						item.GetName(),
						metav1.DeleteOptions{GracePeriodSeconds: &gracePeriod},
					)
					
					if err != nil {
						fmt.Printf("⚠️  Failed to delete %s %s: %v\n", apiResource.Name, item.GetName(), err)
					} else {
						resourcesDeleted++
						fmt.Printf("✅ Successfully deleted %s: %s\n", apiResource.Name, item.GetName())
					}
				}
			}
		}
	}

	fmt.Printf("📊 Processed %d resources, deleted %d resources\n", resourcesProcessed, resourcesDeleted)
	return nil
}

// GetRESTConfig gets the Kubernetes REST config from the clientset or environment
func GetRESTConfig(clientset kubernetes.Interface) (*rest.Config, error) {
	// Since we can't easily extract the config from clientset, we'll try different approaches
	var config *rest.Config
	var err error

	// Try in-cluster config first
	config, err = rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Try to build from default kubeconfig
	config, err = clientcmd.BuildConfigFromFlags("", "")
	if err == nil {
		return config, nil
	}

	// Try with explicit kubeconfig path
	homeDir, err := os.UserHomeDir()
	if err == nil {
		kubeconfigPath := filepath.Join(homeDir, ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err == nil {
			return config, nil
		}
	}

	return nil, fmt.Errorf("could not get REST config")
}
