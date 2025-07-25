package kube

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// DeleteNamespace attempts to delete a namespace and returns true if deleted, false if stuck in terminating, or error.
func DeleteNamespace(ctx context.Context, clientset kubernetes.Interface, name string) (deleted bool, terminating bool, err error) {
	ns, err := clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, false, err
	}
	if ns.Status.Phase == "Terminating" {
		return false, true, nil
	}
	err = clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return false, false, err
	}
	return true, false, nil
}

// ForceRemoveFinalizers removes finalizers from a namespace.
// Returns true if finalizers were removed, false if no finalizers existed.
func ForceRemoveFinalizers(ctx context.Context, clientset kubernetes.Interface, name string) (bool, error) {
	ns, err := clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	if len(ns.ObjectMeta.Finalizers) == 0 {
		return false, nil // No finalizers to remove, but not an error
	}
	ns.ObjectMeta.Finalizers = nil
	_, err = clientset.CoreV1().Namespaces().Finalize(ctx, ns, metav1.UpdateOptions{})
	return true, err
}

// NukeNamespace aggressively deletes a namespace by force-deleting all resources first
func NukeNamespace(ctx context.Context, clientset kubernetes.Interface, name string) error {
	fmt.Printf("💥 NUKE MODE: Aggressively deleting namespace %s and all its contents...\n", name)

	// First, force delete all pods with grace period 0
	if err := forceDeleteAllPods(ctx, clientset, name); err != nil {
		fmt.Printf("⚠️  Warning: Failed to force delete pods: %v\n", err)
	}

	// Force delete other common resources
	if err := forceDeleteCommonResources(ctx, clientset, name); err != nil {
		fmt.Printf("⚠️  Warning: Failed to delete some resources: %v\n", err)
	}

	// Now try to delete the namespace
	deleted, terminating, err := DeleteNamespace(ctx, clientset, name)
	if err != nil {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	if terminating || !deleted {
		fmt.Printf("🔧 Namespace stuck, attempting aggressive finalizer removal...\n")
		return aggressiveFinalizerRemoval(ctx, clientset, name)
	}

	return nil
}

// forceDeleteAllPods force deletes all pods in the namespace with grace period 0
func forceDeleteAllPods(ctx context.Context, clientset kubernetes.Interface, name string) error {
	pods, err := clientset.CoreV1().Pods(name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	if len(pods.Items) == 0 {
		return nil
	}

	fmt.Printf("🚀 Force deleting %d pods...\n", len(pods.Items))

	gracePeriod := int64(0)
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	for _, pod := range pods.Items {
		err := clientset.CoreV1().Pods(name).Delete(ctx, pod.Name, deleteOptions)
		if err != nil {
			fmt.Printf("⚠️  Failed to delete pod %s: %v\n", pod.Name, err)
		}
	}

	// Wait a bit for pods to be deleted
	time.Sleep(2 * time.Second)
	return nil
}

// forceDeleteCommonResources deletes common resources that might prevent namespace deletion
func forceDeleteCommonResources(ctx context.Context, clientset kubernetes.Interface, name string) error {
	gracePeriod := int64(0)
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	// Delete services
	services, err := clientset.CoreV1().Services(name).List(ctx, metav1.ListOptions{})
	if err == nil && len(services.Items) > 0 {
		fmt.Printf("🗑️  Deleting %d services...\n", len(services.Items))
		for _, svc := range services.Items {
			clientset.CoreV1().Services(name).Delete(ctx, svc.Name, deleteOptions)
		}
	}

	// Delete deployments
	deployments, err := clientset.AppsV1().Deployments(name).List(ctx, metav1.ListOptions{})
	if err == nil && len(deployments.Items) > 0 {
		fmt.Printf("🗑️  Deleting %d deployments...\n", len(deployments.Items))
		for _, deploy := range deployments.Items {
			clientset.AppsV1().Deployments(name).Delete(ctx, deploy.Name, deleteOptions)
		}
	}

	// Delete replicasets
	replicasets, err := clientset.AppsV1().ReplicaSets(name).List(ctx, metav1.ListOptions{})
	if err == nil && len(replicasets.Items) > 0 {
		fmt.Printf("🗑️  Deleting %d replicasets...\n", len(replicasets.Items))
		for _, rs := range replicasets.Items {
			clientset.AppsV1().ReplicaSets(name).Delete(ctx, rs.Name, deleteOptions)
		}
	}

	// Delete configmaps
	configmaps, err := clientset.CoreV1().ConfigMaps(name).List(ctx, metav1.ListOptions{})
	if err == nil && len(configmaps.Items) > 0 {
		fmt.Printf("🗑️  Deleting %d configmaps...\n", len(configmaps.Items))
		for _, cm := range configmaps.Items {
			clientset.CoreV1().ConfigMaps(name).Delete(ctx, cm.Name, deleteOptions)
		}
	}

	// Delete secrets
	secrets, err := clientset.CoreV1().Secrets(name).List(ctx, metav1.ListOptions{})
	if err == nil && len(secrets.Items) > 0 {
		fmt.Printf("🗑️  Deleting %d secrets...\n", len(secrets.Items))
		for _, secret := range secrets.Items {
			clientset.CoreV1().Secrets(name).Delete(ctx, secret.Name, deleteOptions)
		}
	}

	// Delete custom resources that might be preventing namespace deletion
	if err := forceDeleteCustomResources(ctx, clientset, name); err != nil {
		fmt.Printf("⚠️  Warning: Failed to delete some custom resources: %v\n", err)
	}

	return nil
}

// aggressiveFinalizerRemoval uses multiple strategies to remove finalizers
func aggressiveFinalizerRemoval(ctx context.Context, clientset kubernetes.Interface, name string) error {
	// Try the standard finalizer removal first
	removed, err := ForceRemoveFinalizers(ctx, clientset, name)
	if err == nil && removed {
		fmt.Printf("🔧 Standard finalizer removal successful\n")
		return nil
	}
	if err != nil {
		fmt.Printf("⚠️  Standard finalizer removal failed: %v\n", err)
	} else if !removed {
		fmt.Printf("ℹ️  No finalizers found to remove\n")
	}

	// If that fails, try a more aggressive approach using patch
	fmt.Printf("🔧 Standard finalizer removal failed, trying aggressive patch...\n")

	// Create a patch to remove all finalizers
	patch := `{"metadata":{"finalizers":null}}`

	_, err = clientset.CoreV1().Namespaces().Patch(
		ctx,
		name,
		types.MergePatchType,
		[]byte(patch),
		metav1.PatchOptions{},
		"finalize",
	)

	if err != nil {
		// Last resort: try to patch the namespace spec directly
		fmt.Printf("🔧 Patch failed, trying direct spec modification...\n")
		return forceRemoveFinalizersDirectly(ctx, clientset, name)
	}

	return nil
}

// forceRemoveFinalizersDirectly attempts to directly modify the namespace
func forceRemoveFinalizersDirectly(ctx context.Context, clientset kubernetes.Interface, name string) error {
	ns, err := clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Clear all finalizers
	ns.ObjectMeta.Finalizers = []string{}
	ns.Spec.Finalizers = []corev1.FinalizerName{}

	// Try to update the namespace
	_, err = clientset.CoreV1().Namespaces().Update(ctx, ns, metav1.UpdateOptions{})
	if err != nil {
		// If update fails, try finalize
		_, err = clientset.CoreV1().Namespaces().Finalize(ctx, ns, metav1.UpdateOptions{})
	}

	return err
}

// ForceDeletePods force deletes specific pods by name with grace period 0
func ForceDeletePods(ctx context.Context, clientset kubernetes.Interface, namespace string, podNames []string) error {
	gracePeriod := int64(0)
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	var errors []string
	successCount := 0

	for _, podName := range podNames {
		fmt.Printf("🚀 Force deleting pod: %s\n", podName)

		// Check if pod exists first
		_, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("⚠️  Pod %s not found: %v\n", podName, err)
			errors = append(errors, fmt.Sprintf("pod %s not found: %v", podName, err))
			continue
		}

		// Force delete the pod
		err = clientset.CoreV1().Pods(namespace).Delete(ctx, podName, deleteOptions)
		if err != nil {
			fmt.Printf("❌ Failed to delete pod %s: %v\n", podName, err)
			errors = append(errors, fmt.Sprintf("failed to delete pod %s: %v", podName, err))
		} else {
			fmt.Printf("✅ Force delete request sent for pod: %s\n", podName)
			successCount++
		}
	}

	fmt.Printf("📊 Summary: %d/%d pods processed successfully\n", successCount, len(podNames))

	if len(errors) > 0 {
		return fmt.Errorf("some pods failed to delete: %v", errors)
	}

	return nil
}

// forceDeleteCustomResources discovers and force deletes custom resources in a namespace
// This is specifically designed to handle complex cases like SignOz with ClickHouse installations
func forceDeleteCustomResources(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	// We need to get the REST config to create discovery and dynamic clients
	// Since we can't easily extract it from the clientset, we'll try different approaches
	var config *rest.Config
	var err error

	// Try in-cluster config first
	config, err = rest.InClusterConfig()
	if err != nil {
		// If not in cluster, try to build from default kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", "")
		if err != nil {
			// Try with explicit kubeconfig path
			homeDir := os.Getenv("HOME")
			if homeDir == "" {
				homeDir = os.Getenv("USERPROFILE") // Windows
			}
			if homeDir != "" {
				kubeconfigPath := filepath.Join(homeDir, ".kube", "config")
				config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
			}
			if err != nil {
				// Skip custom resource deletion if we can't get config
				fmt.Printf("⚠️  Could not get REST config for custom resource deletion, skipping...\n")
				return nil
			}
		}
	}

	// Create discovery client to find all API resources
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		fmt.Printf("⚠️  Could not create discovery client: %v\n", err)
		return nil
	}

	// Create dynamic client for custom resource operations
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		fmt.Printf("⚠️  Could not create dynamic client: %v\n", err)
		return nil
	}

	// Get all API resources
	apiResourceLists, err := discoveryClient.ServerPreferredNamespacedResources()
	if err != nil {
		fmt.Printf("⚠️  Could not discover API resources: %v\n", err)
		return nil
	}

	customResourcesFound := 0
	customResourcesDeleted := 0

	// Look for custom resources (non-core Kubernetes resources)
	for _, apiResourceList := range apiResourceLists {
		// Skip core Kubernetes APIs and common extensions
		if strings.Contains(apiResourceList.GroupVersion, "/v1") ||
			strings.Contains(apiResourceList.GroupVersion, "apps/") ||
			strings.Contains(apiResourceList.GroupVersion, "extensions/") ||
			strings.Contains(apiResourceList.GroupVersion, "networking.k8s.io/") ||
			strings.Contains(apiResourceList.GroupVersion, "policy/") ||
			strings.Contains(apiResourceList.GroupVersion, "rbac.authorization.k8s.io/") {
			continue
		}

		gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			continue
		}

		for _, apiResource := range apiResourceList.APIResources {
			// Skip subresources
			if strings.Contains(apiResource.Name, "/") {
				continue
			}

			// Only process namespaced resources that support delete
			if !apiResource.Namespaced {
				continue
			}

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

			// Create GVR (GroupVersionResource)
			gvr := schema.GroupVersionResource{
				Group:    gv.Group,
				Version:  gv.Version,
				Resource: apiResource.Name,
			}

			// List resources of this type in the namespace
			resourceList, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				// Skip resources we can't list (permissions, etc.)
				continue
			}

			if len(resourceList.Items) > 0 {
				customResourcesFound += len(resourceList.Items)
				fmt.Printf("🔍 Found %d %s resources in namespace %s\n", len(resourceList.Items), apiResource.Name, namespace)

				// Force delete each resource
				gracePeriod := int64(0)
				deleteOptions := metav1.DeleteOptions{
					GracePeriodSeconds: &gracePeriod,
				}

				for _, resource := range resourceList.Items {
					resourceName := resource.GetName()
					fmt.Printf("💥 Force deleting %s: %s\n", apiResource.Name, resourceName)

					// First, try to remove finalizers if they exist
					if finalizers := resource.GetFinalizers(); len(finalizers) > 0 {
						fmt.Printf("🔧 Removing finalizers from %s: %s\n", apiResource.Name, resourceName)
						resource.SetFinalizers([]string{})
						_, err := dynamicClient.Resource(gvr).Namespace(namespace).Update(ctx, &resource, metav1.UpdateOptions{})
						if err != nil {
							fmt.Printf("⚠️  Failed to remove finalizers from %s: %v\n", resourceName, err)
						}
					}

					// Now force delete the resource
					err := dynamicClient.Resource(gvr).Namespace(namespace).Delete(ctx, resourceName, deleteOptions)
					if err != nil {
						fmt.Printf("⚠️  Failed to delete %s %s: %v\n", apiResource.Name, resourceName, err)
					} else {
						customResourcesDeleted++
						fmt.Printf("✅ Successfully deleted %s: %s\n", apiResource.Name, resourceName)
					}
				}
			}
		}
	}

	if customResourcesFound > 0 {
		fmt.Printf("📊 Custom resources summary: %d found, %d deleted\n", customResourcesFound, customResourcesDeleted)
		// Wait a bit for custom resources to be processed
		time.Sleep(3 * time.Second)
	} else {
		fmt.Printf("ℹ️  No custom resources found in namespace %s\n", namespace)
	}

	return nil
}

// WaitForNamespaceDeletion waits for a namespace to be completely deleted
func WaitForNamespaceDeletion(ctx context.Context, clientset kubernetes.Interface, name string, maxWaitSeconds int) bool {
	fmt.Printf("⏳ Waiting for namespace %s to be completely deleted...\n", name)

	for i := 0; i < maxWaitSeconds; i++ {
		time.Sleep(1 * time.Second)

		_, err := clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			// Namespace is gone
			fmt.Printf("✅ Namespace %s has been completely nuked!\n", name)
			return true
		}

		if i%5 == 0 && i > 0 {
			fmt.Printf("⏳ Still waiting... (%d/%d seconds)\n", i, maxWaitSeconds)
		}
	}

	fmt.Printf("⚠️  Namespace %s still exists after %d seconds\n", name, maxWaitSeconds)
	return false
}
