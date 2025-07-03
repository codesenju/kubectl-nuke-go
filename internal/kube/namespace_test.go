package kube

import (
	"context"
	"errors"
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
	err := ForceRemoveFinalizers(ctx, client, "finalizer-ns")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestForceRemoveFinalizers_NoFinalizers(t *testing.T) {
	client := k8sfake.NewSimpleClientset(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "no-finalizer-ns"},
		Status:     corev1.NamespaceStatus{Phase: "Terminating"},
	})
	ctx := context.TODO()
	err := ForceRemoveFinalizers(ctx, client, "no-finalizer-ns")
	if !errors.Is(err, errors.New("no finalizers to remove")) && err == nil {
		t.Errorf("expected error for no finalizers, got %v", err)
	}
}
