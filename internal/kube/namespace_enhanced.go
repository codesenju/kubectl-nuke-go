package kube

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/codesenju/kubectl-nuke-go/pkg/argocd"
)

// EnhancedDeleteNamespace provides ArgoCD-aware namespace deletion
func EnhancedDeleteNamespace(ctx context.Context, clientset kubernetes.Interface, namespace string, forceDelete bool, diagnoseOnly bool) error {
	// Get REST config for dynamic client operations
	config, err := GetRESTConfig(clientset)
	if err != nil {
		return fmt.Errorf("failed to get REST config: %w", err)
	}

	// Create dynamic client for ArgoCD operations
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Create ArgoCD detector and handler
	detector := argocd.NewDetector(clientset, dynamicClient)
	handler := argocd.NewHandler(dynamicClient)

	// Phase 1: Detect ArgoCD applications managing this namespace
	fmt.Printf("🔍 Checking for ArgoCD applications managing namespace: %s\n", namespace)
	argoCDApps, err := detector.DetectArgoCDAppsForNamespace(ctx, namespace)
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to detect ArgoCD applications: %v\n", err)
	}

	if len(argoCDApps) > 0 {
		fmt.Printf("🎯 Found %d ArgoCD application(s) managing this namespace:\n", len(argoCDApps))
		for _, app := range argoCDApps {
			fmt.Printf("  - %s/%s\n", app.GetNamespace(), app.GetName())
		}
	} else {
		fmt.Printf("ℹ️  No ArgoCD applications found managing this namespace\n")
	}

	// Phase 2: Enhanced diagnostics with ArgoCD awareness
	if diagnoseOnly {
		return EnhancedDiagnoseNamespace(ctx, clientset, dynamicClient, namespace, argoCDApps)
	}

	// Phase 3: Handle ArgoCD applications first (if any)
	if len(argoCDApps) > 0 {
		fmt.Printf("🔄 Handling ArgoCD applications before namespace deletion...\n")
		
		// Delete ArgoCD applications first
		if err := handler.DeleteApplications(ctx, argoCDApps); err != nil {
			fmt.Printf("⚠️  Warning: Failed to delete some ArgoCD applications: %v\n", err)
		}

		// Wait a bit for ArgoCD to clean up resources
		fmt.Printf("⏳ Waiting for ArgoCD to clean up resources...\n")
		time.Sleep(10 * time.Second)
	}

	// Phase 4: Proceed with namespace deletion based on mode
	if forceDelete {
		return EnhancedNukeNamespace(ctx, clientset, dynamicClient, namespace, detector)
	}
	return EnhancedStandardDelete(ctx, clientset, namespace)
}

// EnhancedNukeNamespace performs aggressive namespace deletion with ArgoCD awareness
func EnhancedNukeNamespace(ctx context.Context, clientset kubernetes.Interface, dynamicClient dynamic.Interface, namespace string, detector *argocd.Detector) error {
	fmt.Printf("💥 ENHANCED NUKE MODE: ArgoCD-aware aggressive deletion of namespace: %s\n", namespace)

	// Phase 1: Remove any remaining ArgoCD-managed resources with finalizers
	if err := removeArgoCDManagedResourceFinalizers(ctx, clientset, dynamicClient, namespace, detector); err != nil {
		fmt.Printf("⚠️  Warning: Failed to remove ArgoCD finalizers: %v\n", err)
	}

	// Phase 2: Continue with standard nuke process
	return NukeNamespace(ctx, clientset, namespace, false, false)
}

// EnhancedStandardDelete performs standard namespace deletion with ArgoCD awareness
func EnhancedStandardDelete(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	fmt.Printf("🔄 Enhanced standard deletion of namespace: %s\n", namespace)

	// Use existing standard deletion logic
	deleted, terminating, err := DeleteNamespace(ctx, clientset, namespace)
	if err != nil {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	if terminating {
		fmt.Printf("⚠️  Namespace %s is stuck in Terminating state. Attempting finalizer removal...\n", namespace)
		removed, err := ForceRemoveFinalizers(ctx, clientset, namespace)
		if err != nil {
			return fmt.Errorf("failed to remove finalizers: %w", err)
		}
		if removed {
			fmt.Printf("🔧 Finalizers removed for %s. Waiting for namespace to be deleted...\n", namespace)
		}
	}

	if deleted {
		fmt.Printf("📤 Delete request sent for namespace %s\n", namespace)
	}

	return nil
}

// removeArgoCDManagedResourceFinalizers removes finalizers from ArgoCD-managed resources
func removeArgoCDManagedResourceFinalizers(ctx context.Context, clientset kubernetes.Interface, dynamicClient dynamic.Interface, namespace string, detector *argocd.Detector) error {
	fmt.Printf("🔧 Removing finalizers from ArgoCD-managed resources...\n")

	// Get all resources in the namespace and check if they're ArgoCD-managed
	// This is a simplified approach - in a full implementation, you'd want to
	// iterate through all resource types dynamically

	// Check pods
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, pod := range pods.Items {
			podUnstructured := convertToUnstructured(&pod)
			if detector.IsArgoCDManagedResource(podUnstructured) {
				if len(pod.Finalizers) > 0 {
					fmt.Printf("🔧 Removing finalizers from ArgoCD-managed pod: %s\n", pod.Name)
					if err := removePodFinalizers(ctx, clientset, namespace, pod.Name); err != nil {
						fmt.Printf("⚠️  Warning: Failed to remove finalizers from pod %s: %v\n", pod.Name, err)
					}
				}
			}
		}
	}

	// Check PVCs
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, pvc := range pvcs.Items {
			pvcUnstructured := convertToUnstructured(&pvc)
			if detector.IsArgoCDManagedResource(pvcUnstructured) {
				if len(pvc.Finalizers) > 0 {
					fmt.Printf("🔧 Removing finalizers from ArgoCD-managed PVC: %s\n", pvc.Name)
					if err := removePVCFinalizers(ctx, clientset, namespace, pvc.Name); err != nil {
						fmt.Printf("⚠️  Warning: Failed to remove finalizers from PVC %s: %v\n", pvc.Name, err)
					}
				}
			}
		}
	}

	return nil
}

// Helper functions for removing finalizers from specific resource types
func removePodFinalizers(ctx context.Context, clientset kubernetes.Interface, namespace, name string) error {
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	
	pod.Finalizers = nil
	_, err = clientset.CoreV1().Pods(namespace).Update(ctx, pod, metav1.UpdateOptions{})
	return err
}

func removePVCFinalizers(ctx context.Context, clientset kubernetes.Interface, namespace, name string) error {
	pvc, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	
	pvc.Finalizers = nil
	_, err = clientset.CoreV1().PersistentVolumeClaims(namespace).Update(ctx, pvc, metav1.UpdateOptions{})
	return err
}

// convertToUnstructured converts a typed Kubernetes object to unstructured
// This is a simplified version - in production, you'd use proper conversion
func convertToUnstructured(obj interface{}) *unstructured.Unstructured {
	// This is a placeholder implementation
	// In a real implementation, you'd properly convert the object
	u := &unstructured.Unstructured{}
	
	// Extract metadata based on object type
	switch v := obj.(type) {
	case *corev1.Pod:
		u.SetName(v.Name)
		u.SetNamespace(v.Namespace)
		u.SetLabels(v.Labels)
		u.SetAnnotations(v.Annotations)
	case *corev1.PersistentVolumeClaim:
		u.SetName(v.Name)
		u.SetNamespace(v.Namespace)
		u.SetLabels(v.Labels)
		u.SetAnnotations(v.Annotations)
	}
	
	return u
}
