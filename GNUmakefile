# Copyright 2022 The Cockroach Authors.
#
# Use of this software is governed by the CockroachDB Software License
# included in the /LICENSE file.

.PHONY: all
all: build
	$(MAKE) help

.PHONY: help
help:
	@echo
	@echo "Tip: use ./dev instead of 'make'."
	@echo "Try:"
	@echo "    ./dev help"
	@echo

# Generic build rules.
.PHONY: build build%
build:
	./dev build $(TARGET)
# Alias: buildshort -> build short; buildoss -> build oss; buildtests -> build tests etc.
build%:
	./dev build $(@:build%=%)

.PHONY: doctor
doctor:
	./dev doctor

# Most common rules.
.PHONY: generate test bench
generate test bench:
	./dev $@ $(TARGET)

# Documented clean-all rules.
.PHONY: clean
clean:
	./dev ui clean --all
	bazel clean --expunge

# Documented clean-everything rule (dangerous: removes working tree edits!)
.PHONY: unsafe-clean
unsafe-clean: clean
	git clean -dxf

## Indicate the base root directory where to install.
## Can point e.g. to a container root.
DESTDIR      :=
## The target tree inside DESTDIR.
prefix       := /usr/local
## The target bin directory inside the target tree.
bindir       := $(prefix)/bin
libdir       := $(prefix)/lib
## The install program.
INSTALL      := install

TARGET_TRIPLE := $(shell $(shell go env CC) -dumpmachine)
target-is-windows := $(findstring w64,$(TARGET_TRIPLE))
target-is-macos := $(findstring darwin,$(TARGET_TRIPLE))
DYN_EXT     := so
EXE_EXT     :=
ifdef target-is-macos
DYN_EXT     := dylib
endif
ifdef target-is-windows
DYN_EXT     := dll
EXE_EXT     := .exe
endif

.PHONY: install
install: build buildgeos
	: Install the GEOS library.
	$(INSTALL) -d -m 755 $(DESTDIR)$(libdir)
	$(INSTALL) -m 755 lib/libgeos.$(DYN_EXT) $(DESTDIR)$(libdir)/libgeos.$(DYN_EXT)
	$(INSTALL) -m 755 lib/libgeos_c.$(DYN_EXT) $(DESTDIR)$(libdir)/libgeos_c.$(DYN_EXT)
	: Install the CockroachDB binary.
	$(INSTALL) -d -m 755 $(DESTDIR)$(bindir)
	$(INSTALL) -m 755 cockroach$(EXE_EXT) $(DESTDIR)$(bindir)/cockroach$(EXE_EXT)


SHELL := /bin/bash

# ---------------------------------------------------------
# 1. Install system dependencies + Bazelisk + dev tool
# ---------------------------------------------------------
setup:
	@echo ">>> Installing system dependencies..."
	sudo apt update
	sudo apt install -y \
		build-essential \
		cmake \
		ninja-build \
		clang \
		lld \
		git \
		curl \
		unzip \
		python3 \
		python3-pip \
		pkg-config \
		protobuf-compiler

	@echo ">>> Installing Bazelisk (CRDB officially requires this)..."
	curl -LO "https://github.com/bazelbuild/bazelisk/releases/latest/download/bazelisk-linux-amd64"
	chmod +x bazelisk-linux-amd64
	sudo mv bazelisk-linux-amd64 /usr/local/bin/bazel
	@echo ">>> Bazelisk installed at /usr/local/bin/bazel"

	@echo ">>> Installing CockroachDB dev tool..."
	cd ./pkg/cmd/dev && go install .
	@echo ">>> Dev tool installed -> $$HOME/go/bin/dev"

	@echo ">>> Adding GOPATH/bin to PATH..."
	echo 'export PATH="$$HOME/go/bin:$$PATH"' >> ~/.bashrc

	@echo ">>> Setup complete!"
	@echo ">>> CLOSE terminal and REOPEN once for PATH to refresh."

# ---------------------------------------------------------
# 2. Initialize all submodules
# ---------------------------------------------------------
modules:
	git submodule update --init --recursive

# ---------------------------------------------------------
# 3. Run CRDB environment diagnostics
# ---------------------------------------------------------
doctor:
	./dev doctor --debug

# ---------------------------------------------------------
# 4. Build CockroachDB
# ---------------------------------------------------------
build:
	dev build

# ---------------------------------------------------------
# 5. Run CockroachDB (single-node)
# ---------------------------------------------------------
run:
	dev start --insecure

# ---------------------------------------------------------
# 6. Kill running CockroachDB node
# ---------------------------------------------------------
kill:
	pkill -9 cockroach || true

# ---------------------------------------------------------
# 7. Rebuild after code changes
# ---------------------------------------------------------
rebuild: kill build run

# ---------------------------------------------------------
# 8. Open SQL shell
# ---------------------------------------------------------
sql:
	./bin/cockroach sql --insecure

buildshort:
	./dev build short

runshort:
	./cockroach-short start-single-node --insecure --store=type=mem,size=1GiB --vmodule=subscribe=2 --logtostderr 2>error.log

runrtest:
	go run test/reactive/main.go
