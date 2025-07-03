#!/bin/bash

# Test script for install.sh
set -e

echo "Testing install.sh script..."

# Create temporary test directory
TEST_DIR="/tmp/kubectl-nuke-test-$$"
mkdir -p "$TEST_DIR"

# Cleanup function
cleanup() {
    echo "Cleaning up test directory: $TEST_DIR"
    rm -rf "$TEST_DIR"
}
trap cleanup EXIT

# Test 1: Help option
echo "Test 1: Testing --help option"
if ./install.sh --help > /dev/null 2>&1; then
    echo "✅ Help option works"
else
    echo "❌ Help option failed"
    exit 1
fi

# Test 2: Install to custom path
echo "Test 2: Testing installation to custom path"
if ./install.sh --path "$TEST_DIR" > /dev/null 2>&1; then
    echo "✅ Custom path installation works"
    if [ -f "$TEST_DIR/kubectl-nuke" ]; then
        echo "✅ Binary was installed"
    else
        echo "❌ Binary not found after installation"
        exit 1
    fi
else
    echo "❌ Custom path installation failed"
    exit 1
fi

# Test 3: Check if binary is executable
echo "Test 3: Testing if binary is executable"
if [ -x "$TEST_DIR/kubectl-nuke" ]; then
    echo "✅ Binary is executable"
else
    echo "❌ Binary is not executable"
    exit 1
fi

echo "All tests passed! ✅"
