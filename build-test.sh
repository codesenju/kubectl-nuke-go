#!/bin/bash

# Build script to test ArgoCD integration implementation

echo "🔨 Building kubectl-nuke with ArgoCD integration..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go first."
    exit 1
fi

# Build the project
echo "📦 Building binary..."
go build -o kubectl-nuke-test ./cmd/kubectl-nuke

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    echo "📍 Binary created: ./kubectl-nuke-test"
    echo ""
    echo "🧪 Testing basic functionality..."
    ./kubectl-nuke-test version
    echo ""
    echo "💡 To test ArgoCD integration:"
    echo "   ./kubectl-nuke-test ns <namespace> --diagnose-only"
    echo "   ./kubectl-nuke-test ns <namespace> --force"
else
    echo "❌ Build failed!"
    exit 1
fi
