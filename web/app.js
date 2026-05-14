"use strict";

const $ = (id) => document.getElementById(id);
const statusEl = $("status");
const sampleSel = $("sample");
const runBtn = $("run");
const runMenuList = $("run-menu-list");
const src = $("src");
const viewer = $("viewer");
const gutter = $("gutter");
const sourceSel = $("source-select");
const parseStats = $("parse-stats");
const out = $("out");

// Name passed to mvmRun on the last run, so the user's file can be excluded
// from the imports list (it's already the editable "Source" option).
let userSourceName = "main.go";

let wasmReady = false;
let lastLineCount = 0;
let traceMode;

const MODE_LABEL = { run: "Run ▾", trace: "Trace ▾", raw: "Raw trace ▾" };

function setTraceMode(mode) {
  traceMode = mode;
  runBtn.textContent = MODE_LABEL[mode];
  for (const item of runMenuList.querySelectorAll("[data-trace]")) {
    item.setAttribute("aria-checked", item.dataset.trace === mode ? "true" : "false");
  }
}

function toggleRunMenu(open) {
  const willOpen = open ?? runMenuList.hidden;
  if (runMenuList.hidden === !willOpen) return;
  runMenuList.hidden = !willOpen;
  runBtn.setAttribute("aria-expanded", String(willOpen));
}

const setStatus = (text, cls) => {
  statusEl.textContent = text;
  statusEl.className = "status " + cls;
};

const escape = (s) => String(s).replace(/[&<>]/g, (c) =>
  ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" }[c])
);

const traceOpts = (mode) => ({ traceLine: mode === "trace", traceOp: mode === "raw" });

const render = (result) => {
  const parts = [
    result.stdout && escape(result.stdout),
    result.stderr && `<span class="stderr">${escape(result.stderr)}</span>`,
    result.error  && `<span class="err">${escape(result.error)}</span>`,
  ].filter(Boolean);
  out.innerHTML = parts.join("\n") || '<span class="muted">(no output)</span>';
};

function renderSources(sources, ms) {
  sourceSel.innerHTML = "";
  const head = document.createElement("option");
  head.value = "";
  head.textContent = "Source";
  sourceSel.appendChild(head);

  const imported = sources ? sources.filter(s => s.name !== userSourceName) : [];
  // Surface the synthetic bytecode listing first; the rest are imported
  // package files, alphabetized so they're easy to scan.
  imported.sort((a, b) => {
    if (a.name === "<bytecode>") return -1;
    if (b.name === "<bytecode>") return 1;
    return a.name.localeCompare(b.name);
  });
  for (const s of imported) {
    const opt = document.createElement("option");
    opt.value = s.name;
    const short = s.name.split("/").pop() || s.name;
    opt.textContent = `${short} (${s.lines})`;
    opt.title = `${s.name} — ${s.lines} lines, ${s.bytes} bytes`;
    sourceSel.appendChild(opt);
  }

  if (sources && sources.length > 0) {
    const totalLines = sources.reduce((n, s) => n + (s.lines | 0), 0);
    parseStats.textContent = `${sources.length} files · ${totalLines.toLocaleString()} lines${ms != null ? ` · ${ms} ms` : ""}`;
    parseStats.hidden = false;
  } else {
    parseStats.hidden = true;
    parseStats.textContent = "";
  }

  sourceSel.disabled = imported.length === 0;
  sourceSel.value = "";
  selectSource("");
}

function selectSource(name) {
  if (name === "") {
    viewer.hidden = true;
    src.hidden = false;
  } else {
    let content = "";
    if (typeof globalThis.mvmLastSource === "function") {
      content = String(globalThis.mvmLastSource(name) ?? "");
    }
    viewer.textContent = content;
    viewer.hidden = false;
    src.hidden = true;
  }
  updateGutter();
}

function updateGutter() {
  const text = viewer.hidden ? src.value : viewer.textContent;
  const count = Math.max(1, (text || "").split("\n").length);
  if (count !== lastLineCount) {
    gutter.textContent = Array.from({ length: count }, (_, i) => i + 1).join("\n");
    lastLineCount = count;
  }
  syncGutterScroll();
}

function syncGutterScroll() {
  gutter.scrollTop = (viewer.hidden ? src : viewer).scrollTop;
}

sourceSel.addEventListener("change", () => selectSource(sourceSel.value));
src.addEventListener("input", updateGutter);
src.addEventListener("scroll", syncGutterScroll);
viewer.addEventListener("scroll", syncGutterScroll);

async function bootWasm() {
  const go = new Go();
  let result;
  if (typeof WebAssembly.instantiateStreaming === "function") {
    try { result = await WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject); }
    catch { /* bad MIME / no streaming support - fall back below */ }
  }
  if (!result) {
    const resp = await fetch("main.wasm");
    if (!resp.ok) throw new Error(`fetch main.wasm: ${resp.status}`);
    result = await WebAssembly.instantiate(await resp.arrayBuffer(), go.importObject);
  }
  go.run(result.instance); // synchronous prefix runs Go main() to its select{}
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
  const pick = wanted || (names.includes("uuid.go") ? "uuid.go" : names[0]);
  if (!pick) return;
  sampleSel.value = pick;
  loadSample(pick);
}

function loadSample(name) {
  src.value = String(globalThis.mvmGetSample(name) ?? "");
  updateGutter();
  // Setting .value races with focus-driven auto-scroll; defer the reset.
  requestAnimationFrame(() => {
    src.setSelectionRange(0, 0);
    src.scrollTop = 0;
    syncGutterScroll();
  });
}

async function run() {
  if (!wasmReady || runBtn.disabled) return;
  if (src.value.trim() === "") {
    render({ error: "(no source - pick a sample or paste a Go program)" });
    return;
  }
  setStatus("running…", "running");
  runBtn.disabled = true;
  await new Promise(requestAnimationFrame); // let the status pill paint
  const t0 = performance.now();
  const opts = { ...traceOpts(traceMode), name: sampleSel.value || "main.go" };
  let result;
  try { result = globalThis.mvmRun(src.value, opts); }
  catch (e) { result = { error: String(e) }; }
  const ms = Math.round(performance.now() - t0);
  userSourceName = result.name || opts.name;
  render(result);
  renderSources(result.sources, ms);
  setStatus("ready", "ready");
  runBtn.disabled = false;
}

sampleSel.addEventListener("change", () => {
  const name = sampleSel.value;
  if (!name) return;
  loadSample(name);
  renderSources(null); // discard listing/stats from the previous sample's run
});

runBtn.addEventListener("click", (e) => { e.stopPropagation(); toggleRunMenu(); });
runMenuList.addEventListener("click", (e) => {
  const item = e.target.closest("[data-trace]");
  if (!item) return;
  setTraceMode(item.dataset.trace);
  toggleRunMenu(false);
  run();
});
document.addEventListener("click", (e) => {
  if (!e.target.closest("#run-menu")) toggleRunMenu(false);
});

src.addEventListener("keydown", (e) => {
  if ((e.metaKey || e.ctrlKey) && e.key === "Enter") { e.preventDefault(); run(); }
});

(async () => {
  try {
    await bootWasm();
    populateSamples();
    wasmReady = true;
    runBtn.disabled = false;
    setTraceMode("run");
    const versionEl = $("mvm-version");
    if (versionEl && typeof globalThis.mvmVersion === "function") {
      versionEl.textContent = String(globalThis.mvmVersion());
    }
    src.focus();
    setStatus("ready", "ready");
  } catch (e) {
    setStatus("init failed", "error");
    out.innerHTML = `<span class="err">${escape(String(e))}</span>`;
  }
})();
