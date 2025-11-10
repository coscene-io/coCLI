MAKEGO := make/go
PROJECT := cocli
GO_MODULE := github.com/coscene-io/cocli

include make/cocli/all.mk

.PHONY: test-setup
test-setup: ## Set up test infrastructure
	@echo "Test infrastructure setup complete"
