GO ?= CGO_ENABLED=0 go
PKGS ?= $(shell go list ./... | grep -v /vendor/)

.PHONY: clean
clean:
	rm ksp*
	rm -r dist

.PHONY: fmt
fmt:
	goimports -w .

.PHONY: fmt-test
fmt-test:
	test -z $(shell goimports -l .)

.PHONY: lint
lint:
	golint -set_exit_status $(PKGS)

.PHONY: vet
vet:
	$(GO) vet $(PKGS)

.PHONY: test
test:
	$(GO) test -v -cover $(PKGS)

.PHONY: build
build:
	$(GO) build -o ksp .

.PHONY: build-dist
build-dist:
	gox -output "dist/{{.Dir}}_{{.OS}}_{{.Arch}}"
