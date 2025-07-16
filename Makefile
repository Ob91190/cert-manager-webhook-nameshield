GO ?= $(shell which go)
OS ?= $(shell $(GO) env GOOS)
ARCH ?= $(shell $(GO) env GOARCH)

# Docker configuration
REGISTRY := registry.antemeta.io
IMAGE_NAME := cert-manager-webhook-nameshield
IMAGE_REPO := $(REGISTRY)/devops/$(IMAGE_NAME)
IMAGE_TAG := latest

# Build output
OUT := $(shell pwd)/_out

KUBEBUILDER_VERSION=1.28.0

HELM_FILES := $(shell find deploy/nameshield-webhook)

test: _test/kubebuilder-$(KUBEBUILDER_VERSION)-$(OS)-$(ARCH)/etcd _test/kubebuilder-$(KUBEBUILDER_VERSION)-$(OS)-$(ARCH)/kube-apiserver _test/kubebuilder-$(KUBEBUILDER_VERSION)-$(OS)-$(ARCH)/kubectl
	TEST_ASSET_ETCD=_test/kubebuilder-$(KUBEBUILDER_VERSION)-$(OS)-$(ARCH)/etcd \
	TEST_ASSET_KUBE_APISERVER=_test/kubebuilder-$(KUBEBUILDER_VERSION)-$(OS)-$(ARCH)/kube-apiserver \
	TEST_ASSET_KUBECTL=_test/kubebuilder-$(KUBEBUILDER_VERSION)-$(OS)-$(ARCH)/kubectl \
	$(GO) test -v .

_test/kubebuilder-$(KUBEBUILDER_VERSION)-$(OS)-$(ARCH).tar.gz: | _test
	curl -fsSL https://go.kubebuilder.io/test-tools/$(KUBEBUILDER_VERSION)/$(OS)/$(ARCH) -o $@

_test/kubebuilder-$(KUBEBUILDER_VERSION)-$(OS)-$(ARCH)/etcd _test/kubebuilder-$(KUBEBUILDER_VERSION)-$(OS)-$(ARCH)/kube-apiserver _test/kubebuilder-$(KUBEBUILDER_VERSION)-$(OS)-$(ARCH)/kubectl: _test/kubebuilder-$(KUBEBUILDER_VERSION)-$(OS)-$(ARCH).tar.gz | _test/kubebuilder-$(KUBEBUILDER_VERSION)-$(OS)-$(ARCH)
	tar xfO $< kubebuilder/bin/$(notdir $@) > $@ && chmod +x $@

.PHONY: clean
clean:
	rm -r _test $(OUT)

# Docker targets
.PHONY: docker-build docker-push docker-run docker-test

# Build Docker image
docker-build:
	@echo "🏗️  Building Docker image: $(IMAGE_REPO):$(IMAGE_TAG)"
	docker build -t $(IMAGE_REPO):$(IMAGE_TAG) .
	@echo "✅ Build completed: $(IMAGE_REPO):$(IMAGE_TAG)"

# Push Docker image
docker-push: docker-build
	@echo "🚀 Pushing Docker image: $(IMAGE_REPO):$(IMAGE_TAG)"
	docker push $(IMAGE_REPO):$(IMAGE_TAG)
	@echo "✅ Push completed: $(IMAGE_REPO):$(IMAGE_TAG)"

# Run Docker image locally
docker-run: docker-build
	@echo "🔄 Running Docker image: $(IMAGE_REPO):$(IMAGE_TAG)"
	docker run --rm -p 443:443 $(IMAGE_REPO):$(IMAGE_TAG)

# Test Docker image
docker-test: docker-build
	@echo "🧪 Testing Docker image: $(IMAGE_REPO):$(IMAGE_TAG)"
	docker run --rm $(IMAGE_REPO):$(IMAGE_TAG) --help

# Clean Docker images
docker-clean:
	@echo "🧹 Cleaning Docker images"
	docker rmi $(IMAGE_REPO):$(IMAGE_TAG) || true
	docker system prune -f

# Development targets
.PHONY: build run clean

# Build the webhook binary
build:
	@echo "🔨 Building webhook binary"
	CGO_ENABLED=0 $(GO) build -o webhook -ldflags '-w -extldflags "-static"' .

# Run locally (requires valid kubeconfig)
run: build
	@echo "🔄 Running webhook locally"
	./webhook

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts"
	rm -f webhook
	rm -rf $(OUT)
	rm -rf _test/

# Helm targets
.PHONY: helm-lint helm-template helm-package

# Lint Helm chart
helm-lint:
	@echo "🔍 Linting Helm chart"
	helm lint deploy/nameshield-webhook

# Generate Helm templates
helm-template:
	@echo "📋 Generating Helm templates"
	helm template nameshield-webhook deploy/nameshield-webhook \
		--set groupName=acme.nameshield.webhook \
		--set image.tag=$(IMAGE_TAG)

# Package Helm chart
helm-package:
	@echo "📦 Packaging Helm chart"
	helm package deploy/nameshield-webhook -d $(OUT)

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build the webhook binary"
	@echo "  run            - Run the webhook locally"
	@echo "  test           - Run tests"
	@echo "  clean          - Clean build artifacts"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-push    - Build and push Docker image"
	@echo "  docker-run     - Build and run Docker image locally"
	@echo "  docker-test    - Build and test Docker image"
	@echo "  docker-clean   - Clean Docker images"
	@echo ""
	@echo "Helm targets:"
	@echo "  helm-lint      - Lint Helm chart"
	@echo "  helm-template  - Generate Helm templates"
	@echo "  helm-package   - Package Helm chart"
