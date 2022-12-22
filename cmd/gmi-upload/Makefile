all:
	$(eval GITCOMMIT := $(shell git rev-parse --short=8 HEAD))
	go build -ldflags="-X main.gitCommit=$(GITCOMMIT)$(UNCOMMITTED)"
