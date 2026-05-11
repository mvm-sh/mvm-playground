# mvm-playground

Run [mvm](https://github.com/mvm-sh/mvm) entirely in the browser.
The interpreter is compiled to WebAssembly; the page is fully static.

## Quick start

```sh
make build                              # web/main.wasm + web/wasm_exec.js
make serve                              # binds 0.0.0.0:8080 - open http://<host>:8080
```

Open the URL and use the **Mode** selector:

- **Run** — pick a sample (or paste your own Go program), click **Run**.
  The `uuid` and `errors1` samples demonstrate remote imports
  (`github.com/google/uuid`, `github.com/pkg/errors`) — their module sources
  are bundled into the wasm and resolved offline.
- **REPL** — an interactive read-eval-print loop; definitions persist across
  lines, expressions print their value.

The two **Trace** checkboxes apply in both modes: they turn on `set -x`-style
line tracing and bytecode-execution tracing (mirroring mvm's `-x` flag); the
trace shows in the stderr lane (Run) or the transcript (REPL).

To run a third-party package's full test suite, use the CLI:
`mvm test -v github.com/google/uuid`.

URL params: `?mode=run|repl` and `?sample=NAME` preselect the mode/sample.

## Deploy

Any static host works. Upload the contents of `web/`.
Ensure `.wasm` files are served with `Content-Type: application/wasm`.

## Limitations

- No goroutine fairness guarantees beyond what mvm provides;
  long-running programs block the page.
- No persistent storage (program is lost on reload).
- Plain `<textarea>`, no syntax highlighting.
- Multi-file programs aren't supported.
- The wasm bundle embeds the full mvm stdlib (needed by the remote-import
  samples), so it is large (~35 MB uncompressed; serve it gzipped).
- Browser module-proxy fetches fail on CORS; only the bundled modules
  (`github.com/google/uuid`, `github.com/pkg/errors`) resolve in the browser.

## License

BSD-3-Clause. See [LICENSE](LICENCE).
