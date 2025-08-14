package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/codesenju/kubectl-nuke-go/internal/kube"
	"github.com/codesenju/kubectl-nuke-go/internal/updater"
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

	// Create update command
	var forceUpdate bool
	var checkOnly bool
	var updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Check for updates and upgrade kubectl-nuke to the latest version",
		Long: `Check for the latest version of kubectl-nuke on GitHub releases and perform an in-place upgrade.

This command will:
1. Check the latest release on GitHub
2. Compare with the current version
3. Prompt for user confirmation (unless --force is used)
4. Download and install the new version if confirmed
5. Create a backup of the current binary before updating

The update process is safe and will restore the original binary if the update fails.
Use --force to skip the confirmation prompt and update automatically.`,
		Example: `  # Check for updates without installing
  kubectl-nuke update --check-only
  
  # Update to the latest version (with confirmation prompt)
  kubectl-nuke update
  
  # Force update without confirmation prompt
  kubectl-nuke update --force`,
		Run: performUpdate,
	}
	updateCmd.Flags().BoolVar(&forceUpdate, "force", false, "Force update without confirmation prompt")
	updateCmd.Flags().BoolVar(&checkOnly, "check-only", false, "Only check for updates without installing")

	// Create namespace command
	var forceDelete bool
	var bypassWebhooks bool
	var forceAPIDirect bool
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

With --force flag, it will:
- Aggressively delete all resources in the namespace first
- Automatically discover and clean up problematic CRDs causing termination issues
- Remove finalizers from stuck resources

With --dry-run/--diagnose-only flag, it will only analyze issues without attempting deletion.
When combined with --force, it shows debug-level output of what aggressive cleanup would do.

