VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
BIN_DIR := bin
INSTALL_DIR := $(HOME)/.fire-commit/bin

.PHONY: build install clean dist

build:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/firecommit ./cmd/firecommit
	@ln -sf firecommit $(BIN_DIR)/fcmt
	@ln -sf firecommit $(BIN_DIR)/git-fire-commit

install: build
	@mkdir -p $(INSTALL_DIR)
	@cp $(BIN_DIR)/firecommit $(INSTALL_DIR)/firecommit
	@ln -sf firecommit $(INSTALL_DIR)/fcmt
	@ln -sf firecommit $(INSTALL_DIR)/git-fire-commit
	@echo "Installed to $(INSTALL_DIR)"
	@echo "Make sure $(INSTALL_DIR) is in your PATH."

clean:
	rm -rf $(BIN_DIR) dist

dist:
	@mkdir -p dist
	@for platform in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do \
		IFS='/' read -r GOOS GOARCH <<< "$$platform"; \
		BIN=firecommit; \
		[ "$$GOOS" = "windows" ] && BIN=firecommit.exe; \
		DIR="dist/fire-commit_$(VERSION)_$${GOOS}_$${GOARCH}"; \
		echo "Building $${GOOS}/$${GOARCH}..."; \
		mkdir -p "$$DIR"; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build $(LDFLAGS) -o "$$DIR/$$BIN" ./cmd/firecommit; \
	done
