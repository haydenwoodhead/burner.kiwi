# This makefile has few dependencies which need to be installed before
# you can use most of the functionality
_dep_minify := $(shell which minify 2> /dev/null)
_dep_golangci := $(shell which golangci-lint 2> /dev/null)

check_deps:
ifndef _dep_minify
	$(error github.com/tdewolff/minify is required to build burner.kiwi)
endif

git_commit = $(shell git rev-parse --short HEAD)
custom_css = styles.$(shell md5sum ./burner/static/styles.css | cut -c -32).min.css

lint:
ifndef _dep_golangci
	$(error github.com/golangci/golangci-lint is required to lint burner.kiwi)
endif
	golangci-lint run ./... --skip-dirs vendor/ --skip-files [A-Za-z]*_test.go --enable misspell --enable gocyclo

test:
	go test -race ./...

clean:
	rm ./burner/static/*.min.css || true

minify: check_deps
	minify -o ./burner/static/${custom_css} ./burner/static/styles.css

static: clean minify
	@echo "Static assets done"

do-build: static
	GO_ENABLED=0 go build -ldflags "-X github.com/haydenwoodhead/burner.kiwi/burner.version=${git_commit} -X github.com/haydenwoodhead/burner.kiwi/burner.css=${custom_css}" -o "./burnerkiwi"

do-build-sqlite: static
	CGO_ENABLED=1 go build -ldflags "-X github.com/haydenwoodhead/burner.kiwi/burner.version=${git_commit} -X github.com/haydenwoodhead/burner.kiwi/burner.css=${custom_css}" -o "./burnerkiwi"

build-sqlite-linux-arm: 
	CC=aarch64-linux-gnu-gcc GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -ldflags "-X github.com/haydenwoodhead/burner.kiwi/burner.version=${git_commit} -X github.com/haydenwoodhead/burner.kiwi/burner.css=${custom_css}" -o "./burnerkiwi.arm64"

build-sqlite-linux-amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -ldflags "-X github.com/haydenwoodhead/burner.kiwi/burner.version=${git_commit} -X github.com/haydenwoodhead/burner.kiwi/burner.css=${custom_css}" -o "./burnerkiwi.amd64"

build-for-docker: static build-sqlite-linux-arm build-sqlite-linux-amd64
	@echo "Binaries built"

# clean up static dir after build
build build-sqlite:  %: do-% clean
	@echo "Done"