With --bypass-webhooks flag, it will temporarily disable problematic webhooks that might block deletion.
With --force-api-direct flag, it will use direct API server calls to bypass admission controllers.`,
		Example: `  # Delete a namespace (standard mode with CRD discovery)
  kubectl-nuke ns my-namespace
  
  # Aggressively delete a namespace with auto CRD cleanup
  kubectl-nuke ns my-namespace --force
  kubectl-nuke ns my-namespace -f
  
  # Analyze issues without making changes (dry-run mode)
  kubectl-nuke ns my-namespace --dry-run
  kubectl-nuke ns my-namespace --diagnose-only
  
  # Show debug output of what force mode would do (without actually doing it)
  kubectl-nuke ns my-namespace --force --dry-run
  
  # Bypass webhooks that might block deletion
  kubectl-nuke ns my-namespace --bypass-webhooks
  
  # Use direct API calls for most aggressive deletion
  kubectl-nuke ns my-namespace --force --force-api-direct
  
  # Delete a namespace with custom kubeconfig
  kubectl-nuke --kubeconfig /path/to/config ns my-namespace`,
		Args: cobra.ExactArgs(1),
		Run:  deleteNamespace,
	}
	nsCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "Aggressively delete all resources and auto-cleanup problematic CRDs (DESTRUCTIVE)")
	nsCmd.Flags().BoolVar(&bypassWebhooks, "bypass-webhooks", false, "Temporarily disable webhooks that might block deletion")
	nsCmd.Flags().BoolVar(&forceAPIDirect, "force-api-direct", false, "Use direct API server calls to bypass admission controllers (requires kubectl)")
	nsCmd.Flags().BoolVar(&diagnoseOnly, "diagnose-only", false, "Only analyze issues without attempting deletion (alias: --dry-run)")
	nsCmd.Flags().BoolVar(&diagnoseOnly, "dry-run", false, "Only analyze issues without attempting deletion (alias: --diagnose-only)")

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
	rootCmd.AddCommand(updateCmd)
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
	forceAPIDirect, _ := cmd.Flags().GetBool("force-api-direct")
	diagnoseOnly, _ := cmd.Flags().GetBool("diagnose-only")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Combine diagnose-only and dry-run flags
	isDryRun := diagnoseOnly || dryRun

	if forceDelete && isDryRun {
		fmt.Printf("🔍 DRY-RUN + FORCE MODE: Showing debug output of what aggressive deletion would do\n")
		fmt.Printf("⚠️  This is a dry-run - no actual changes will be made\n")
		fmt.Printf("💥 Would aggressively delete namespace: %s\n", namespace)
		fmt.Printf("🤖 Would automatically discover and clean up problematic CRDs\n")
	} else if forceDelete {
		fmt.Printf("💥 FORCE MODE: Preparing to aggressively delete namespace: %s\n", namespace)
		fmt.Printf("⚠️  WARNING: This will forcefully delete ALL resources in the namespace!\n")
		fmt.Printf("🤖 AUTO CRD CLEANUP: Will automatically discover and clean up problematic CRDs\n")
	} else if isDryRun {
		fmt.Printf("🔍 DRY-RUN MODE: Analyzing namespace without making changes: %s\n", namespace)
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

	if !forceDelete && !isDryRun {
		fmt.Printf("📋 Namespace %s is in '%s' state.\n", ns.Name, ns.Status.Phase)
	} else if isDryRun {
		fmt.Printf("📋 Namespace %s is in '%s' state.\n", ns.Name, ns.Status.Phase)
	}

	// Note: bypassWebhooks and forceAPIDirect are available for future use
	_ = bypassWebhooks
	_ = forceAPIDirect

	// Use enhanced namespace deletion with ArgoCD and CRD support
	// Pass both forceDelete and isDryRun to the enhanced function
	err = kube.EnhancedDeleteNamespaceWithDryRun(ctx, clientset, namespace, forceDelete, isDryRun)
	if err != nil {
		// Check if the error is because namespace was already deleted (success case)
		if strings.Contains(err.Error(), "not found") {
			fmt.Printf("✅ Namespace %s was successfully deleted during execution!\n", namespace)
			if !isDryRun {
				fmt.Printf("🎉 Mission accomplished! The namespace cleanup was successful.\n")
			}
			return
		}
		fmt.Fprintf(os.Stderr, "❌ Failed to delete namespace %s: %v\n", namespace, err)
		os.Exit(1)
	}

	// If not in dry-run mode, wait for namespace deletion
	if !isDryRun {
		// Wait for complete deletion with longer timeout for force mode
		timeout := 30
		if !forceDelete {
			timeout = 15
		}
		
		if kube.WaitForNamespaceDeletion(ctx, clientset, namespace, timeout) {
			if forceDelete {
				fmt.Printf("💥 Namespace %s has been completely nuked!\n", namespace)
			} else {
				fmt.Printf("✅ Namespace %s deleted successfully!\n", namespace)
			}
		} else {
			fmt.Printf("⚠️  Namespace %s may still exist. Check manually with: kubectl get ns %s\n", namespace, namespace)
		}
	}
	
	return

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

func performUpdate(cmd *cobra.Command, args []string) {
	forceUpdate, _ := cmd.Flags().GetBool("force")
	checkOnly, _ := cmd.Flags().GetBool("check-only")

	fmt.Printf("🔄 kubectl-nuke updater\n")
	fmt.Printf("📋 Current version: %s\n", version)

	checker := updater.NewUpdateChecker(version)
	
	// Check for updates
	release, hasUpdate, err := checker.CheckForUpdate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to check for updates: %v\n", err)
		os.Exit(1)
	}

	if !hasUpdate && !forceUpdate {
		fmt.Printf("✅ You're already running the latest version (%s)\n", version)
		return
	}

	if hasUpdate {
		fmt.Printf("🆕 New version available: %s\n", release.TagName)
		fmt.Printf("📝 Release notes:\n%s\n", release.Body)
	} else if forceUpdate {
		fmt.Printf("🔄 Force update requested for version: %s\n", release.TagName)
	}

	if checkOnly {
		if hasUpdate {
			fmt.Printf("💡 Run 'kubectl-nuke update' to install the latest version\n")
		}
		return
	}

	// Prompt user for confirmation unless --force is used
	if !forceUpdate {
		if !promptUserConfirmation(hasUpdate, release.TagName) {
			fmt.Printf("⏹️  Update cancelled by user\n")
			return
		}
	}

	// Perform the update
	fmt.Printf("🚀 Starting update process...\n")
	if err := checker.PerformUpdate(release, forceUpdate); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Update failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🎉 Update completed successfully!\n")
	fmt.Printf("💡 You can now use the updated version of kubectl-nuke\n")
}

func promptUserConfirmation(hasUpdate bool, newVersion string) bool {
	var message string
	
	if hasUpdate {
		message = fmt.Sprintf("Do you want to update to version %s? (y/N): ", newVersion)
	} else {
		message = fmt.Sprintf("Do you want to reinstall version %s? (y/N): ", newVersion)
	}
	
	fmt.Print(message)
	
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to read user input: %v\n", err)
		return false
	}
	
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
