GO                ?= $(shell which go)
GIT_UPDATE_INDEX  := $(shell git update-index --refresh)
GIT_VERSION       := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

GOENV_GOOS               := $(shell go env GOOS)
GOENV_GOARCH             := $(shell go env GOARCH)
GOOS                     ?= $(GOENV_GOOS)
GOARCH                   ?= $(GOENV_GOARCH)
GO_BUILD_SRC             := $(shell find . -name \*.go -type f) go.mod go.sum
GO_BUILD_EXTLDFLAGS      :=
GO_BUILD_TAGS            := static
GO_BUILD_TARGET_DEPS     :=
GO_BUILD_FLAGS           := -trimpath
GO_BUILD_LDFLAGS_OPTIMS  :=

ifeq ($(GOOS)/$(GOARCH),$(GOENV_GOOS)/$(GOENV_GOARCH))
GO_BUILD_TARGET          ?= dist/yage
GO_BUILD_VERSION_TARGET  ?= dist/yage-$(GIT_VERSION)
else
GO_BUILD_TARGET          := dist/yage-$(GOOS)-$(GOARCH)
GO_BUILD_VERSION_TARGET  := dist/yage-$(GOOS)-$(GOARCH)-$(GIT_VERSION)
endif # $(GOOS)/$(GOARCH)

ifneq ($(DEBUG),)
GO_BUILD_FLAGS           += -race -gcflags="all=-N -l"
else
GO_BUILD_LDFLAGS_OPTIMS  += -s -w
endif # $(DEBUG)

GO_BUILD_FLAGS_TARGET                    := .go-build-flags
GO_CROSSBUILD_PLATFORMS                  ?= linux/amd64 linux/386 linux/arm linux/arm64 linux/arm/v7 linux/arm/v6
GO_CROSSBUILD_PLATFORMS                  += freebsd/amd64 freebsd/386 freebsd/arm freebsd/arm64 freebsd/arm/v7 freebsd/arm/v6
GO_CROSSBUILD_PLATFORMS                  += openbsd/amd64 openbsd/386 openbsd/arm openbsd/arm64 openbsd/arm/v7 openbsd/arm/v6
GO_CROSSBUILD_PLATFORMS                  += windows/amd64 windows/386 windows/arm
GO_CROSSBUILD_PLATFORMS                  += darwin/amd64 darwin/arm64
GO_CROSSBUILD_LINUX_PLATFORMS            := $(filter linux/%,$(GO_CROSSBUILD_PLATFORMS))
GO_CROSSBUILD_LINUX_PLATFORMS            := $(filter-out linux/arm/%,$(GO_CROSSBUILD_LINUX_PLATFORMS))
GO_CROSSBUILD_LINUX_ARM_PLATFORMS        := $(filter linux/arm/%,$(GO_CROSSBUILD_PLATFORMS))
GO_CROSSBUILD_FREEBSD_PLATFORMS          := $(filter freebsd/%,$(GO_CROSSBUILD_PLATFORMS))
GO_CROSSBUILD_FREEBSD_PLATFORMS          := $(filter-out freebsd/arm/%,$(GO_CROSSBUILD_FREEBSD_PLATFORMS))
GO_CROSSBUILD_FREEBSD_ARM_PLATFORMS      := $(filter freebsd/arm/%,$(GO_CROSSBUILD_PLATFORMS))
GO_CROSSBUILD_OPENBSD_PLATFORMS          := $(filter openbsd/%,$(GO_CROSSBUILD_PLATFORMS))
GO_CROSSBUILD_OPENBSD_PLATFORMS          := $(filter-out openbsd/arm/%,$(GO_CROSSBUILD_OPENBSD_PLATFORMS))
GO_CROSSBUILD_OPENBSD_ARM_PLATFORMS      := $(filter openbsd/arm/%,$(GO_CROSSBUILD_PLATFORMS))
GO_CROSSBUILD_WINDOWS_PLATFORMS          := $(filter windows/%,$(GO_CROSSBUILD_PLATFORMS))
GO_CROSSBUILD_DARWIN_PLATFORMS           := $(filter darwin/%,$(GO_CROSSBUILD_PLATFORMS))
GO_CROSSBUILD_LINUX_TARGET_PATTERN       := dist/yage-linux-%-$(GIT_VERSION)
GO_CROSSBUILD_LINUX_ARM_TARGET_PATTERN   := dist/yage-linux-armv%-$(GIT_VERSION)
GO_CROSSBUILD_FREEBSD_TARGET_PATTERN     := dist/yage-freebsd-%-$(GIT_VERSION)
GO_CROSSBUILD_FREEBSD_ARM_TARGET_PATTERN := dist/yage-freebsd-armv%-$(GIT_VERSION)
GO_CROSSBUILD_OPENBSD_TARGET_PATTERN     := dist/yage-openbsd-%-$(GIT_VERSION)
GO_CROSSBUILD_OPENBSD_ARM_TARGET_PATTERN := dist/yage-openbsd-armv%-$(GIT_VERSION)
GO_CROSSBUILD_WINDOWS_TARGET_PATTERN     := dist/yage-windows-%-$(GIT_VERSION).exe
GO_CROSSBUILD_DARWIN_TARGET_PATTERN      := dist/yage-darwin-%-$(GIT_VERSION)
GO_CROSSBUILD_TARGETS                    := $(patsubst linux/%,$(GO_CROSSBUILD_LINUX_TARGET_PATTERN),$(GO_CROSSBUILD_LINUX_PLATFORMS))
GO_CROSSBUILD_TARGETS                    += $(patsubst linux/arm/v%,$(GO_CROSSBUILD_LINUX_ARM_TARGET_PATTERN),$(GO_CROSSBUILD_LINUX_ARM_PLATFORMS))
GO_CROSSBUILD_TARGETS                    += $(patsubst freebsd/%,$(GO_CROSSBUILD_FREEBSD_TARGET_PATTERN),$(GO_CROSSBUILD_FREEBSD_PLATFORMS))
GO_CROSSBUILD_TARGETS                    += $(patsubst freebsd/arm/v%,$(GO_CROSSBUILD_FREEBSD_ARM_TARGET_PATTERN),$(GO_CROSSBUILD_FREEBSD_ARM_PLATFORMS))
GO_CROSSBUILD_TARGETS                    += $(patsubst openbsd/%,$(GO_CROSSBUILD_OPENBSD_TARGET_PATTERN),$(GO_CROSSBUILD_OPENBSD_PLATFORMS))
GO_CROSSBUILD_TARGETS                    += $(patsubst openbsd/arm/v%,$(GO_CROSSBUILD_OPENBSD_ARM_TARGET_PATTERN),$(GO_CROSSBUILD_OPENBSD_ARM_PLATFORMS))
GO_CROSSBUILD_TARGETS                    += $(patsubst windows/%,$(GO_CROSSBUILD_WINDOWS_TARGET_PATTERN),$(GO_CROSSBUILD_WINDOWS_PLATFORMS))
GO_CROSSBUILD_TARGETS                    += $(patsubst darwin/%,$(GO_CROSSBUILD_DARWIN_TARGET_PATTERN),$(GO_CROSSBUILD_DARWIN_PLATFORMS))

