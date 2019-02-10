clean:
	-rm -rf buildres/
	-rm burnerkiwi
	-rm burnerkiwi.test

prepare: 
	mv buildres/burnerkiwi .

build: 
	./build.sh

image: TAG ?= latest
image: clean
image: build
image: prepare
image: 
	docker build -t haydensw/burner-kiwi:$(TAG) .

push: TAG ?= latest
push:
	docker push haydensw/burner-kiwi:$(TAG)

image-and-push: image
image-and-push: push

deploy:
	kubectl apply -f kubernetes/service.yaml
