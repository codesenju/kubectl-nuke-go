package kube

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DiagnoseStuckNamespace provides detailed diagnostics for stuck namespaces
func DiagnoseStuckNamespace(ctx context.Context, clientset kubernetes.Interface, namespace string) {
	fmt.Printf("🔍 Running diagnostics on namespace: %s\n", namespace)

	// Get namespace details
	ns, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Printf("✅ Namespace %s was successfully deleted during execution!\n", namespace)
			return
		}
		fmt.Printf("⚠️  Could not get namespace details: %v\n", err)
		return
	}

	// Check status conditions
	fmt.Printf("📊 Namespace Status Conditions:\n")
	for _, condition := range ns.Status.Conditions {
		fmt.Printf("  - %s: %s (Reason: %s)\n", condition.Type, condition.Status, condition.Reason)
		if condition.Message != "" {
			fmt.Printf("    Message: %s\n", condition.Message)
		}
	}

	// Check for finalizers on the namespace
	if len(ns.Finalizers) > 0 {
		fmt.Printf("🔍 Namespace has finalizers: %v\n", ns.Finalizers)
	}

	// Check for remaining resources
	fmt.Printf("🔍 Checking for remaining resources in namespace...\n")

	// List common resource types
	resourceTypes := []struct {
		name     string
		listFunc func() (int, error)
	}{
		{"pods", func() (int, error) {
			list, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
			return len(list.Items), err
		}},
		{"services", func() (int, error) {
			list, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
			return len(list.Items), err
		}},
		{"persistentvolumeclaims", func() (int, error) {
			list, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
			return len(list.Items), err
		}},
		{"configmaps", func() (int, error) {
			list, err := clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
			return len(list.Items), err
		}},
		{"secrets", func() (int, error) {
			list, err := clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
			return len(list.Items), err
		}},
		{"deployments", func() (int, error) {
			list, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
			return len(list.Items), err
		}},
		{"statefulsets", func() (int, error) {
			list, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
			return len(list.Items), err
		}},
		{"daemonsets", func() (int, error) {
			list, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
			return len(list.Items), err
		}},
	}

	for _, rt := range resourceTypes {
		count, err := rt.listFunc()
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				// Namespace was deleted during diagnostics
				fmt.Printf("✅ Namespace %s was successfully deleted during diagnostics!\n", namespace)
				return
			}
			fmt.Printf("⚠️  Error listing %s: %v\n", rt.name, err)
		} else if count > 0 {
			fmt.Printf("⚠️  Found %d %s resources still in namespace\n", count, rt.name)
		}
	}

	// Check for PVCs with finalizers
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Printf("✅ Namespace %s was successfully deleted during diagnostics!\n", namespace)
			return
		}
	} else if len(pvcs.Items) > 0 {
		for _, pvc := range pvcs.Items {
			if len(pvc.Finalizers) > 0 {
				fmt.Printf("⚠️  PVC %s has finalizers: %v\n", pvc.Name, pvc.Finalizers)
			}
		}
	}

	// Check for ArgoCD resources
	DetectArgocdResources(ctx, clientset, namespace)
}

// DetectArgocdResources detects resources created by ArgoCD
func DetectArgocdResources(ctx context.Context, clientset kubernetes.Interface, namespace string) {
	// Check for ArgoCD annotations on resources
	found := false

	// Check pods
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, pod := range pods.Items {
			for key := range pod.Annotations {
				if strings.Contains(key, "argocd.argoproj.io") {
					fmt.Printf("🔍 Detected pod managed by ArgoCD: %s\n", pod.Name)
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}

	// Check PVCs
	if !found {
		pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, pvc := range pvcs.Items {
				for key := range pvc.Annotations {
					if strings.Contains(key, "argocd.argoproj.io") {
						fmt.Printf("🔍 Detected PVC managed by ArgoCD: %s\n", pvc.Name)
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}
	}

	// Check services
	if !found {
		services, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, svc := range services.Items {
				for key := range svc.Annotations {
					if strings.Contains(key, "argocd.argoproj.io") {
						fmt.Printf("🔍 Detected service managed by ArgoCD: %s\n", svc.Name)
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}
	}

	if found {
		fmt.Printf("ℹ️  This namespace contains resources managed by ArgoCD\n")
		fmt.Printf("💡 Tip: Check if the ArgoCD application was properly deleted with: kubectl get applications -A\n")
		fmt.Printf("💡 Tip: You may need to remove the ArgoCD finalizers from resources\n")
	}
}