GO_BUILD_EXTLDFLAGS      := $(strip $(GO_BUILD_EXTLDFLAGS))
GO_BUILD_TAGS            := $(strip $(GO_BUILD_TAGS))
GO_BUILD_TARGET_DEPS     := $(strip $(GO_BUILD_TARGET_DEPS))
GO_BUILD_FLAGS           := $(strip $(GO_BUILD_FLAGS))
GO_BUILD_LDFLAGS_OPTIMS  := $(strip $(GO_BUILD_LDFLAGS_OPTIMS))
GO_BUILD_LDFLAGS         := -ldflags '$(GO_BUILD_LDFLAGS_OPTIMS) -X main.Version=$(GIT_VERSION) -extldflags "$(GO_BUILD_EXTLDFLAGS)"'

GO_TOOLS_GOLANGCI_LINT  ?= $(shell $(GO) env GOPATH)/bin/golangci-lint

# ------------------------------------------------------------------------------

all: crossbuild crossbuild-checksums

.PHONY: build-go .FORCE

install:
	CGO_ENABLED=0 $(GO) install -tags $(GO_BUILD_TAGS) $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS)

$(GO_BUILD_FLAGS_TARGET) : .FORCE
	@(echo "GO_VERSION=$(shell $(GO) version)"; \
	  echo "GO_GOOS=$(GOOS)"; \
	  echo "GO_GOARCH=$(GOARCH)"; \
	  echo "GO_BUILD_TAGS=$(GO_BUILD_TAGS)"; \
	  echo "GO_BUILD_FLAGS=$(GO_BUILD_FLAGS)"; \
	  echo 'GO_BUILD_LDFLAGS=$(subst ','\'',$(GO_BUILD_LDFLAGS))') \
	    | cmp -s - $@ \
	        || (echo "GO_VERSION=$(shell $(GO) version)"; \
	            echo "GO_GOOS=$(GOOS)"; \
	            echo "GO_GOARCH=$(GOARCH)"; \
	            echo "GO_BUILD_TAGS=$(GO_BUILD_TAGS)"; \
	            echo "GO_BUILD_FLAGS=$(GO_BUILD_FLAGS)"; \
	            echo 'GO_BUILD_LDFLAGS=$(subst ','\'',$(GO_BUILD_LDFLAGS))') > $@

