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

// EnhancedDeleteNamespaceWithOptions provides ArgoCD-aware namespace deletion with intelligent CRD cleanup
func EnhancedDeleteNamespaceWithOptions(ctx context.Context, clientset kubernetes.Interface, namespace string, forceDelete bool, diagnoseOnly bool, aggressiveCRDCleanup bool) error {
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

	// Phase 2: Discover problematic CRDs (always run for diagnostics)
	fmt.Printf("\n🔍 Discovering CRDs that might be causing namespace termination issues...\n")
	crdDiscoveryResult, err := DiscoverProblematicCRDs(ctx, clientset, namespace)
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to discover problematic CRDs: %v\n", err)
		crdDiscoveryResult = &CRDDiscoveryResult{} // Continue with empty result
	}

	// Phase 3: Enhanced diagnostics with ArgoCD and CRD awareness
	if diagnoseOnly {
		return EnhancedDiagnoseNamespaceWithCRDs(ctx, clientset, dynamicClient, namespace, argoCDApps, crdDiscoveryResult)
	}

	// Phase 4: Handle ArgoCD applications first (if any)
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

	// Phase 5: Intelligent CRD cleanup based on mode
	shouldCleanupCRDs := false
	
	if forceDelete {
		// Force mode: Always cleanup CRDs if any are found with finalizers
		shouldCleanupCRDs = len(crdDiscoveryResult.ProblematicCRDs) > 0
		if shouldCleanupCRDs {
			fmt.Printf("\n💥 FORCE MODE: Aggressively cleaning up all problematic CRDs...\n")
		}
	} else {
		// Standard mode: Only cleanup CRDs if namespace conditions indicate they're causing issues
		shouldCleanupCRDs = (crdDiscoveryResult.NamespaceStatus.HasFinalizersRemaining || 
							 crdDiscoveryResult.NamespaceStatus.HasResourcesRemaining) && 
							len(crdDiscoveryResult.ProblematicCRDs) > 0
		if shouldCleanupCRDs {
			fmt.Printf("\n🧹 Namespace conditions indicate CRD issues - attempting cleanup...\n")
		}
	}
	
	if shouldCleanupCRDs {
		if err := AttemptCRDCleanup(ctx, clientset, crdDiscoveryResult, namespace); err != nil {
			fmt.Printf("⚠️  Warning: Failed to clean up some CRDs: %v\n", err)
		}
	} else if len(crdDiscoveryResult.ProblematicCRDs) > 0 {
		fmt.Printf("\n💡 Found %d CRDs with finalizers, but namespace conditions don't indicate they're blocking deletion\n", len(crdDiscoveryResult.ProblematicCRDs))
		fmt.Printf("💡 Use --force flag for aggressive CRD cleanup if needed\n")
	}

	// Phase 6: Proceed with namespace deletion based on mode
	if forceDelete {
		return EnhancedNukeNamespace(ctx, clientset, dynamicClient, namespace, detector)
	}
	return EnhancedStandardDeleteWithCRDRetry(ctx, clientset, namespace, crdDiscoveryResult)
}

