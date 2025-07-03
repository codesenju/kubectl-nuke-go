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

	"kubectl-nuke-go/internal/kube"
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
including namespaces stuck in Terminating state. It attempts a normal delete first, 
and if the resource is stuck, it forcefully removes finalizers.`,
		Example: `  # Delete a namespace using the 'ns' subcommand
  kubectl-nuke ns my-namespace
  
  # Delete a namespace using the 'namespace' subcommand  
  kubectl-nuke namespace my-namespace
  
  # Use with custom kubeconfig
  kubectl-nuke --kubeconfig /path/to/config ns my-namespace
  
  # Use as kubectl plugin
  kubectl nuke ns my-namespace`,
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
	var nsCmd = &cobra.Command{
		Use:     "ns <namespace>",
		Aliases: []string{"namespace"},
		Short:   "Delete a namespace, including those stuck in Terminating state",
		Long: `Delete a Kubernetes namespace. This command will attempt a normal delete first,
and if the namespace is stuck in Terminating state, it will forcefully remove finalizers.

The command will:
1. Check the current state of the namespace
2. Attempt a normal delete operation
3. If the namespace gets stuck in Terminating state, remove finalizers to force deletion
4. Wait and verify the namespace is fully deleted`,
		Example: `  # Delete a namespace
  kubectl-nuke ns my-namespace
  
  # Delete a namespace with custom kubeconfig
  kubectl-nuke --kubeconfig /path/to/config ns my-namespace`,
		Args: cobra.ExactArgs(1),
		Run:  deleteNamespace,
	}

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(nsCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func deleteNamespace(cmd *cobra.Command, args []string) {
	namespace := args[0]
	ctx := context.TODO()

	fmt.Printf("🔍 Checking namespace: %s\n", namespace)

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

	fmt.Printf("📋 Namespace %s is in '%s' state.\n", ns.Name, ns.Status.Phase)

	// Use internal/kube package for deletion logic
	deleted, terminating, err := kube.DeleteNamespace(ctx, clientset, namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to delete namespace %s: %v\n", namespace, err)
		os.Exit(1)
	}

	if terminating {
		fmt.Printf("⚠️  Namespace %s is already in Terminating state. Attempting to force delete by removing finalizers...\n", namespace)
		err = kube.ForceRemoveFinalizers(ctx, clientset, namespace)
		if err != nil {
			fmt.Printf("❌ Failed to remove finalizers for %s: %v\n", namespace, err)
			os.Exit(1)
		}
		fmt.Printf("🔧 Finalizers removed for %s. Waiting for namespace to be deleted...\n", namespace)
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
			err = kube.ForceRemoveFinalizers(ctx, clientset, namespace)
			if err != nil {
				fmt.Printf("❌ Failed to remove finalizers for %s: %v\n", namespace, err)
				os.Exit(1)
			}
			fmt.Printf("🔧 Finalizers removed for %s. Waiting for namespace to be deleted...\n", namespace)
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

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
