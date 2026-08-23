async function getJSON(url) {
  const res = await fetch(url);
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

function renderChambers(rows) {
  const tb = document.querySelector("#chambers tbody");
  tb.innerHTML = "";
  (rows || []).forEach((c) => {
    const tr = document.createElement("tr");
    tr.innerHTML = `<td>${c.code}</td><td>${c.name}</td><td>${c.status}</td><td>${c.max_head_diff_m}m</td>`;
    tb.appendChild(tr);
  });
}

function renderAlerts(rows) {
  const tb = document.querySelector("#alerts tbody");
  tb.innerHTML = "";
  (rows || []).forEach((a) => {
    const tr = document.createElement("tr");
    tr.className = a.severity === "critical" ? "critical" : "warning";
    tr.innerHTML = `<td>${a.severity}</td><td>${a.kind}</td><td>${a.status}</td><td>${a.occurrences}</td><td>${a.message}</td>`;
    tb.appendChild(tr);
  });
}

async function refresh() {
  document.getElementById("clock").textContent = new Date().toISOString();
  try {
    renderChambers(await getJSON("/api/chambers"));
    renderAlerts(await getJSON("/api/alerts?status=open&limit=20"));
  } catch (e) { /* 静默重试 */ }
}

document.getElementById("kpi-form").addEventListener("submit", async (ev) => {
  ev.preventDefault();
  const fd = new FormData(ev.target);
  const url = `/api/kpi/weekly?chamber_id=${fd.get("chamber_id")}&days=${fd.get("days")}`;
  try {
    const kpi = await getJSON(url);
    document.getElementById("kpi-out").textContent = JSON.stringify(kpi, null, 2);
  } catch (e) {
    document.getElementById("kpi-out").textContent = String(e);
  }
});

refresh();
setInterval(refresh, 15000);