// EnhancedDeleteNamespaceWithDryRun provides ArgoCD-aware namespace deletion with dry-run support
func EnhancedDeleteNamespaceWithDryRun(ctx context.Context, clientset kubernetes.Interface, namespace string, forceDelete bool, isDryRun bool) error {
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

	// Phase 2: Discover problematic CRDs (always run for diagnostics)
	fmt.Printf("\n🔍 Discovering CRDs that might be causing namespace termination issues...\n")
	crdDiscoveryResult, err := DiscoverProblematicCRDs(ctx, clientset, namespace)
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to discover problematic CRDs: %v\n", err)
		crdDiscoveryResult = &CRDDiscoveryResult{} // Continue with empty result
	}

	// Phase 3: Enhanced diagnostics with ArgoCD and CRD awareness (always run in dry-run)
	if isDryRun {
		if forceDelete {
			return EnhancedDryRunWithForceMode(ctx, clientset, dynamicClient, namespace, argoCDApps, crdDiscoveryResult)
		} else {
			return EnhancedDiagnoseNamespaceWithCRDs(ctx, clientset, dynamicClient, namespace, argoCDApps, crdDiscoveryResult)
		}
	}

	// Phase 4: Handle ArgoCD applications first (if any)
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

	// Phase 5: Intelligent CRD cleanup based on mode
	shouldCleanupCRDs := false
	
	if forceDelete {
		// Force mode: Always cleanup CRDs if any are found with finalizers
		shouldCleanupCRDs = len(crdDiscoveryResult.ProblematicCRDs) > 0
		if shouldCleanupCRDs {
			fmt.Printf("\n💥 FORCE MODE: Aggressively cleaning up all problematic CRDs...\n")
		}
	} else {
		// Standard mode: Only cleanup CRDs if namespace conditions indicate they're causing issues
		shouldCleanupCRDs = (crdDiscoveryResult.NamespaceStatus.HasFinalizersRemaining || 
							 crdDiscoveryResult.NamespaceStatus.HasResourcesRemaining) && 
							len(crdDiscoveryResult.ProblematicCRDs) > 0
		if shouldCleanupCRDs {
			fmt.Printf("\n🧹 Namespace conditions indicate CRD issues - attempting cleanup...\n")
		}
	}
	
	if shouldCleanupCRDs {
		if err := AttemptCRDCleanup(ctx, clientset, crdDiscoveryResult, namespace); err != nil {
			fmt.Printf("⚠️  Warning: Failed to clean up some CRDs: %v\n", err)
		}
	} else if len(crdDiscoveryResult.ProblematicCRDs) > 0 {
		fmt.Printf("\n💡 Found %d CRDs with finalizers, but namespace conditions don't indicate they're blocking deletion\n", len(crdDiscoveryResult.ProblematicCRDs))
		fmt.Printf("💡 Use --force flag for aggressive CRD cleanup if needed\n")
	}

	// Phase 6: Proceed with namespace deletion based on mode
	if forceDelete {
		return EnhancedNukeNamespace(ctx, clientset, dynamicClient, namespace, detector)
	}
	return EnhancedStandardDeleteWithCRDRetry(ctx, clientset, namespace, crdDiscoveryResult)
}

