"use strict";

const $ = (id) => document.getElementById(id);
const statusEl = $("status");
const modeSel = $("mode");
const sampleSel = $("sample");
const traceLine = $("trace-line");
const traceOp = $("trace-op");
const runBtn = $("run");
const replResetBtn = $("repl-reset");
const src = $("src");
const viewer = $("viewer");
const tabsEl = $("tabs");
const parseStats = $("parse-stats");
const aux = $("aux");
const out = $("out");
const outTitle = $("out-title");
const timing = $("timing");
const replForm = $("repl-form");
const replInput = $("repl-input");
const replPrompt = $("repl-prompt");

// USER_SOURCE_KEY is the data-source value used for the editable user-code tab.
// The wasm side names that source "m:playground" (see runMVM in wasm/main.go);
// we strip that name out of the imported-sources list so it doesn't show up
// twice.
const USER_SOURCE_NAME = "m:playground";

let wasmReady = false;
let mode = "run";
let replMore = false; // last mvmReplEval returned an unterminated block
let replStarted = false;

const setStatus = (text, cls) => {
  statusEl.textContent = text;
  statusEl.className = "status " + cls;
};

const escape = (s) => String(s).replace(/[&<>]/g, (c) =>
  ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" }[c])
);

const traceOpts = () => ({ traceLine: traceLine.checked, traceOp: traceOp.checked });

const AUX_TEXT = {
  repl:
    "<h3>Interactive REPL</h3>" +
    "<p>Type a Go statement or expression and press Enter. Definitions persist across lines; " +
    "an unterminated block continues at the <code>&gt;&gt;</code> prompt.</p>" +
    "<p>Expressions print their value after <code>:</code>. <strong>Reset</strong> starts a fresh session. " +
    "The <strong>Trace</strong> checkboxes trace each evaluated line.</p>",
};

function setMode(m) {
  mode = m;
  document.body.dataset.mode = m;
  modeSel.value = m;
  for (const el of document.querySelectorAll(".run-only"))  el.hidden = m !== "run";
  for (const el of document.querySelectorAll(".repl-only")) el.hidden = m !== "repl";
  aux.hidden = m === "run";
  aux.innerHTML = AUX_TEXT[m] || "";
  // viewer is opt-in via selectTab; keep it hidden outside the run-mode tab flow.
  if (m !== "run") viewer.hidden = true;

  runBtn.hidden = m === "repl";
  outTitle.textContent = m === "repl" ? "Session" : "Output";
  timing.textContent = "";

  if (m === "repl") {
    if (!replStarted) startRepl();
    replInput.focus();
  } else {
    src.focus();
  }
}

const render = (result, ms) => {
  const parts = [
    result.stdout && escape(result.stdout),
    result.stderr && `<span class="stderr">${escape(result.stderr)}</span>`,
    result.error  && `<span class="err">${escape(result.error)}</span>`,
  ].filter(Boolean);
  out.innerHTML = parts.join("\n") || '<span class="muted">(no output)</span>';
  timing.textContent = ms != null ? `${ms} ms` : "";
};

// renderSources rebuilds the tab strip from a run's source list and updates
// the parse-stats line. The first tab is always the editable user code; the
// rest are read-only views of every other source the parser loaded
// (stdlib, bundled modules, etc.). null clears the panel back to its
// pre-run state.
function renderSources(sources, ms) {
  // Keep the editable "Source" tab around; replace anything that follows.
  const head = tabsEl.firstElementChild;
  tabsEl.innerHTML = "";
  tabsEl.appendChild(head);

  if (!sources || sources.length === 0) {
    tabsEl.hidden = true;
    parseStats.hidden = true;
    parseStats.textContent = "";
    selectTab("");
    return;
  }

  const imported = sources.filter(s => s.name !== USER_SOURCE_NAME);
  // Surface the synthetic listing first; the rest are imported package files,
  // alphabetized so they're easy to scan.
  imported.sort((a, b) => {
    if (a.name === "<bytecode>") return -1;
    if (b.name === "<bytecode>") return 1;
    return a.name.localeCompare(b.name);
  });
  for (const s of imported) {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "tab";
    btn.dataset.source = s.name;
    btn.title = `${s.name} — ${s.lines} lines, ${s.bytes} bytes`;
    const short = s.name.split("/").pop() || s.name;
    btn.innerHTML = `<span class="name">${escape(short)}</span><span class="lines">${s.lines}</span>`;
    tabsEl.appendChild(btn);
  }

  const totalLines = sources.reduce((n, s) => n + (s.lines | 0), 0);
  parseStats.textContent = `${sources.length} files · ${totalLines.toLocaleString()} lines${ms != null ? ` · ${ms} ms` : ""}`;
  parseStats.hidden = false;
  tabsEl.hidden = false;
  selectTab(""); // default back to the editable Source tab after each run
}

