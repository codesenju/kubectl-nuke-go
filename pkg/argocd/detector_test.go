package argocd

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestIsArgoCDManagedResource(t *testing.T) {
	detector := &Detector{}

	tests := []struct {
		name     string
		resource *unstructured.Unstructured
		expected bool
	}{
		{
			name: "Resource with ArgoCD managed-by label",
			resource: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							LabelArgoCDManagedBy: "argocd",
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Resource with ArgoCD instance annotation",
			resource: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"annotations": map[string]interface{}{
							AnnotationArgoCDInstance: "my-app",
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Resource with ArgoCD instance label",
			resource: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							LabelArgoCDInstance: "my-app",
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Resource without ArgoCD labels/annotations",
			resource: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": "my-app",
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Resource with empty metadata",
			resource: &unstructured.Unstructured{
				Object: map[string]interface{}{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.IsArgoCDManagedResource(tt.resource)
			if result != tt.expected {
				t.Errorf("IsArgoCDManagedResource() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
