VERSION?=$$(cat version.go | grep VERSION | cut -d"=" -f2 | sed 's/"//g' | sed 's/ //g')
GOFMT_FILES?=$$(find . -name '*.go')
PROJECT_BIN?=prometheus-slacker
PROJECT_SRC?=github.com/MikolajGasior/prometheus-slacker

default: build

fmt:
	gofmt -w $(GOFMT_FILES)

fmtcheck:
	@ gofmt_files=$$(gofmt -l $(GOFMT_FILES)); \
	if [[ -n $${gofmt_files} ]]; then \
		echo "The following files fail gofmt:"; \
		echo "$${gofmt_files}"; \
		echo "Run \`make fmt\` to fix this."; \
		exit 1; \
	fi

build:
	mkdir -p target/bin/linux
	mkdir -p target/bin/darwin
	GOOS=linux GOARCH=amd64 go build -v -o target/bin/linux/${PROJECT_BIN} *.go
	GOOS=darwin GOARCH=amd64 go build -v -o target/bin/darwin/${PROJECT_BIN} *.go

release: build
	mkdir -p target/releases
	tar -cvzf target/releases/${PROJECT_BIN}-${VERSION}-linux-amd64.tar.gz -C target/bin/linux ${PROJECT_BIN}
	tar -cvzf target/releases/${PROJECT_BIN}-${VERSION}-darwin-amd64.tar.gz -C target/bin/darwin ${PROJECT_BIN}

.NOTPARALLEL:

.PHONY: fmt build
