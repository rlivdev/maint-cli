.PHONY: build docker-build docker-run docker-push test vet install

version ?= latest
image := rlivdev/maint-cli:$(version)

build:
	CGO_ENABLED=0 go build -trimpath -o bin/maint ./main.go

vet:
	go vet ./...

test:
	go test ./...

docker-build:
	docker build -t $(image) .

docker-run:
	docker run --rm -it \
		-v "$$HOME/.maint:/data" \
		$(image) $(ARGS)

docker-push:
	docker push $(image)

install:
	./install.sh