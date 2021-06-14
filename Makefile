# This makefile has few dependencies which need to be installed before
# you can use most of the functionality
_dep_minify := $(shell which minify 2> /dev/null)
_dep_golangci := $(shell which golangci-lint 2> /dev/null)

check_deps:
ifndef _dep_minify
	$(error github.com/tdewolff/minify/tree/master/cmd/minify is required to build burner.kiwi)
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

minify:
	minify -o ./burner/static/${custom_css} ./burner/static/styles.css

static: clean minify
	@echo "Static assets done"

do-build: check_deps clean build_dir minify
	CGO_ENABLED=0 packr build -ldflags "-X github.com/haydenwoodhead/burner.kiwi/burner.version=${git_commit} -X github.com/haydenwoodhead/burner.kiwi/burner.custom=${custom_css}.gz -o "./burnerkiwi"

do-build-sqlite: check_deps clean build_dir minify
	CGO_ENABLED=1 packr build -ldflags "-X github.com/haydenwoodhead/burner.kiwi/burner.version=${git_commit} -X github.com/haydenwoodhead/burner.kiwi/burner.custom=${custom_css}.gz -o "./burnerkiwi"

# clean up static dir after build
build build-sqlite:  %: do-% clean
	@echo "Done"

prepare-aws:
ifndef _dep_zip
	$(error zip is required to prepare aws assets for burner.kiwi)
endif
	mkdir -p ./build/cloudformation
	cp cloudformation.json ./build/cloudformation/
	zip ./build/cloudformation/burnerkiwi.zip ./build/burnerkiwi