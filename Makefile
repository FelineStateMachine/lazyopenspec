STATICCHECK := honnef.co/go/tools/cmd/staticcheck@v0.8.2

.PHONY: build test check fmt-check vet lint demo clean

build:
	go build -trimpath -o bin/lazyopenspec .

test:
	go test -race ./...

# What CI runs.
check: fmt-check vet lint test

fmt-check:
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

vet:
	go vet ./...

lint:
	go run $(STATICCHECK) ./...

# Re-record docs/media/demo.gif (needs vhs, ttyd, ffmpeg). See demo/README.md.
demo:
	vhs demo/demo.tape

clean:
	rm -rf bin dist
