GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)
exe = ./cmd/liteswitch-controller/*.go
bin = liteswitch

init: git-hooks

git-hooks:
	$(info INFO: Starting build $@)
	ln -sf ../../.githooks/pre-commit .git/hooks/pre-commit

build:
	$(info INFO: Starting $@)
	rm -f $(bin) 
	@CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o $(bin) $(exe)

start: build
	$(info INFO: Starting $@)
	./$(bin) -kubeconfig=$(CURDIR)/kubeconfig.yaml -configfile=$(CURDIR)/config.json

test: 
	$(info INFO: Starting $@)
	go test -cover ./... -coverprofile=coverage.out

fmt:
	$(info INFO: Starting $@)
	go fmt ./...