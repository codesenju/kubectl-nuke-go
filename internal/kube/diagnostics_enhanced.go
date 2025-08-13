package kube

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// EnhancedDiagnoseNamespace provides detailed diagnostics with ArgoCD awareness
func EnhancedDiagnoseNamespace(
	ctx context.Context, 
	clientset kubernetes.Interface, 
	dynamicClient dynamic.Interface,
	namespace string,
	argoCDApps []unstructured.Unstructured,
) error {
	fmt.Printf("🔍 Running enhanced diagnostics on namespace: %s\n", namespace)

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
		
		fmt.Printf("\n💡 RECOMMENDATIONS:\n")
		fmt.Printf("=================\n")
		fmt.Printf("1. Delete the ArgoCD Application(s) first:\n")
		for _, app := range argoCDApps {
			fmt.Printf("   kubectl delete application %s -n %s\n", app.GetName(), app.GetNamespace())
		}
		fmt.Printf("\n2. If applications are stuck, remove their finalizers:\n")
		for _, app := range argoCDApps {
			fmt.Printf("   kubectl patch application %s -n %s --type json -p '[{\"op\":\"remove\",\"path\":\"/metadata/finalizers\"}]'\n", 
				app.GetName(), app.GetNamespace())
		}
		fmt.Printf("\n3. Then delete the namespace with kubectl-nuke:\n")
		fmt.Printf("   kubectl-nuke ns %s --force\n", namespace)
	}
	
	return nil
}