function selectTab(name) {
  for (const t of tabsEl.querySelectorAll(".tab")) {
    t.classList.toggle("active", t.dataset.source === name);
  }
  if (name === "") {
    viewer.hidden = true;
    src.hidden = false;
    return;
  }
  let content = "";
  if (typeof globalThis.mvmLastSource === "function") {
    content = String(globalThis.mvmLastSource(name) ?? "");
  }
  viewer.textContent = content;
  viewer.hidden = false;
  src.hidden = true;
}

tabsEl.addEventListener("click", (e) => {
  const t = e.target.closest(".tab");
  if (!t) return;
  selectTab(t.dataset.source);
});

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
  const pick = wanted || (names.includes("fib.go") ? "fib.go" : names[0]);
  if (!pick) return;
  sampleSel.value = pick;
  loadSample(pick);
}

// loadSample replaces the editable source with the named bundled sample and
// resets the textarea scroll/selection so the first line is in view.
function loadSample(name) {
  src.value = String(globalThis.mvmGetSample(name) ?? "");
  // Defer scroll reset to the next frame: setting .value can race with later
  // layout / focus-driven auto-scrolling, especially when the tabs strip is
  // just becoming visible and the textarea shrinks underneath the focus.
  requestAnimationFrame(() => {
    src.setSelectionRange(0, 0);
    src.scrollTop = 0;
  });
}

async function run() {
  if (!wasmReady || runBtn.disabled || mode === "repl") return;
  if (src.value.trim() === "") {
    render({ error: "(no source - pick a sample or paste a Go program)" }, null);
    return;
  }
  setStatus("running…", "running");
  runBtn.disabled = true;
  await new Promise(requestAnimationFrame); // let the status pill paint
  const t0 = performance.now();
  let result;
  try { result = globalThis.mvmRun(src.value, traceOpts()); }
  catch (e) { result = { error: String(e) }; }
  const ms = Math.round(performance.now() - t0);
  render(result, ms);
  renderSources(result.sources, ms);
  setStatus("ready", "ready");
  runBtn.disabled = false;
}

function startRepl() {
  globalThis.mvmReplReset();
  replStarted = true;
  replMore = false;
  replPrompt.textContent = ">";
  out.innerHTML = '<span class="muted">// mvm REPL - type a statement or expression and press Enter</span>';
}

function appendOut(html) {
  const placeholder = out.querySelector(".muted");
  if (placeholder) placeholder.remove();
  out.insertAdjacentHTML("beforeend", html);
  out.scrollTop = out.scrollHeight;
}

function replSubmit() {
  if (!wasmReady) return;
  const line = replInput.value;
  if (line.trim() === "" && !replMore) return;
  replInput.value = "";
  appendOut(`<span class="echo">${replMore ? "&gt;&gt;" : "&gt;"} ${escape(line)}</span>\n`);
  let r;
  try { r = globalThis.mvmReplEval(line, traceOpts()); }
  catch (e) { appendOut(`<span class="err">${escape(String(e))}</span>\n`); return; }
  if (r.stdout) appendOut(escape(r.stdout));
  if (r.stderr) appendOut(`<span class="stderr">${escape(r.stderr)}</span>`);
  if (r.error)  appendOut(`<span class="err">${escape(r.error)}</span>\n`);
  if (r.result !== "") appendOut(`<span class="result">: ${escape(r.result)}</span>\n`);
  replMore = !!r.more;
  replPrompt.textContent = replMore ? ">>" : ">";
}

modeSel.addEventListener("change", () => setMode(modeSel.value));

sampleSel.addEventListener("change", () => {
  const name = sampleSel.value;
  if (!name) return;
  loadSample(name);
  renderSources(null); // discard tabs/stats from the previous sample's run
});

runBtn.addEventListener("click", run);
replResetBtn.addEventListener("click", () => { startRepl(); replInput.focus(); });

src.addEventListener("keydown", (e) => {
  if ((e.metaKey || e.ctrlKey) && e.key === "Enter") { e.preventDefault(); run(); }
});
replForm.addEventListener("submit", (e) => { e.preventDefault(); replSubmit(); });

(async () => {
  try {
    await bootWasm();
    populateSamples();
    wasmReady = true;
    modeSel.disabled = false;
    runBtn.disabled = false;
    replResetBtn.disabled = false;
    replInput.disabled = false;
    const versionEl = $("mvm-version");
    if (versionEl && typeof globalThis.mvmVersion === "function") {
      versionEl.textContent = String(globalThis.mvmVersion());
    }
    const wantMode = new URLSearchParams(location.search).get("mode");
    setMode(["run", "repl"].includes(wantMode) ? wantMode : "run");
    setStatus("ready", "ready");
  } catch (e) {
    setStatus("init failed", "error");
    out.innerHTML = `<span class="err">${escape(String(e))}</span>`;
  }
})();
