GOROOT  := $(shell go env GOROOT)
SAMPLES_SRC := ../mvm/_samples
SAMPLES_DST := playground/_samples
CURATED := fib.go heap.go maps.go slices.go sort1.go iter1.go context1.go \
           cmp.go json_nested_marshaler.go rangefunc1.go generic.go intf1.go \
           sieve.go

.PHONY: build serve test clean

build: web/wasm_exec.js web/main.wasm

web/wasm_exec.js: $(GOROOT)/lib/wasm/wasm_exec.js
	cp $< $@

$(SAMPLES_DST): $(addprefix $(SAMPLES_SRC)/,$(CURATED))
	rm -rf $@
	mkdir -p $@
	cp $^ $@/
	@touch $@

web/main.wasm: wasm/main.go $(SAMPLES_DST) go.sum
	GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o web/main.wasm ./wasm

go.sum: go.mod
	go mod tidy

serve: build
	go run github.com/mvm-sh/mvm -e 'http.ListenAndServe(":8080", http.FileServer(http.Dir("web")))'

test: $(SAMPLES_DST) go.sum
	go test ./...

clean:
	rm -rf web/main.wasm web/wasm_exec.js $(SAMPLES_DST) go.sum
