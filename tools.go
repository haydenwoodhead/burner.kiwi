// +build tools

package main

import (
	_ "github.com/gobuffalo/packr"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/mattn/goveralls"
	_ "github.com/tdewolff/minify/cmd/minify"
	_ "golang.org/x/tools/cmd/cover"
)
