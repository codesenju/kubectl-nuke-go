package argocd

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
)

// Handler handles ArgoCD application deletion and cleanup
type Handler struct {
	dynamicClient dynamic.Interface
}

// NewHandler creates a new ArgoCD handler
func NewHandler(dynamicClient dynamic.Interface) *Handler {
	return &Handler{
		dynamicClient: dynamicClient,
	}
}

// DeleteApplication deletes an ArgoCD application and waits for it to be deleted
func (h *Handler) DeleteApplication(ctx context.Context, app unstructured.Unstructured) error {
	appGVR := schema.GroupVersionResource{
		Group:    ArgoCDGroup,
		Version:  ArgoCDVersion,
		Resource: ArgoCDResource,
	}

	appName := app.GetName()
	appNamespace := app.GetNamespace()

	// Try to delete the application
	err := h.dynamicClient.Resource(appGVR).Namespace(appNamespace).Delete(ctx, appName, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete ArgoCD application %s/%s: %w", appNamespace, appName, err)
	}

	// Wait for application to be deleted with timeout
	err = wait.PollImmediate(2*time.Second, ArgoCDTimeout, func() (bool, error) {
		_, err := h.dynamicClient.Resource(appGVR).Namespace(appNamespace).Get(ctx, appName, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil // Successfully deleted
		}
		if err != nil {
			return false, err // Unexpected error
		}
		return false, nil // Still exists
	})

	if err != nil {
		// If timeout or other error, try to remove finalizers
		return h.RemoveApplicationFinalizers(ctx, app)
	}

	return nil
}

// RemoveApplicationFinalizers removes finalizers from an ArgoCD application
func (h *Handler) RemoveApplicationFinalizers(ctx context.Context, app unstructured.Unstructured) error {
	appGVR := schema.GroupVersionResource{
		Group:    ArgoCDGroup,
		Version:  ArgoCDVersion,
		Resource: ArgoCDResource,
	}

	appName := app.GetName()
	appNamespace := app.GetNamespace()

	// Check if application still exists
	currentApp, err := h.dynamicClient.Resource(appGVR).Namespace(appNamespace).Get(ctx, appName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to get ArgoCD application %s/%s: %w", appNamespace, appName, err)
	}

	// Check if there are finalizers
	finalizers := currentApp.GetFinalizers()
	if len(finalizers) == 0 {
		return nil // No finalizers to remove
	}

	// Create patch to remove finalizers
	patch := []byte(`{"metadata":{"finalizers":[]}}`)
	_, err = h.dynamicClient.Resource(appGVR).Namespace(appNamespace).Patch(
		ctx, appName, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove finalizers from ArgoCD application %s/%s: %w", appNamespace, appName, err)
	}

	// Try to delete again after removing finalizers
	err = h.dynamicClient.Resource(appGVR).Namespace(appNamespace).Delete(ctx, appName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete ArgoCD application %s/%s after removing finalizers: %w", appNamespace, appName, err)
	}

	return nil
}

// DeleteApplications deletes multiple ArgoCD applications
func (h *Handler) DeleteApplications(ctx context.Context, apps []unstructured.Unstructured) error {
	for _, app := range apps {
		appName := app.GetName()
		appNamespace := app.GetNamespace()
		
		fmt.Printf("🔄 Deleting ArgoCD Application: %s/%s\n", appNamespace, appName)
		
		if err := h.DeleteApplication(ctx, app); err != nil {
			fmt.Printf("⚠️ Warning: Failed to delete ArgoCD Application %s/%s: %v\n", appNamespace, appName, err)
			// Continue with other applications even if one fails
			continue
		}
		
		fmt.Printf("✅ Successfully deleted ArgoCD Application: %s/%s\n", appNamespace, appName)
	}
	
	return nil
}
