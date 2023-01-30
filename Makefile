GO ?= go
GOFMT ?= gofmt "-s"
GO_FILES := $(shell find . -name "*.go")
EXAMPLES_PATH = ./examples

# check can build
.PHONY: build-examples
build-examples:
	$(GO) build -v $(EXAMPLES_PATH)/...

.PHONY: fmt
fmt:
	$(GOFMT) -w $(GO_FILES)

.PHONY: fmt-check
fmt-check:
	@diff=$$($(GOFMT) -d $(GO_FILES)); \
	if [ -n "$$diff" ]; then \
		echo "Please run 'make fmt' and commit the result:"; \
		echo "$${diff}"; \
		exit 1; \
	fi;
