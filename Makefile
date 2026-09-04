.PHONY: build

build:
	docker build -t rlivdev/maint-cli:latest .

install:
	./install.sh