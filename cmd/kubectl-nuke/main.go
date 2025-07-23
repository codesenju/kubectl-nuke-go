package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/codesenju/kubectl-nuke-go/internal/kube"
)

var (
	kubeconfig string
	version    = "dev" // This will be set during build
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "kubectl-nuke",
		Short: "A kubectl plugin to forcefully delete Kubernetes resources",
		Long: `kubectl-nuke is a kubectl plugin that can forcefully delete Kubernetes resources, 
including namespaces stuck in Terminating state and unresponsive pods. It provides both 
gentle and aggressive deletion modes to handle stuck resources effectively.

Features:
• Namespace deletion with automatic finalizer removal
• Force mode for aggressive resource cleanup (--force flag)
• Pod force deletion with grace period 0
• Multiple resource type support (pods, services, deployments, etc.)
• Smart finalizer removal with multiple strategies`,
		Example: `  # Delete a namespace (standard mode)
  kubectl-nuke ns my-namespace
  
  # Aggressively delete a namespace and all its contents
  kubectl-nuke ns my-namespace --force
  kubectl-nuke ns my-namespace -f
  
  # Force delete unresponsive pods
  kubectl-nuke pod stuck-pod -n my-namespace
  kubectl-nuke pods pod1 pod2 pod3 -n production
  
  # Use with custom kubeconfig
  kubectl-nuke --kubeconfig /path/to/config ns my-namespace --force
  
  # Use as kubectl plugin
  kubectl nuke ns my-namespace -f
  kubectl nuke pods nginx-123 redis-456 -n default`,
	}

	// Add kubeconfig flag to root command
	if home := homeDir(); home != "" {
		rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "absolute path to the kubeconfig file")
	} else {
		rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}

	// Create version command
	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number of kubectl-nuke",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("kubectl-nuke version %s\n", version)
		},
	}

	// Create namespace command
	var forceDelete bool
	var bypassWebhooks bool
	var forceApiDirect bool
	var diagnoseOnly bool
	var nsCmd = &cobra.Command{
		Use:     "ns <namespace>",
		Aliases: []string{"namespace"},
		Short:   "Delete a namespace, including those stuck in Terminating state",
		Long: `Delete a Kubernetes namespace. This command will attempt a normal delete first,
and if the namespace is stuck in Terminating state, it will forcefully remove finalizers.

The command will:
1. Check the current state of the namespace
2. Attempt a normal delete operation (or aggressive delete with --force)
3. If the namespace gets stuck in Terminating state, remove finalizers to force deletion
4. Wait and verify the namespace is fully deleted

With --force flag, it will aggressively delete all resources first before deleting the namespace.
With --bypass-webhooks flag, it will temporarily disable problematic webhooks that might block deletion.
With --force-api-direct flag, it will use direct API server calls to bypass admission controllers.
With --diagnose-only flag, it will only diagnose issues without attempting deletion.`,
		Example: `  # Delete a namespace
  kubectl-nuke ns my-namespace
  
  # Aggressively delete a namespace and all its contents
  kubectl-nuke ns my-namespace --force
  kubectl-nuke ns my-namespace -f
  
  # Bypass webhooks that might block deletion
  kubectl-nuke ns my-namespace --bypass-webhooks
  
  # Use direct API calls for most aggressive deletion
  kubectl-nuke ns my-namespace --force --force-api-direct
  
  # Only diagnose issues without attempting deletion
  kubectl-nuke ns my-namespace --diagnose-only
  
  # Delete a namespace with custom kubeconfig
  kubectl-nuke --kubeconfig /path/to/config ns my-namespace`,
		Args: cobra.ExactArgs(1),
		Run:  deleteNamespace,
	}
	nsCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "Aggressively delete all resources in the namespace first (DESTRUCTIVE)")
	nsCmd.Flags().BoolVar(&bypassWebhooks, "bypass-webhooks", false, "Temporarily disable webhooks that might block deletion")
	nsCmd.Flags().BoolVar(&forceApiDirect, "force-api-direct", false, "Use direct API server calls to bypass admission controllers (requires kubectl)")
	nsCmd.Flags().BoolVar(&diagnoseOnly, "diagnose-only", false, "Only diagnose issues without attempting deletion")

	// Create pod command for force deleting pods
	var podCmd = &cobra.Command{
		Use:     "pod <pod-name> [pod-name2] [pod-name3]...",
		Aliases: []string{"pods", "po"},
		Short:   "Force delete pods with grace period 0 (DESTRUCTIVE)",
		Long: `Force delete one or more pods with grace period 0 (immediate termination).
This command will forcefully terminate pods without waiting for graceful shutdown.
Use this when pods are stuck or unresponsive.

⚠️  WARNING: This bypasses graceful shutdown and may cause data loss or corruption
if the application doesn't handle sudden termination properly.`,
		Example: `  # Force delete a single pod in default namespace
  kubectl-nuke pod my-pod
  
  # Force delete a pod in a specific namespace
  kubectl-nuke pod my-pod -n my-namespace
  
  # Force delete multiple pods
  kubectl-nuke pods pod1 pod2 pod3 -n my-namespace
  
  # Use with custom kubeconfig
  kubectl-nuke --kubeconfig /path/to/config pod my-pod -n my-namespace`,
		Args: cobra.MinimumNArgs(1),
		Run:  nukePods,
	}
	podCmd.Flags().StringP("namespace", "n", "default", "namespace to delete pods from")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(nsCmd)
	rootCmd.AddCommand(podCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func deleteNamespace(cmd *cobra.Command, args []string) {
	namespace := args[0]
	ctx := context.TODO()

	// Get flag values
	forceDelete, _ := cmd.Flags().GetBool("force")
	bypassWebhooks, _ := cmd.Flags().GetBool("bypass-webhooks")
	forceApiDirect, _ := cmd.Flags().GetBool("force-api-direct")
	diagnoseOnly, _ := cmd.Flags().GetBool("diagnose-only")

	if forceDelete {
		fmt.Printf("💥 FORCE MODE: Preparing to aggressively delete namespace: %s\n", namespace)
		fmt.Printf("⚠️  WARNING: This will forcefully delete ALL resources in the namespace!\n")
	} else {
		fmt.Printf("🔍 Checking namespace: %s\n", namespace)
	}

	// Build config from flags
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load kubeconfig: %v\n", err)
		os.Exit(1)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create Kubernetes client: %v\n", err)
		os.Exit(1)
	}

	// Get namespace to check current state
	ns, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to get namespace %s: %v\n", namespace, err)
		os.Exit(1)
	}

	if !forceDelete && !diagnoseOnly {
		fmt.Printf("📋 Namespace %s is in '%s' state.\n", ns.Name, ns.Status.Phase)
	}

	// If diagnose-only mode, just run diagnostics and exit
	if diagnoseOnly {
		kube.DiagnoseStuckNamespace(ctx, clientset, namespace)
		return
	}

	// If bypass-webhooks is enabled but not force mode, still check for problematic webhooks
	if bypassWebhooks && !forceDelete {
		fmt.Printf("🔍 Checking for problematic webhooks...\n")
		if err := kube.DetectAndHandleWebhookIssues(ctx, clientset, false); err != nil {
			fmt.Printf("⚠️  Warning: Failed to handle webhook issues: %v\n", err)
		}
	}

	// If force mode, use aggressive deletion
	if forceDelete {
		err = kube.NukeNamespace(ctx, clientset, namespace, bypassWebhooks, forceApiDirect)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to force delete namespace %s: %v\n", namespace, err)
			os.Exit(1)
		}

		// Wait for complete deletion with longer timeout for force mode
		if kube.WaitForNamespaceDeletion(ctx, clientset, namespace, 30) {
			fmt.Printf("💥 Namespace %s has been completely nuked!\n", namespace)
		} else {
			fmt.Printf("⚠️  Namespace %s may still exist. Check manually with: kubectl get ns %s\n", namespace, namespace)
		}
		return
	}

	// Use internal/kube package for normal deletion logic
	deleted, terminating, err := kube.DeleteNamespace(ctx, clientset, namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to delete namespace %s: %v\n", namespace, err)
		os.Exit(1)
	}

	if terminating {
		fmt.Printf("⚠️  Namespace %s is already in Terminating state. Attempting to force delete by removing finalizers...\n", namespace)
		
		// If bypass-webhooks is enabled, check for problematic webhooks
		if bypassWebhooks {
			fmt.Printf("🔍 Checking for problematic webhooks...\n")
			if err := kube.DetectAndHandleWebhookIssues(ctx, clientset, false); err != nil {
				fmt.Printf("⚠️  Warning: Failed to handle webhook issues: %v\n", err)
			}
		}
		
		// Handle PVC finalizers if force-api-direct is enabled
		if forceApiDirect {
			if err := kube.HandlePVCFinalizers(ctx, clientset, namespace, true); err != nil {
				fmt.Printf("⚠️  Warning: Failed to handle PVC finalizers: %v\n", err)
			}
		}
		
		removed, err := kube.ForceRemoveFinalizers(ctx, clientset, namespace)
		if err != nil {
			fmt.Printf("❌ Failed to remove finalizers for %s: %v\n", namespace, err)
			os.Exit(1)
		}
		if removed {
			fmt.Printf("🔧 Finalizers removed for %s. Waiting for namespace to be deleted...\n", namespace)
		} else {
			fmt.Printf("ℹ️  No finalizers found on %s. Namespace should delete naturally or may need manual intervention.\n", namespace)
		}
		waitForDeletion(ctx, clientset, namespace, 10)
		return
	}

	if deleted {
		fmt.Printf("📤 Delete request sent for namespace %s. Waiting to see if it terminates...\n", namespace)

		// Wait and check if namespace is deleted, up to 5 seconds
		if waitForDeletion(ctx, clientset, namespace, 5) {
			return
		}

		fmt.Printf("⚠️  Namespace %s was not deleted after 10 seconds. Checking if it's stuck in Terminating...\n", namespace)

		// Check if namespace is now stuck in Terminating
		nsCheck, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		if err == nil && nsCheck.Status.Phase == "Terminating" {
			fmt.Printf("🔧 Namespace %s is stuck in Terminating. Forcibly removing finalizers...\n", namespace)
			
			// If bypass-webhooks is enabled, check for problematic webhooks
			if bypassWebhooks {
				fmt.Printf("🔍 Checking for problematic webhooks...\n")
				if err := kube.DetectAndHandleWebhookIssues(ctx, clientset, false); err != nil {
					fmt.Printf("⚠️  Warning: Failed to handle webhook issues: %v\n", err)
				}
			}
			
			// Handle PVC finalizers if force-api-direct is enabled
			if forceApiDirect {
				if err := kube.HandlePVCFinalizers(ctx, clientset, namespace, true); err != nil {
					fmt.Printf("⚠️  Warning: Failed to handle PVC finalizers: %v\n", err)
				}
			}
			
			removed, err := kube.ForceRemoveFinalizers(ctx, clientset, namespace)
			if err != nil {
				fmt.Printf("❌ Failed to remove finalizers for %s: %v\n", namespace, err)
				os.Exit(1)
			}
			if removed {
				fmt.Printf("🔧 Finalizers removed for %s. Waiting for namespace to be deleted...\n", namespace)
			} else {
				fmt.Printf("ℹ️  No finalizers found on %s. Namespace should delete naturally or may need manual intervention.\n", namespace)
			}
			waitForDeletion(ctx, clientset, namespace, 10)
		} else {
			fmt.Printf("✅ Namespace %s deleted or not stuck in Terminating.\n", namespace)
		}
	}
}

