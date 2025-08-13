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
	// Check for ValidatingWebhookConfiguration resources
	webhookConfigs, err := clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list webhook configurations: %w", err)
	}

	fmt.Printf("🔍 Checking for problematic webhook configurations...\n")
	problematicWebhooks := 0
	disabledWebhooks := 0

	for _, webhookConfig := range webhookConfigs.Items {
		isProblematic := false
		reason := ""

		// Check each webhook in the configuration
		for _, webhook := range webhookConfig.Webhooks {
			if webhook.ClientConfig.Service != nil {
				service := webhook.ClientConfig.Service

				// Check if the service exists
				_, err := clientset.CoreV1().Services(service.Namespace).Get(ctx, service.Name, metav1.GetOptions{})
				if err != nil {
					isProblematic = true
					reason = fmt.Sprintf("service %s/%s not found", service.Namespace, service.Name)
					break
				}

				// Check if the namespace is terminating
				ns, err := clientset.CoreV1().Namespaces().Get(ctx, service.Namespace, metav1.GetOptions{})
				if err == nil && ns.Status.Phase == "Terminating" {
					isProblematic = true
					reason = fmt.Sprintf("namespace %s is terminating", service.Namespace)
					break
				}
			}
		}

		if isProblematic {
			problematicWebhooks++
			fmt.Printf("⚠️  Found potentially problematic webhook: %s (%s)\n", webhookConfig.Name, reason)

			shouldDisable := autoDisable
			if !autoDisable {
				// Ask for confirmation before removing
				fmt.Printf("❓ Would you like to temporarily disable this webhook to proceed with deletion? (y/n): ")
				var response string
				fmt.Scanln(&response)
				shouldDisable = strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
			}

			if shouldDisable {
				fmt.Printf("🔧 Temporarily removing webhook configuration: %s\n", webhookConfig.Name)
				err := clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(ctx, webhookConfig.Name, metav1.DeleteOptions{})
				if err != nil {
					fmt.Printf("⚠️  Failed to remove webhook configuration: %v\n", err)
				} else {
					fmt.Printf("✅ Successfully removed webhook configuration: %s\n", webhookConfig.Name)
					disabledWebhooks++
				}
			}
		}
	}

	// Also check MutatingWebhookConfigurations
	mutatingWebhookConfigs, err := clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list mutating webhook configurations: %w", err)
	}

	for _, webhookConfig := range mutatingWebhookConfigs.Items {
		isProblematic := false
		reason := ""

		// Check each webhook in the configuration
		for _, webhook := range webhookConfig.Webhooks {
			if webhook.ClientConfig.Service != nil {
				service := webhook.ClientConfig.Service

				// Check if the service exists
				_, err := clientset.CoreV1().Services(service.Namespace).Get(ctx, service.Name, metav1.GetOptions{})
				if err != nil {
					isProblematic = true
					reason = fmt.Sprintf("service %s/%s not found", service.Namespace, service.Name)
					break
				}

				// Check if the namespace is terminating
				ns, err := clientset.CoreV1().Namespaces().Get(ctx, service.Namespace, metav1.GetOptions{})
				if err == nil && ns.Status.Phase == "Terminating" {
					isProblematic = true
					reason = fmt.Sprintf("namespace %s is terminating", service.Namespace)
					break
				}
			}
		}

		if isProblematic {
			problematicWebhooks++
			fmt.Printf("⚠️  Found potentially problematic mutating webhook: %s (%s)\n", webhookConfig.Name, reason)

			shouldDisable := autoDisable
			if !autoDisable {
				// Ask for confirmation before removing
				fmt.Printf("❓ Would you like to temporarily disable this webhook to proceed with deletion? (y/n): ")
				var response string
				fmt.Scanln(&response)
				shouldDisable = strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
			}

			if shouldDisable {
				fmt.Printf("🔧 Temporarily removing mutating webhook configuration: %s\n", webhookConfig.Name)
				err := clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(ctx, webhookConfig.Name, metav1.DeleteOptions{})
				if err != nil {
					fmt.Printf("⚠️  Failed to remove webhook configuration: %v\n", err)
				} else {
					fmt.Printf("✅ Successfully removed webhook configuration: %s\n", webhookConfig.Name)
					disabledWebhooks++
				}
			}
		}
	}

	if problematicWebhooks > 0 {
		fmt.Printf("📊 Webhook summary: %d problematic webhooks found, %d disabled\n", problematicWebhooks, disabledWebhooks)
	} else {
		fmt.Printf("✅ No problematic webhooks detected\n")
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
