all:
	$(eval GITCOMMIT := $(shell git rev-parse --short=8 HEAD))
	go build -ldflags="-X main.gitCommit=$(GITCOMMIT)$(UNCOMMITTED)"
	GOOS=windows GOARCH=amd64 go build -ldflags="-X main.gitCommit=$(GITCOMMIT)$(UNCOMMITTED)" -o gmi-upload-win-amd64.exe
	GOOS=linux GOARCH=amd64 go build -ldflags="-X main.gitCommit=$(GITCOMMIT)$(UNCOMMITTED)" -o gmi-upload-linux-amd64
	GOOS=windows GOARCH=386 go build -ldflags="-X main.gitCommit=$(GITCOMMIT)$(UNCOMMITTED)" -o gmi-upload-win-386.exe
	GOOS=linux GOARCH=386 go build -ldflags="-X main.gitCommit=$(GITCOMMIT)$(UNCOMMITTED)" -o gmi-upload-linux-386
