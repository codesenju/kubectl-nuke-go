package kube

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestDeleteNamespace(t *testing.T) {
	client := k8sfake.NewSimpleClientset(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "test-ns"},
		Status:     corev1.NamespaceStatus{Phase: "Active"},
	})
	ctx := context.TODO()
	deleted, terminating, err := DeleteNamespace(ctx, client, "test-ns")
	if err != nil || !deleted || terminating {
		t.Errorf("expected namespace to be deleted, got deleted=%v terminating=%v err=%v", deleted, terminating, err)
	}
}

func TestDeleteNamespace_Terminating(t *testing.T) {
	client := k8sfake.NewSimpleClientset(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "term-ns"},
		Status:     corev1.NamespaceStatus{Phase: "Terminating"},
	})
	ctx := context.TODO()
	deleted, terminating, err := DeleteNamespace(ctx, client, "term-ns")
	if err != nil || deleted || !terminating {
		t.Errorf("expected namespace to be terminating, got deleted=%v terminating=%v err=%v", deleted, terminating, err)
	}
}

func TestForceRemoveFinalizers(t *testing.T) {
	client := k8sfake.NewSimpleClientset(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "finalizer-ns", Finalizers: []string{"kubernetes"}},
		Status:     corev1.NamespaceStatus{Phase: "Terminating"},
	})
	ctx := context.TODO()
	removed, err := ForceRemoveFinalizers(ctx, client, "finalizer-ns")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !removed {
		t.Errorf("expected finalizers to be removed, got removed=%v", removed)
	}
}

func TestForceRemoveFinalizers_NoFinalizers(t *testing.T) {
	client := k8sfake.NewSimpleClientset(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "no-finalizer-ns"},
		Status:     corev1.NamespaceStatus{Phase: "Terminating"},
	})
	ctx := context.TODO()
	removed, err := ForceRemoveFinalizers(ctx, client, "no-finalizer-ns")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if removed {
		t.Errorf("expected no finalizers to be removed, got removed=%v", removed)
	}
}

func TestForceDeletePods(t *testing.T) {
	client := k8sfake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "test-ns"},
		Status:     corev1.PodStatus{Phase: "Running"},
	})
	ctx := context.TODO()
	err := ForceDeletePods(ctx, client, "test-ns", []string{"test-pod"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestForceDeletePods_NotFound(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	ctx := context.TODO()
	err := ForceDeletePods(ctx, client, "test-ns", []string{"nonexistent-pod"})
	if err == nil {
		t.Errorf("expected error for nonexistent pod, got nil")
	}
}
