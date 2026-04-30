"use strict";

const $ = (id) => document.getElementById(id);
const status = $("status");
const sampleSel = $("sample");
const runBtn = $("run");
const src = $("src");
const out = $("out");
const timing = $("timing");

const setStatus = (text, cls) => {
  status.textContent = text;
  status.className = "status " + cls;
};

const escape = (s) => s.replace(/[&<>]/g, (c) =>
  ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" }[c])
);

const render = (result, ms) => {
  const parts = [
    result.stdout && escape(result.stdout),
    result.stderr && `<span class="stderr">${escape(result.stderr)}</span>`,
    result.error  && `<span class="err">${escape(result.error)}</span>`,
  ].filter(Boolean);
  out.innerHTML = parts.join("\n") || '<span class="muted">(no output)</span>';
  timing.textContent = ms != null ? `${ms} ms` : "";
};

async function bootWasm() {
  const go = new Go();
  const resp = await fetch("main.wasm");
  if (!resp.ok) throw new Error(`fetch main.wasm: ${resp.status}`);
  const { instance } = await WebAssembly.instantiateStreaming(resp, go.importObject);
  go.run(instance); // synchronous prefix runs Go main() to its select{}
}

function populateSamples() {
  const names = [...globalThis.mvmListSamples()].map(String);
  sampleSel.innerHTML = '<option value="">- pick a sample -</option>' +
    names.map(n => `<option value="${n}">${n.replace(/\.go$/, "")}</option>`).join("");
  sampleSel.disabled = false;
  const want = new URLSearchParams(location.search).get("sample");
  const wanted = want && (names.includes(want) ? want
                       : names.includes(want + ".go") ? want + ".go"
                       : null);
  const pick = wanted || (names.includes("fib.go") ? "fib.go" : names[0]);
  if (!pick) return;
  sampleSel.value = pick;
  src.value = String(globalThis.mvmGetSample(pick) ?? "");
}

async function run() {
  if (runBtn.disabled) return; // wasm not ready, or a run is already in flight
  if (src.value.trim() === "") {
    render({ error: "(no source - pick a sample or paste a Go program)" }, null);
    return;
  }
  setStatus("running…", "running");
  runBtn.disabled = true;
  await new Promise(requestAnimationFrame); // let the status pill paint
  const t0 = performance.now();
  let result;
  try { result = globalThis.mvmRun(src.value); }
  catch (e) { result = { error: String(e) }; }
  render(result, Math.round(performance.now() - t0));
  setStatus("ready", "ready");
  runBtn.disabled = false;
}

sampleSel.addEventListener("change", () => {
  const name = sampleSel.value;
  if (!name) return;
  src.value = String(globalThis.mvmGetSample(name) ?? "");
});

runBtn.addEventListener("click", run);

src.addEventListener("keydown", (e) => {
  if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
    e.preventDefault();
    run();
  }
});

(async () => {
  try {
    await bootWasm();
    populateSamples();
    setStatus("ready", "ready");
    runBtn.disabled = false;
  } catch (e) {
    setStatus("init failed", "error");
    out.innerHTML = `<span class="err">${escape(String(e))}</span>`;
  }
})();
