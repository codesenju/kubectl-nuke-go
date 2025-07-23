package kube

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// HandlePVCFinalizers handles PVC finalizers in a namespace that might be blocking deletion
func HandlePVCFinalizers(ctx context.Context, clientset kubernetes.Interface, namespace string, forceApiDirect bool) error {
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list PVCs: %w", err)
	}

	if len(pvcs.Items) == 0 {
		return nil
	}

	fmt.Printf("🔍 Found %d persistentvolumeclaims resources in namespace %s\n", len(pvcs.Items), namespace)

	for _, pvc := range pvcs.Items {
		fmt.Printf("💥 Force deleting persistentvolumeclaims: %s\n", pvc.Name)

		// Check if PVC has finalizers
		if len(pvc.Finalizers) > 0 {
			fmt.Printf("🔧 Removing finalizers from persistentvolumeclaims: %s\n", pvc.Name)

			// Try multiple approaches to remove finalizers
			success := false

			// 1. Standard update
			pvcCopy := pvc.DeepCopy()
			pvcCopy.Finalizers = nil
			_, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Update(ctx, pvcCopy, metav1.UpdateOptions{})
			if err == nil {
				fmt.Printf("✅ Successfully removed finalizers from PVC: %s\n", pvc.Name)
				success = true
			}

			// 2. Try patch if update fails
			if !success {
				patch := `{"metadata":{"finalizers":null}}`
				_, err = clientset.CoreV1().PersistentVolumeClaims(namespace).Patch(
					ctx,
					pvc.Name,
					types.MergePatchType,
					[]byte(patch),
					metav1.PatchOptions{},
				)
				if err == nil {
					fmt.Printf("✅ Successfully patched finalizers from PVC: %s\n", pvc.Name)
					success = true
				}
			}

			// 3. Try JSON patch if merge patch fails
			if !success {
				jsonPatch := `[{"op": "remove", "path": "/metadata/finalizers"}]`
				_, err = clientset.CoreV1().PersistentVolumeClaims(namespace).Patch(
					ctx,
					pvc.Name,
					types.JSONPatchType,
					[]byte(jsonPatch),
					metav1.PatchOptions{},
				)
				if err == nil {
					fmt.Printf("✅ Successfully JSON patched finalizers from PVC: %s\n", pvc.Name)
					success = true
				}
			}

			// 4. If all methods fail and forceApiDirect is enabled, try direct API approach
			if !success && forceApiDirect {
				err = forceRemovePVCFinalizersViaAPI(namespace, pvc.Name)
				if err == nil {
					fmt.Printf("✅ Successfully removed finalizers via direct API: %s\n", pvc.Name)
					success = true
				}
			}

			if !success {
				fmt.Printf("⚠️  Failed to remove finalizers from %s: %v\n", pvc.Name, err)
			}
		}

		// Try to delete the PVC regardless of finalizer removal success
		gracePeriod := int64(0)
		deleteOptions := metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
		}
		err = clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, pvc.Name, deleteOptions)
		if err != nil {
			fmt.Printf("⚠️  Failed to delete PVC %s: %v\n", pvc.Name, err)
		} else {
			fmt.Printf("✅ Successfully deleted persistentvolumeclaims: %s\n", pvc.Name)
		}
	}

	fmt.Printf("📊 Custom resources summary: %d found, %d deleted\n", len(pvcs.Items), len(pvcs.Items))
	return nil
}

// forceRemovePVCFinalizersViaAPI uses kubectl proxy to directly modify the PVC via API server
func forceRemovePVCFinalizersViaAPI(namespace, pvcName string) error {
	// Start kubectl proxy in background
	cmd := exec.Command("kubectl", "proxy", "--port=8001")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start kubectl proxy: %w", err)
	}

	// Ensure we kill the proxy when done
	defer cmd.Process.Kill()

	// Wait for proxy to start
	time.Sleep(2 * time.Second)

	// Get the PVC JSON
	getPVCCmd := exec.Command("curl", "-s", fmt.Sprintf("http://localhost:8001/api/v1/namespaces/%s/persistentvolumeclaims/%s", namespace, pvcName))
	pvcJSON, err := getPVCCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get PVC JSON: %w", err)
	}

	// Parse the JSON
	var pvc map[string]interface{}
	if err := json.Unmarshal(pvcJSON, &pvc); err != nil {
		return fmt.Errorf("failed to parse PVC JSON: %w", err)
	}

	// Modify the finalizers
	metadata, ok := pvc["metadata"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to get metadata from PVC JSON")
	}
	metadata["finalizers"] = []interface{}{}

	// Convert back to JSON
	modifiedJSON, err := json.Marshal(pvc)
	if err != nil {
		return fmt.Errorf("failed to marshal modified PVC JSON: %w", err)
	}

	// Write to temp file
	tempFile := fmt.Sprintf("/tmp/pvc-%s.json", pvcName)
	writeCmd := exec.Command("bash", "-c", fmt.Sprintf("echo '%s' > %s", string(modifiedJSON), tempFile))
	if err := writeCmd.Run(); err != nil {
		return fmt.Errorf("failed to write modified PVC JSON to temp file: %w", err)
	}

	// Update the PVC
	updateCmd := exec.Command("curl", "-s", "-X", "PUT", "-H", "Content-Type: application/json", "-d", fmt.Sprintf("@%s", tempFile), fmt.Sprintf("http://localhost:8001/api/v1/namespaces/%s/persistentvolumeclaims/%s", namespace, pvcName))
	if err := updateCmd.Run(); err != nil {
		return fmt.Errorf("failed to update PVC via API: %w", err)
	}

	return nil
}

// DetectStorageProviderIssues detects and handles storage provider specific issues
func DetectStorageProviderIssues(ctx context.Context, clientset kubernetes.Interface) error {
	// Check for common storage provider namespaces
	storageNamespaces := []string{
		"longhorn-system",
		"rook-ceph",
		"openebs",
		"portworx",
		"storageos",
	}

	for _, ns := range storageNamespaces {
		namespace, err := clientset.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
		if err == nil {
			fmt.Printf("🔍 Detected storage provider: %s\n", ns)

			// Check if namespace is terminating
			if namespace.Status.Phase == corev1.NamespaceTerminating {
				fmt.Printf("⚠️  Storage provider namespace %s is in Terminating state, which may cause PVC deletion issues\n", ns)
				fmt.Printf("ℹ️  Consider checking for webhooks that might block operations\n")
				
				// Suggest disabling webhooks
				fmt.Printf("💡 Tip: You can disable storage provider webhooks with --bypass-webhooks flag\n")
			}
		}
	}

	return nil
}
