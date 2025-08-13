# ArgoCD Integration

kubectl-nuke-go now includes enhanced support for handling namespaces that contain resources managed by ArgoCD. This feature helps prevent conflicts and ensures proper cleanup when deleting namespaces with ArgoCD-managed applications.

## Features

### ArgoCD Application Detection
- Automatically detects ArgoCD Applications that manage resources in the target namespace
- Identifies resources with ArgoCD labels and annotations
- Provides detailed information about detected applications

### Smart Deletion Workflow
1. **Detection Phase**: Scans for ArgoCD Applications targeting the namespace
2. **Application Cleanup**: Deletes ArgoCD Applications first to prevent reconciliation conflicts
3. **Resource Cleanup**: Removes ArgoCD-managed resources and their finalizers
4. **Namespace Deletion**: Proceeds with standard or force deletion

### Enhanced Diagnostics
- Shows ArgoCD Applications managing the namespace
- Displays application sync and health status
- Provides specific recommendations for ArgoCD-managed resources
- Suggests proper cleanup commands

## Usage

### Standard Deletion with ArgoCD Support
```bash
# Enhanced namespace deletion (automatically detects ArgoCD)
kubectl-nuke ns my-namespace

# Force mode with ArgoCD awareness
kubectl-nuke ns my-namespace --force
```

### Enhanced Diagnostics
```bash
# Comprehensive diagnostics including ArgoCD analysis
kubectl-nuke ns my-namespace --diagnose-only
```

## Example Output

### ArgoCD Detection
```
🔍 Checking for ArgoCD applications managing namespace: my-app-namespace
🎯 Found 2 ArgoCD application(s) managing this namespace:
  - argocd/my-app-frontend
  - argocd/my-app-backend
```

### Enhanced Diagnostics
```
🔍 ARGOCD DIAGNOSTICS:
====================
🎯 Found 1 ArgoCD application(s) managing this namespace:

📊 ArgoCD Application: argocd/my-app
⚠️  Application has finalizers: [resources-finalizer.argocd.argoproj.io]
🔗 Destination: Server=https://kubernetes.default.svc, Namespace=my-app-namespace
🔄 Sync Status: Synced
💓 Health Status: Healthy

💡 RECOMMENDATIONS:
=================
1. Delete the ArgoCD Application(s) first:
   kubectl delete application my-app -n argocd

2. If applications are stuck, remove their finalizers:
   kubectl patch application my-app -n argocd --type json -p '[{"op":"remove","path":"/metadata/finalizers"}]'

3. Then delete the namespace with kubectl-nuke:
   kubectl-nuke ns my-app-namespace --force
```

### Application Cleanup
```
🔄 Handling ArgoCD applications before namespace deletion...
🔄 Deleting ArgoCD Application: argocd/my-app-frontend
✅ Successfully deleted ArgoCD Application: argocd/my-app-frontend
🔄 Deleting ArgoCD Application: argocd/my-app-backend
✅ Successfully deleted ArgoCD Application: argocd/my-app-backend
⏳ Waiting for ArgoCD to clean up resources...
```

## How It Works

### Detection Logic
The tool identifies ArgoCD-managed resources by checking for:

**Labels:**
- `app.kubernetes.io/managed-by: argocd`
- `app.kubernetes.io/part-of: argocd`
- `app.kubernetes.io/instance: <app-name>`
- `app.kubernetes.io/name: argocd-application`

**Annotations:**
- `argocd.argoproj.io/instance: <app-name>`

### Application Discovery
- Searches for ArgoCD Application CRDs across all namespaces
- Filters applications that target the specified namespace
- Extracts application metadata, sync status, and health information

### Cleanup Strategy
1. **Graceful Cleanup**: Deletes ArgoCD Applications first, allowing ArgoCD to perform its normal cleanup
2. **Finalizer Removal**: Removes ArgoCD finalizers from applications if they get stuck
3. **Resource Cleanup**: Removes finalizers from ArgoCD-managed resources in the namespace
4. **Fallback**: Uses standard force deletion if ArgoCD cleanup fails

## Benefits

### Prevents Conflicts
- Avoids race conditions between kubectl-nuke and ArgoCD reconciliation
- Ensures ArgoCD doesn't try to recreate resources during deletion
- Maintains proper state consistency

### Better User Experience
- Provides clear feedback about ArgoCD involvement
- Offers specific recommendations for manual intervention
- Handles the entire cleanup chain automatically

### Safer Operations
- Follows ArgoCD best practices for application deletion
- Reduces risk of orphaned resources
- Provides detailed diagnostics before destructive operations

## Troubleshooting

### Common Issues

**ArgoCD Applications Won't Delete**
```bash
# Check application status
kubectl get applications -A

# Remove finalizers manually
kubectl patch application <app-name> -n <namespace> --type json \
  -p '[{"op":"remove","path":"/metadata/finalizers"}]'
```

**Resources Keep Getting Recreated**
- Ensure ArgoCD Applications are deleted first
- Check for multiple applications managing the same namespace
- Verify ArgoCD server is responsive

**Finalizers Won't Remove**
- Use `--force` mode for aggressive finalizer removal
- Check for webhook admission controllers blocking updates
- Consider using `--bypass-webhooks` flag

## Configuration

### Environment Variables
- `ARGOCD_TIMEOUT`: Timeout for ArgoCD operations (default: 60s)
- `ARGOCD_NAMESPACE`: Default ArgoCD namespace to search (default: all namespaces)

### Flags
- `--diagnose-only`: Run diagnostics without making changes
- `--force`: Enable aggressive deletion mode
- `--bypass-webhooks`: Disable problematic webhooks during cleanup

## Limitations

- Requires ArgoCD CRDs to be installed for full functionality
- May not detect all ArgoCD patterns in custom installations
- Relies on standard ArgoCD labels and annotations
- Cannot handle ArgoCD Applications in different clusters

## Future Enhancements

- Support for ArgoCD ApplicationSets
- Cross-cluster ArgoCD Application detection
- Integration with ArgoCD API for better status checking
- Support for custom ArgoCD label patterns
- Batch processing of multiple namespaces with ArgoCD apps
