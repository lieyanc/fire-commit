VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
BIN_DIR := bin

.PHONY: build install clean

build:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/firecommit ./cmd/firecommit
	@ln -sf firecommit $(BIN_DIR)/fcmt
	@ln -sf firecommit $(BIN_DIR)/git-fire-commit

install: build
	@cp $(BIN_DIR)/firecommit $(GOPATH)/bin/firecommit 2>/dev/null || cp $(BIN_DIR)/firecommit $(HOME)/go/bin/firecommit
	@ln -sf $(GOPATH)/bin/firecommit $(GOPATH)/bin/fcmt 2>/dev/null || ln -sf $(HOME)/go/bin/firecommit $(HOME)/go/bin/fcmt
	@ln -sf $(GOPATH)/bin/firecommit $(GOPATH)/bin/git-fire-commit 2>/dev/null || ln -sf $(HOME)/go/bin/firecommit $(HOME)/go/bin/git-fire-commit

clean:
	rm -rf $(BIN_DIR)
