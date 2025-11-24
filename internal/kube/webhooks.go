package kube

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DetectAndHandleWebhookIssues detects and handles webhook validation issues
// that might be blocking namespace or resource deletion
func DetectAndHandleWebhookIssues(ctx context.Context, clientset kubernetes.Interface, autoDisable bool) error {
	webhookConfigs, err := clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list webhook configurations: %w", err)
	}

	problematicWebhooks := 0
	disabledWebhooks := 0

	for _, webhookConfig := range webhookConfigs.Items {
		isProblematic := false

		for _, webhook := range webhookConfig.Webhooks {
			if webhook.ClientConfig.Service != nil {
				service := webhook.ClientConfig.Service

				_, err := clientset.CoreV1().Services(service.Namespace).Get(ctx, service.Name, metav1.GetOptions{})
				if err != nil {
					isProblematic = true
					// reason removed
					break
				}

				ns, err := clientset.CoreV1().Namespaces().Get(ctx, service.Namespace, metav1.GetOptions{})
				if err == nil && ns.Status.Phase == "Terminating" {
					isProblematic = true
					// reason removed
					break
				}
			}
		}

		if isProblematic {
			problematicWebhooks++

			shouldDisable := autoDisable
			if !autoDisable {
				fmt.Printf("❓ Disable webhook %s? (y/n): ", webhookConfig.Name)
				var response string
				fmt.Scanln(&response)
				shouldDisable = strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
			}

			if shouldDisable {
				err := clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(ctx, webhookConfig.Name, metav1.DeleteOptions{})
				if err == nil {
					disabledWebhooks++
				}
			}
		}
	}

	// Check MutatingWebhookConfigurations
	mutatingWebhookConfigs, err := clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list mutating webhook configurations: %w", err)
	}

	for _, webhookConfig := range mutatingWebhookConfigs.Items {
		isProblematic := false
		// reason removed

		for _, webhook := range webhookConfig.Webhooks {
			if webhook.ClientConfig.Service != nil {
				service := webhook.ClientConfig.Service

				_, err := clientset.CoreV1().Services(service.Namespace).Get(ctx, service.Name, metav1.GetOptions{})
				if err != nil {
					isProblematic = true
					// reason removed
					break
				}

				ns, err := clientset.CoreV1().Namespaces().Get(ctx, service.Namespace, metav1.GetOptions{})
				if err == nil && ns.Status.Phase == "Terminating" {
					isProblematic = true
					// reason removed
					break
				}
			}
		}

		if isProblematic {
			problematicWebhooks++

			shouldDisable := autoDisable
			if !autoDisable {
				fmt.Printf("❓ Disable webhook %s? (y/n): ", webhookConfig.Name)
				var response string
				fmt.Scanln(&response)
				shouldDisable = strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
			}

			if shouldDisable {
				err := clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(ctx, webhookConfig.Name, metav1.DeleteOptions{})
				if err == nil {
					disabledWebhooks++
				}
			}
		}
	}

	if disabledWebhooks > 0 {
		fmt.Printf("🔧 Removed %d webhook(s)\n", disabledWebhooks)
	}

	return nil
}

// DisableStorageProviderWebhooks specifically targets webhooks from common storage providers
// that might be causing issues with namespace deletion
func DisableStorageProviderWebhooks(ctx context.Context, clientset kubernetes.Interface) error {
	// Check for common storage provider webhooks
	storageProviders := []string{
		"longhorn",
		"rook-ceph",
		"openebs",
		"portworx",
		"storageos",
	}

	fmt.Printf("🔍 Checking for storage provider webhooks...\n")
	disabledCount := 0

	// Check ValidatingWebhookConfigurations
	webhookConfigs, err := clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list webhook configurations: %w", err)
	}

	for _, webhookConfig := range webhookConfigs.Items {
		for _, provider := range storageProviders {
			if strings.Contains(strings.ToLower(webhookConfig.Name), provider) {
				fmt.Printf("🔧 Found %s webhook: %s. Attempting to remove...\n", provider, webhookConfig.Name)
				err := clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(ctx, webhookConfig.Name, metav1.DeleteOptions{})
				if err != nil {
					fmt.Printf("⚠️  Failed to remove webhook: %v\n", err)
				} else {
					fmt.Printf("✅ Successfully removed webhook: %s\n", webhookConfig.Name)
					disabledCount++
				}
				break
			}
		}
	}

	// Check MutatingWebhookConfigurations
	mutatingWebhookConfigs, err := clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list mutating webhook configurations: %w", err)
	}

	for _, webhookConfig := range mutatingWebhookConfigs.Items {
		for _, provider := range storageProviders {
			if strings.Contains(strings.ToLower(webhookConfig.Name), provider) {
				fmt.Printf("🔧 Found %s mutating webhook: %s. Attempting to remove...\n", provider, webhookConfig.Name)
				err := clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(ctx, webhookConfig.Name, metav1.DeleteOptions{})
				if err != nil {
					fmt.Printf("⚠️  Failed to remove webhook: %v\n", err)
				} else {
					fmt.Printf("✅ Successfully removed webhook: %s\n", webhookConfig.Name)
					disabledCount++
				}
				break
			}
		}
	}

	if disabledCount > 0 {
		fmt.Printf("📊 Disabled %d storage provider webhooks\n", disabledCount)
	} else {
		fmt.Printf("ℹ️  No storage provider webhooks found\n")
	}

	return nil
}
