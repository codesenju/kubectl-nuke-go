package kube

import (
	"context"
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DeleteNamespace attempts to delete a namespace and returns true if deleted, false if stuck in terminating, or error.
func DeleteNamespace(ctx context.Context, clientset kubernetes.Interface, name string) (deleted bool, terminating bool, err error) {
	ns, err := clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, false, err
	}
	if ns.Status.Phase == "Terminating" {
		return false, true, nil
	}
	err = clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return false, false, err
	}
	return true, false, nil
}

// ForceRemoveFinalizers removes finalizers from a namespace.
func ForceRemoveFinalizers(ctx context.Context, clientset kubernetes.Interface, name string) error {
	ns, err := clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if len(ns.ObjectMeta.Finalizers) == 0 {
		return errors.New("no finalizers to remove")
	}
	ns.ObjectMeta.Finalizers = nil
	_, err = clientset.CoreV1().Namespaces().Finalize(ctx, ns, metav1.UpdateOptions{})
	return err
}
