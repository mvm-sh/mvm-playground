GOROOT  := $(shell go env GOROOT)
SAMPLES_SRC := ../mvm/_samples
EXTRA_SAMPLES_SRC := playground/_extra_samples
SAMPLES_DST := playground/_samples
CURATED := fib.go sieve.go generic.go iter1.go

# Third-party modules bundled (as Go-module-proxy zips) so the playground
# resolves their imports offline. Listed as path@query; the query is resolved
# to a concrete version at fetch time and recorded in modules.txt.
MODULES := github.com/google/uuid@latest github.com/pkg/errors@latest
MODULES_DST := playground/_modules

.PHONY: build serve test clean

build: web/wasm_exec.js web/main.wasm

web/wasm_exec.js: $(GOROOT)/lib/wasm/wasm_exec.js
	cp $< $@

$(SAMPLES_DST): $(addprefix $(SAMPLES_SRC)/,$(CURATED)) $(wildcard $(EXTRA_SAMPLES_SRC)/*.go)
	rm -rf $@
	mkdir -p $@
	cp $(addprefix $(SAMPLES_SRC)/,$(CURATED)) $@/
	cp $(EXTRA_SAMPLES_SRC)/*.go $@/
	@touch $@

$(MODULES_DST):
	rm -rf $@
	mkdir -p $@
	@for m in $(MODULES); do \
	  echo "fetching $$m"; \
	  json=$$(go mod download -json $$m) || exit 1; \
	  path=$$(printf '%s\n' "$$json" | sed -n 's/.*"Path": *"\([^"]*\)".*/\1/p'); \
	  ver=$$(printf '%s\n' "$$json"  | sed -n 's/.*"Version": *"\([^"]*\)".*/\1/p'); \
	  zip=$$(printf '%s\n' "$$json"  | sed -n 's/.*"Zip": *"\([^"]*\)".*/\1/p'); \
	  test -n "$$path" -a -n "$$ver" -a -n "$$zip" || { echo "bad go mod download output for $$m"; exit 1; }; \
	  cp "$$zip" "$@/$$(printf '%s' "$$path" | tr / -)@$$ver.zip"; \
	  echo "$$path $$ver" >> $@/modules.txt; \
	done
	@touch $@

web/main.wasm: wasm/main.go $(SAMPLES_DST) $(MODULES_DST) go.sum
	GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o web/main.wasm ./wasm

go.sum: go.mod
	go mod tidy

serve: build
	go run github.com/mvm-sh/mvm -e 'http.ListenAndServe(":8080", http.FileServer(http.Dir("web")))'

test: $(SAMPLES_DST) $(MODULES_DST) go.sum
	go test ./...

clean:
	rm -rf web/main.wasm web/wasm_exec.js $(SAMPLES_DST) $(MODULES_DST) go.sum