// EnhancedDryRunWithForceMode shows debug output of what force mode would do without actually doing it
func EnhancedDryRunWithForceMode(
	ctx context.Context, 
	clientset kubernetes.Interface, 
	dynamicClient dynamic.Interface,
	namespace string,
	argoCDApps []unstructured.Unstructured,
	crdResult *CRDDiscoveryResult,
) error {
	fmt.Printf("🔍 DRY-RUN + FORCE MODE: Debug output for namespace: %s\n", namespace)
	fmt.Printf("=======================================================\n")

	// Run standard diagnostics first
	DiagnoseStuckNamespace(ctx, clientset, namespace)

	// Show what would be done with ArgoCD applications
	if len(argoCDApps) > 0 {
		fmt.Printf("\n🔍 ARGOCD APPLICATIONS (WOULD BE HANDLED):\n")
		fmt.Printf("=========================================\n")
		fmt.Printf("🎯 Found %d ArgoCD application(s) that WOULD BE DELETED:\n", len(argoCDApps))
		
		for _, app := range argoCDApps {
			appName := app.GetName()
			appNamespace := app.GetNamespace()
			
			fmt.Printf("\n📊 ArgoCD Application: %s/%s\n", appNamespace, appName)
			fmt.Printf("   🗑️  WOULD DELETE: kubectl delete application %s -n %s\n", appName, appNamespace)
			
			// Check application finalizers
			finalizers := app.GetFinalizers()
			if len(finalizers) > 0 {
				fmt.Printf("   ⚠️  Has finalizers: %v\n", finalizers)
				fmt.Printf("   🔧 WOULD REMOVE FINALIZERS if stuck\n")
			}
			
			// Extract and display destination info
			destination, found, _ := unstructured.NestedMap(app.Object, "spec", "destination")
			if found {
				fmt.Printf("   🔗 Destination: ")
				if server, ok := destination["server"].(string); ok {
					fmt.Printf("Server=%s, ", server)
				}
				if ns, ok := destination["namespace"].(string); ok {
					fmt.Printf("Namespace=%s", ns)
				}
				fmt.Println()
			}
		}
	}

	// Show what would be done with CRDs
	if len(crdResult.ProblematicCRDs) > 0 {
		fmt.Printf("\n🎯 CRD CLEANUP (WOULD BE PERFORMED):\n")
		fmt.Printf("===================================\n")
		fmt.Printf("💥 FORCE MODE would aggressively clean up %d problematic CRDs:\n", len(crdResult.ProblematicCRDs))
		
		for i, crd := range crdResult.ProblematicCRDs {
			fmt.Printf("\n%d. CRD: %s (Group: %s, Version: %s)\n", i+1, crd.Name, crd.Group, crd.Version)
			fmt.Printf("   Kind: %s\n", crd.Kind)
			fmt.Printf("   Total Resources: %d\n", crd.TotalResources)
			fmt.Printf("   Resources with Finalizers: %d\n", len(crd.ResourcesWithFinalizers))
			
			fmt.Printf("   📋 WOULD CLEAN UP these resources:\n")
			for _, resource := range crd.ResourcesWithFinalizers {
				fmt.Printf("     - %s (finalizers: %v)\n", resource.Name, resource.Finalizers)
				fmt.Printf("       🔧 WOULD REMOVE FINALIZERS: kubectl patch %s %s -n %s --type json -p '[{\"op\":\"remove\",\"path\":\"/metadata/finalizers\"}]'\n", 
					crd.Name, resource.Name, namespace)
				fmt.Printf("       🗑️  WOULD DELETE: kubectl delete %s %s -n %s --grace-period=0\n", 
					crd.Name, resource.Name, namespace)
			}
		}
	} else {
		fmt.Printf("\n✅ CRD ANALYSIS:\n")
		fmt.Printf("===============\n")
		fmt.Printf("No problematic CRDs found - force mode would skip CRD cleanup\n")
	}

	// Show what would be done with namespace
	fmt.Printf("\n💥 NAMESPACE DELETION (WOULD BE PERFORMED):\n")
	fmt.Printf("==========================================\n")
	fmt.Printf("FORCE MODE would perform these actions:\n")
	fmt.Printf("1. 🚀 WOULD FORCE DELETE all pods with grace period 0\n")
	fmt.Printf("2. 🗑️  WOULD DELETE all services, deployments, configmaps, secrets\n")
	fmt.Printf("3. 💥 WOULD FORCE DELETE all custom resources\n")
	fmt.Printf("4. 🔧 WOULD REMOVE finalizers from all resources\n")
	fmt.Printf("5. 🗑️  WOULD DELETE the namespace itself\n")
	fmt.Printf("6. 🔧 WOULD REMOVE namespace finalizers if stuck\n")

	// Show comprehensive recommendations
	fmt.Printf("\n💡 COMPREHENSIVE RECOMMENDATIONS:\n")
	fmt.Printf("================================\n")
	
	step := 1
	
	if len(argoCDApps) > 0 {
		fmt.Printf("%d. Delete the ArgoCD Application(s) first:\n", step)
		for _, app := range argoCDApps {
			fmt.Printf("   kubectl delete application %s -n %s\n", app.GetName(), app.GetNamespace())
		}
		fmt.Printf("\n   If applications are stuck, remove their finalizers:\n")
		for _, app := range argoCDApps {
			fmt.Printf("   kubectl patch application %s -n %s --type json -p '[{\"op\":\"remove\",\"path\":\"/metadata/finalizers\"}]'\n", 
				app.GetName(), app.GetNamespace())
		}
		step++
	}
	
	if len(crdResult.ProblematicCRDs) > 0 {
		fmt.Printf("\n%d. Clean up problematic CRDs:\n", step)
		for _, crd := range crdResult.ProblematicCRDs {
			fmt.Printf("   For CRD %s:\n", crd.Name)
			for _, resource := range crd.ResourcesWithFinalizers {
				fmt.Printf("   kubectl patch %s %s -n %s --type json -p '[{\"op\":\"remove\",\"path\":\"/metadata/finalizers\"}]'\n", 
					crd.Name, resource.Name, namespace)
			}
		}
		step++
	}
	
	fmt.Printf("\n%d. Execute the actual cleanup:\n", step)
	fmt.Printf("   # Standard mode (cleans up CRDs only if they're causing termination issues)\n")
	fmt.Printf("   kubectl-nuke ns %s\n", namespace)
	fmt.Printf("   \n")
	fmt.Printf("   # Force mode (aggressively cleans up all CRDs with finalizers)\n")
	fmt.Printf("   kubectl-nuke ns %s --force\n", namespace)
	
	fmt.Printf("\n%d. If all else fails, try with webhook bypass:\n", step+1)
	fmt.Printf("   kubectl-nuke ns %s --force --bypass-webhooks\n", namespace)
	
	return nil
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

// EnhancedStandardDeleteWithCRDRetry performs standard namespace deletion with CRD retry capability
func EnhancedStandardDeleteWithCRDRetry(ctx context.Context, clientset kubernetes.Interface, namespace string, crdResult *CRDDiscoveryResult) error {
	fmt.Printf("🔄 Enhanced standard deletion of namespace: %s\n", namespace)

	// Use existing standard deletion logic
	deleted, terminating, err := DeleteNamespace(ctx, clientset, namespace)
	if err != nil {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	if terminating {
		fmt.Printf("⚠️  Namespace %s is stuck in Terminating state.\n", namespace)
		
		// If we have CRD discovery results and there are problematic CRDs, try cleaning them up again
		if len(crdResult.ProblematicCRDs) > 0 {
			fmt.Printf("🔄 Re-attempting CRD cleanup for stuck namespace...\n")
			if err := AttemptCRDCleanup(ctx, clientset, crdResult, namespace); err != nil {
				fmt.Printf("⚠️  Warning: CRD cleanup retry failed: %v\n", err)
			}
		}
		
		fmt.Printf("🔧 Attempting finalizer removal...\n")
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

// EnhancedDiagnoseNamespaceWithCRDs provides detailed diagnostics with ArgoCD and CRD awareness
func EnhancedDiagnoseNamespaceWithCRDs(
	ctx context.Context, 
	clientset kubernetes.Interface, 
	dynamicClient dynamic.Interface,
	namespace string,
	argoCDApps []unstructured.Unstructured,
	crdResult *CRDDiscoveryResult,
) error {
	fmt.Printf("🔍 Running enhanced diagnostics with CRD discovery on namespace: %s\n", namespace)

	// Run standard diagnostics first
	DiagnoseStuckNamespace(ctx, clientset, namespace)

	// Enhanced ArgoCD diagnostics
	if len(argoCDApps) > 0 {
		fmt.Printf("\n🔍 ARGOCD DIAGNOSTICS:\n")
		fmt.Printf("====================\n")
		fmt.Printf("🎯 Found %d ArgoCD application(s) managing this namespace:\n", len(argoCDApps))
		
		for _, app := range argoCDApps {
			appName := app.GetName()
			appNamespace := app.GetNamespace()
			
			fmt.Printf("\n📊 ArgoCD Application: %s/%s\n", appNamespace, appName)
			
			// Check application finalizers
			finalizers := app.GetFinalizers()
			if len(finalizers) > 0 {
				fmt.Printf("⚠️  Application has finalizers: %v\n", finalizers)
				fmt.Printf("💡 Tip: These finalizers may prevent proper cleanup\n")
			}
			
			// Extract and display destination info
			destination, found, _ := unstructured.NestedMap(app.Object, "spec", "destination")
			if found {
				fmt.Printf("🔗 Destination: ")
				if server, ok := destination["server"].(string); ok {
					fmt.Printf("Server=%s, ", server)
				}
				if ns, ok := destination["namespace"].(string); ok {
					fmt.Printf("Namespace=%s", ns)
				}
				fmt.Println()
			}
			
			// Extract sync status
			syncStatus, found, _ := unstructured.NestedMap(app.Object, "status", "sync")
			if found {
				if status, ok := syncStatus["status"].(string); ok {
					fmt.Printf("🔄 Sync Status: %s\n", status)
				}
			}
			
			// Extract health status
			healthStatus, found, _ := unstructured.NestedMap(app.Object, "status", "health")
			if found {
				if status, ok := healthStatus["status"].(string); ok {
					fmt.Printf("💓 Health Status: %s\n", status)
				}
				if message, ok := healthStatus["message"].(string); ok && message != "" {
					fmt.Printf("   Message: %s\n", message)
				}
			}
		}
	}

	// CRD Discovery Results (already displayed by DiscoverProblematicCRDs)
	if len(crdResult.ProblematicCRDs) > 0 {
		fmt.Printf("\n🎯 CRD ANALYSIS SUMMARY:\n")
		fmt.Printf("======================\n")
		fmt.Printf("Found %d problematic CRDs with resources that have finalizers\n", len(crdResult.ProblematicCRDs))
		
		totalResourcesWithFinalizers := 0
		for _, crd := range crdResult.ProblematicCRDs {
			totalResourcesWithFinalizers += len(crd.ResourcesWithFinalizers)
		}
		fmt.Printf("Total resources with finalizers: %d\n", totalResourcesWithFinalizers)
	}
	
	// Combined recommendations
	fmt.Printf("\n💡 COMPREHENSIVE RECOMMENDATIONS:\n")
	fmt.Printf("================================\n")
	
	step := 1
	
	if len(argoCDApps) > 0 {
		fmt.Printf("%d. Delete the ArgoCD Application(s) first:\n", step)
		for _, app := range argoCDApps {
			fmt.Printf("   kubectl delete application %s -n %s\n", app.GetName(), app.GetNamespace())
		}
		fmt.Printf("\n   If applications are stuck, remove their finalizers:\n")
		for _, app := range argoCDApps {
			fmt.Printf("   kubectl patch application %s -n %s --type json -p '[{\"op\":\"remove\",\"path\":\"/metadata/finalizers\"}]'\n", 
				app.GetName(), app.GetNamespace())
		}
		step++
	}
	
	if len(crdResult.ProblematicCRDs) > 0 {
		fmt.Printf("\n%d. Clean up problematic CRDs:\n", step)
		for _, crd := range crdResult.ProblematicCRDs {
			fmt.Printf("   For CRD %s:\n", crd.Name)
			for _, resource := range crd.ResourcesWithFinalizers {
				fmt.Printf("   kubectl patch %s %s -n %s --type json -p '[{\"op\":\"remove\",\"path\":\"/metadata/finalizers\"}]'\n", 
					crd.Name, resource.Name, namespace)
			}
		}
		step++
	}
	
	fmt.Printf("\n%d. Use kubectl-nuke with intelligent CRD cleanup:\n", step)
	fmt.Printf("   # Standard mode (cleans up CRDs only if they're causing termination issues)\n")
	fmt.Printf("   kubectl-nuke ns %s\n", namespace)
	fmt.Printf("   \n")
	fmt.Printf("   # Force mode (aggressively cleans up all CRDs with finalizers)\n")
	fmt.Printf("   kubectl-nuke ns %s --force\n", namespace)
	
	fmt.Printf("\n%d. If all else fails, try with webhook bypass:\n", step+1)
	fmt.Printf("   kubectl-nuke ns %s --force --bypass-webhooks\n", namespace)
	
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

// EnhancedDeleteNamespace provides ArgoCD-aware namespace deletion with CRD discovery (backward compatibility)
func EnhancedDeleteNamespace(ctx context.Context, clientset kubernetes.Interface, namespace string, forceDelete bool, diagnoseOnly bool) error {
	// Call the new function with dry-run mode
	return EnhancedDeleteNamespaceWithDryRun(ctx, clientset, namespace, forceDelete, diagnoseOnly)
}
