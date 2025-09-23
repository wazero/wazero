package main

// Force importing wazero here, so that we can see the actual wazero version in the main_test.go.
import (
	_ "github.com/wazero/wazero/internal/version"
)
