# mvm-playground

Run [mvm](https://github.com/mvm-sh/mvm) entirely in the browser.
The interpreter is compiled to WebAssembly; the page is fully static.

## Quick start

```sh
make build                              # web/main.wasm + web/wasm_exec.js
make serve                              # binds 0.0.0.0:8080 - open http://<host>:8080
```

Open the URL, pick a sample (or paste your own Go program), click **Run**.

## Deploy

Any static host works (GitHub Pages, Netlify, S3+CloudFront, plain nginx).
Upload the contents of `web/`.
Ensure `.wasm` files are served with `Content-Type: application/wasm`.

## Limitations

- No goroutine fairness guarantees beyond what mvm provides;
  long-running programs block the page.
- No persistent storage (program is lost on reload).
- Plain `<textarea>`, no syntax highlighting.
- Multi-file programs aren't supported.
