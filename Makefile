# A whole bunch of helper stuff including building, testing and
# pushing to docker hub.

# This makefile has few dependencies which need to be installed before
# you can use most of the functionality
_dep_minify := $(shell which minify 2> /dev/null)
_dep_packr := $(shell which packr 2> /dev/null)
_dep_zip := $(shell which zip 2> /dev/null)
_dep_golangci := $(shell which golangci-lint 2> /dev/null)

check_deps:
ifndef _dep_minify
	$(error github.com/tdewolff/minify/tree/master/cmd/minify is required to build burner.kiwi)
endif
ifndef _dep_packr
	$(error github.com/gobuffalo/packr is required to build burner.kiwi)
endif

git_commit = $(shell git rev-parse --short HEAD)
custom_css = custom.$(shell md5sum ./static/custom.css | cut -c -32).min.css
icons_css = icons.$(shell md5sum ./static/icons.css | cut -c -32).min.css
milligram_css = milligram.$(shell md5sum ./static/milligram.css | cut -c -32).min.css
normalize_css = normalize.$(shell md5sum ./static/normalize.css | cut -c -32).min.css

lint:
ifndef _dep_golangci
	$(error github.com/golangci/golangci-lint is required to lint burner.kiwi)
endif
	golangci-lint run ./... --skip-dirs vendor/ --skip-files [A-Za-z]*_test.go --enable misspell --enable gocyclo

test:
	go test -race ./...

build_dir:
	mkdir -p ./build/static

clean:
	@rm -rf build/ 2> /dev/null || true

minify:
	minify -o ./static/${custom_css} ./static/custom.css
	minify -o ./static/${icons_css} ./static/icons.css
	minify -o ./static/${milligram_css} ./static/milligram.css
	minify -o ./static/${normalize_css} ./static/normalize.css

do-build: check_deps clean build_dir minify
	CGO_ENABLED=0 packr build -ldflags "-X github.com/haydenwoodhead/burner.kiwi/server.version=${git_commit} -X github.com/haydenwoodhead/burner.kiwi/server.milligram=${milligram_css} -X github.com/haydenwoodhead/burner.kiwi/server.custom=${custom_css} -X github.com/haydenwoodhead/burner.kiwi/server.icons=${icons_css} -X github.com/haydenwoodhead/burner.kiwi/server.normalize=${normalize_css}" -o "./build/burnerkiwi"

do-build-sqlite: check_deps clean build_dir minify
	CGO_ENABLED=1 packr build -ldflags "-X github.com/haydenwoodhead/burner.kiwi/server.version=${git_commit} -X github.com/haydenwoodhead/burner.kiwi/server.milligram=${milligram_css} -X github.com/haydenwoodhead/burner.kiwi/server.custom=${custom_css} -X github.com/haydenwoodhead/burner.kiwi/server.icons=${icons_css} -X github.com/haydenwoodhead/burner.kiwi/server.normalize=${normalize_css}" -o "./build/burnerkiwi"

# clean up static dir after build
build build-sqlite:  %: do-%
	mv ./static/${custom_css} ./build/static/
	mv ./static/${icons_css} ./build/static/
	mv ./static/${milligram_css} ./build/static/
	mv ./static/${normalize_css} ./build/static/
	cp ./static/roger-proportional.svg ./build/static
	mkdir -p ./build/static/icons
	cp -r ./static/icons/* ./build/static/icons/

prepare-aws:
ifndef _dep_zip
	$(error zip is required to prepare aws assets for burner.kiwi)
endif
	mkdir -p ./build/cloudformation
	cp cloudformation.json ./build/cloudformation/
	zip ./build/cloudformation/burnerkiwi.zip ./build/burnerkiwi

prepare-docker:
	mv build/burnerkiwi .

image: TAG ?= latest
image: build prepare-docker
image:
	docker build -t haydensw/burner-kiwi:$(TAG) .

push: TAG ?= latest
push:
	docker push haydensw/burner-kiwi:$(TAG)

image-and-push: image push

deploy:
	kubectl apply -f kubernetes/service.yaml
