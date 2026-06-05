// app.js wires the drop zone to the WASM module. When a PDF is dropped,
// its bytes are passed to pdfa11y.check (registered by the Go runtime),
// and the returned HTML report is rendered inside an iframe so the
// report's own styling stays isolated from the host page.

const drop     = document.getElementById("drop");
const picker   = document.getElementById("picker");
const dropSub  = document.getElementById("dropSub");
const status   = document.getElementById("status");
const result   = document.getElementById("result");

let resultURL  = null;  // tracks the current Blob URL so we can revoke it

function setStatus(text, isError) {
  status.textContent = text;
  status.classList.toggle("err", !!isError);
}

function setDropState(state) {
  drop.classList.remove("drag", "busy", "ready");
  if (state) drop.classList.add(state);
}

(async function init() {
  setDropState("busy");
  dropSub.textContent = "Initialising…";
  try {
    const go = new Go();
    const w = await WebAssembly.instantiateStreaming(fetch("pdfa11y.wasm"), go.importObject);
    // go.run never resolves -- the Go program loops in select{} after
    // setting up window.pdfa11y. We do NOT await it.
    go.run(w.instance);

    // Tiny tick: give Go the chance to set the globals before the
    // first user interaction. In practice the registration is
    // synchronous within go.run, but defer once to be safe.
    await Promise.resolve();

    setStatus("Ready. Pick a PDF or drop one above.");
    setDropState("ready");
    dropSub.textContent = "PDF/UA checks run locally, in your browser.";

    // Populate the scope section's rules table from the WASM module
    // so the list never drifts from what this build actually runs.
    renderRules();
    renderVersion();
  } catch (err) {
    setStatus("Failed to load WebAssembly: " + err.message, true);
    setDropState(null);
  }
})();

// Drag-over visual feedback
["dragenter", "dragover"].forEach(ev => {
  drop.addEventListener(ev, e => {
    e.preventDefault();
    setDropState("drag");
  });
});
["dragleave", "drop"].forEach(ev => {
  drop.addEventListener(ev, e => {
    e.preventDefault();
    setDropState("ready");
  });
});

drop.addEventListener("drop", async e => {
  e.preventDefault();
  const f = e.dataTransfer.files[0];
  if (f) await handleFile(f);
});

picker.addEventListener("change", async e => {
  const f = e.target.files[0];
  if (f) await handleFile(f);
});

function renderVersion() {
  const el = document.getElementById("version");
  if (!el || !window.pdfa11y) return;
  const v = window.pdfa11y.version;
  if (!v) return;
  // Prefix with "v" unless the build did not stamp one in (e.g.
  // local "dev" builds).
  el.textContent = /^\d/.test(v) ? "v" + v : v;
}

function renderRules() {
  if (!window.pdfa11y || !window.pdfa11y.rules) return;
  const rules = window.pdfa11y.rules();
  if (!Array.isArray(rules)) return;

  // Hero count in the alpha aside + scope-section header.
  const alphaCount = document.getElementById("alphaCount");
  const rulesCount = document.getElementById("rulesCount");
  if (alphaCount) alphaCount.textContent = String(rules.length);
  if (rulesCount) rulesCount.textContent = String(rules.length);

  // Sort by ID for a stable layout (engine.All() already sorts, but
  // we re-sort here so future ordering changes do not affect the UI).
  const sorted = rules.slice().sort((a, b) => a.id.localeCompare(b.id));

  const tbody = document.querySelector("#rulesTable tbody");
  if (!tbody) return;
  tbody.replaceChildren();
  for (const r of sorted) {
    const tr = document.createElement("tr");
    const cells = [r.id, r.category, r.title, r.severity];
    cells.forEach((value, idx) => {
      const td = document.createElement("td");
      td.textContent = value;
      if (idx === 3) td.classList.add("sev-" + value);
      tr.appendChild(td);
    });
    tbody.appendChild(tr);
  }
}

async function handleFile(file) {
  if (!window.pdfa11y) {
    setStatus("WebAssembly not ready yet, try again in a moment.", true);
    return;
  }
  setDropState("busy");
  dropSub.textContent = `${file.name} (${Math.round(file.size / 1024)} KB)`;
  setStatus("Reading and analysing…");

  try {
    const bytes = new Uint8Array(await file.arrayBuffer());

    // Yield to the event loop so the busy state actually paints before
    // we block the main thread inside the Go-WASM call. For < 5 MB
    // PDFs the check is essentially instantaneous, but on bigger
    // files this guarantees the UI shows progress.
    await new Promise(r => setTimeout(r, 0));

    const res = window.pdfa11y.check(bytes, file.name);
    if (res.error) {
      setStatus("PDF could not be parsed: " + res.error, true);
      setDropState("ready");
      return;
    }

    const summary = `${res.verdict} · ${res.passed}/${res.total} checks passed · ${res.errors} error(s), ${res.warnings} warning(s)`;
    setStatus(summary);

    // Render the report into an iframe via a Blob URL. srcdoc has size
    // limits in some browsers; blob: URLs do not.
    if (resultURL) URL.revokeObjectURL(resultURL);
    const blob = new Blob([res.html], { type: "text/html" });
    resultURL = URL.createObjectURL(blob);
    result.src = resultURL;
    result.classList.add("active");
    setDropState("ready");
  } catch (err) {
    setStatus("Unexpected error: " + err.message, true);
    setDropState("ready");
  }
}
