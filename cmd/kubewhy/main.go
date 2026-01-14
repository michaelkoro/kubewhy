// Package main is the entry point for the kubewhy CLI tool.
// kubewhy diagnoses Kubernetes resource failures and explains them in plain English.
package main

import (
	"os"

	"github.com/michaelkoro/kubewhy/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}

