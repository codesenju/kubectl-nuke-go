package argocd

import (
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Common ArgoCD labels and annotations
const (
	// Labels
	LabelArgoCDInstance     = "app.kubernetes.io/instance"
	LabelArgoCDName         = "app.kubernetes.io/name"
	LabelArgoCDPartOf       = "app.kubernetes.io/part-of"
	LabelArgoCDManagedBy    = "app.kubernetes.io/managed-by"
	
	// Annotations
	AnnotationArgoCDInstance = "argocd.argoproj.io/instance"
	
	// ArgoCD application CRD
	ArgoCDGroup      = "argoproj.io"
	ArgoCDVersion    = "v1alpha1"
	ArgoCDResource   = "applications"
	ArgoCDKind       = "Application"
	
	// Timeout for ArgoCD operations
	ArgoCDTimeout = 60 * time.Second
)

// ArgoCDDetector handles detection and management of ArgoCD resources
type ArgoCDDetector struct {
	kubeClient    kubernetes.Interface
	dynamicClient dynamic.Interface
}

// NewArgoCDDetector creates a new ArgoCD detector
func NewArgoCDDetector(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface) *ArgoCDDetector {
	return &ArgoCDDetector{
		kubeClient:    kubeClient,
		dynamicClient: dynamicClient,
	}
}

// DetectArgoCDAppsForNamespace finds all ArgoCD Applications that manage resources in the given namespace
func (d *ArgoCDDetector) DetectArgoCDAppsForNamespace(ctx context.Context, namespace string) ([]unstructured.Unstructured, error) {
	// Get ArgoCD Application GVR
	appGVR := schema.GroupVersionResource{
		Group:    ArgoCDGroup,
		Version:  ArgoCDVersion,
		Resource: ArgoCDResource,
	}
	
	// List all ArgoCD applications in all namespaces
	// ArgoCD apps can be in any namespace but typically in argocd namespace
	allApps, err := d.dynamicClient.Resource(appGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		// If CRD doesn't exist, return empty list (ArgoCD might not be installed)
		if strings.Contains(err.Error(), "the server could not find the requested resource") {
			return []unstructured.Unstructured{}, nil
		}
		return nil, fmt.Errorf("failed to list ArgoCD applications: %w", err)
	}
	
	// Filter apps that target the namespace
	var matchingApps []unstructured.Unstructured
	for _, app := range allApps.Items {
		// Check if app targets our namespace
		targetNs, found, err := unstructured.NestedString(app.Object, "spec", "destination", "namespace")
		if err != nil {
			continue // Skip if error extracting namespace
		}
		
		if found && targetNs == namespace {
			matchingApps = append(matchingApps, app)
		}
	}
	
	return matchingApps, nil
}

// IsArgoCDManagedResource checks if a resource is managed by ArgoCD
func (d *ArgoCDDetector) IsArgoCDManagedResource(resource *unstructured.Unstructured) bool {
	// Check for ArgoCD labels
	labels := resource.GetLabels()
	if labels != nil {
		// Check for common ArgoCD labels
		if val, ok := labels[LabelArgoCDManagedBy]; ok && val == "argocd" {
			return true
		}
		if val, ok := labels[LabelArgoCDPartOf]; ok && val == "argocd" {
			return true
		}
		if val, ok := labels[LabelArgoCDName]; ok && (val == "argocd-application" || val == "argocd") {
			return true
		}
		if _, ok := labels[LabelArgoCDInstance]; ok {
			return true
		}
	}
	
	// Check for ArgoCD annotations
	annotations := resource.GetAnnotations()
	if annotations != nil {
		if _, ok := annotations[AnnotationArgoCDInstance]; ok {
			return true
		}
	}
	
	return false
}