$(GO_BUILD_TARGET): $(GO_BUILD_VERSION_TARGET)
	@(test -e $@ && unlink $@) || true
	@ln $< $@

$(GO_BUILD_VERSION_TARGET): $(GO_BUILD_SRC) $(GO_GENERATE_TARGET) $(GO_BUILD_FLAGS_TARGET) | $(GO_BUILD_TARGET_DEPS)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -tags $(GO_BUILD_TAGS) $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

crossbuild: $(GO_BUILD_VERSION_TARGET) $(GO_CROSSBUILD_TARGETS)

$(GO_CROSSBUILD_LINUX_ARM_TARGET_PATTERN): $(GO_BUILD_SRC) $(GO_BUILD_FLAGS_TARGET)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=$* $(GO) build -tags $(GO_BUILD_TAGS),crossbuild $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

$(GO_CROSSBUILD_LINUX_TARGET_PATTERN): $(GO_BUILD_SRC) $(GO_BUILD_FLAGS_TARGET)
	CGO_ENABLED=0 GOOS=linux GOARCH=$* $(GO) build -tags $(GO_BUILD_TAGS),crossbuild $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

$(GO_CROSSBUILD_FREEBSD_ARM_TARGET_PATTERN): $(GO_BUILD_SRC) $(GO_BUILD_FLAGS_TARGET)
	CGO_ENABLED=0 GOOS=freebsd GOARCH=arm GOARM=$* $(GO) build -tags $(GO_BUILD_TAGS),crossbuild $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

$(GO_CROSSBUILD_FREEBSD_TARGET_PATTERN): $(GO_BUILD_SRC) $(GO_BUILD_FLAGS_TARGET)
	CGO_ENABLED=0 GOOS=freebsd GOARCH=$* $(GO) build -tags $(GO_BUILD_TAGS),crossbuild $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

$(GO_CROSSBUILD_OPENBSD_ARM_TARGET_PATTERN): $(GO_BUILD_SRC) $(GO_BUILD_FLAGS_TARGET)
	CGO_ENABLED=0 GOOS=openbsd GOARCH=arm GOARM=$* $(GO) build -tags $(GO_BUILD_TAGS),crossbuild $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

$(GO_CROSSBUILD_OPENBSD_TARGET_PATTERN): $(GO_BUILD_SRC) $(GO_BUILD_FLAGS_TARGET)
	CGO_ENABLED=0 GOOS=openbsd GOARCH=$* $(GO) build -tags $(GO_BUILD_TAGS),crossbuild $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

$(GO_CROSSBUILD_WINDOWS_TARGET_PATTERN): $(GO_BUILD_SRC) $(GO_BUILD_FLAGS_TARGET)
	CGO_ENABLED=0 GOOS=windows GOARCH=$* $(GO) build -tags $(GO_BUILD_TAGS),crossbuild $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

$(GO_CROSSBUILD_DARWIN_TARGET_PATTERN): $(GO_BUILD_SRC) $(GO_BUILD_FLAGS_TARGET)
	CGO_ENABLED=0 GOOS=darwin GOARCH=$* $(GO) build -tags $(GO_BUILD_TAGS),crossbuild $(GO_BUILD_FLAGS) $(GO_BUILD_LDFLAGS) -o $@

crossbuild-checksums: dist/checksums

dist/checksums : $(GO_CROSSBUILD_TARGETS)
	cd dist && shasum -a 256 yage-*-* > checksums

# -- go mod --------------------------------------------------------------------

.PHONY: go-mod-verify go-mod-tidy

go-mod-verify:
	$(GO) mod download
	git diff --quiet go.* || git diff --exit-code go.* || exit 1

go-mod-tidy:
	$(GO) mod download
	$(GO) mod tidy

# ------------------------------------------------------------------------------

test:
	go test ./...

lint: $(GO_TOOLS_GOLANGCI_LINT)
	$(GO_TOOLS_GOLANGCI_LINT) run

# -- tools ---------------------------------------------------------------------

.PHONY: tools

tools: $(GO_TOOLS_GOLANGCI_LINT)

$(GO_TOOLS_GOLANGCI_LINT):
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.36.0