func waitForDeletion(ctx context.Context, clientset kubernetes.Interface, namespace string, maxAttempts int) bool {
	for i := 0; i < maxAttempts; i++ {
		time.Sleep(2 * time.Second)
		_, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("✅ Namespace %s deleted successfully!\n", namespace)
			return true
		}
		fmt.Printf("⏳ Waiting for namespace %s to be deleted... (%d/%d)\n", namespace, i+1, maxAttempts)
	}
	fmt.Printf("⚠️  Namespace %s was not deleted after %d seconds. It may still be terminating or stuck.\n", namespace, maxAttempts*2)
	return false
}

func nukePods(cmd *cobra.Command, args []string) {
	podNames := args
	ctx := context.TODO()

	// Get the namespace flag value
	namespace, _ := cmd.Flags().GetString("namespace")

	fmt.Printf("💥 FORCE DELETE MODE: Preparing to force delete %d pod(s) in namespace: %s\n", len(podNames), namespace)
	fmt.Printf("⚠️  WARNING: This will forcefully terminate pods without graceful shutdown!\n")

	// Build config from flags
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load kubeconfig: %v\n", err)
		os.Exit(1)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create Kubernetes client: %v\n", err)
		os.Exit(1)
	}

	// Use the ForceDeletePods function
	err = kube.ForceDeletePods(ctx, clientset, namespace, podNames)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Some pods failed to delete: %v\n", err)
		// Don't exit with error code since some pods might have been deleted successfully
	}

	fmt.Printf("✅ Force delete operation completed!\n")
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
