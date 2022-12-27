all:
	# https://stackoverflow.com/a/71765306/263310
	$(eval GITCOMMIT := $(shell git describe --tags --exact-match --match "v*.*.*" || git describe --match "v*.*.*" --tags || git describe --tags || git rev-parse HEAD))
	go build -ldflags="-X main.gitCommit=$(GITCOMMIT)$(UNCOMMITTED)"
	GOOS=windows GOARCH=amd64 go build -ldflags="-X main.gitCommit=$(GITCOMMIT)$(UNCOMMITTED)" -o gmi-upload-win-amd64.exe
	GOOS=linux GOARCH=amd64 go build -ldflags="-X main.gitCommit=$(GITCOMMIT)$(UNCOMMITTED)" -o gmi-upload-linux-amd64
	GOOS=windows GOARCH=386 go build -ldflags="-X main.gitCommit=$(GITCOMMIT)$(UNCOMMITTED)" -o gmi-upload-win-386.exe
	GOOS=linux GOARCH=386 go build -ldflags="-X main.gitCommit=$(GITCOMMIT)$(UNCOMMITTED)" -o gmi-upload-linux-386
