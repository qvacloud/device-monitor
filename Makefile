GO      ?= go
DIST    := dist
LDFLAGS := -s -w
BFLAGS  := CGO_ENABLED=0 $(GO) build -trimpath -ldflags "$(LDFLAGS)"

.PHONY: all check vet build cross windows agent apk clean

all: cross windows apk

check: vet build

vet:
	$(GO) vet ./...

build:
	$(GO) build ./...

agent:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(DIST)/dm-agent ./agent

cross:
	GOOS=linux GOARCH=amd64       $(BFLAGS) -o $(DIST)/dm-agent-amd64   ./agent
	GOOS=linux GOARCH=arm64       $(BFLAGS) -o $(DIST)/dm-agent-arm64   ./agent
	GOOS=linux GOARCH=arm GOARM=7 $(BFLAGS) -o $(DIST)/dm-agent-arm7    ./agent

windows:
	GOOS=windows GOARCH=amd64 $(BFLAGS) -o $(DIST)/dm-agent-windows-amd64.exe ./agent
	GOOS=windows GOARCH=arm64 $(BFLAGS) -o $(DIST)/dm-agent-windows-arm64.exe ./agent

apk:
	./scripts/build-apk.sh

clean:
	rm -rf $(DIST) android/build
