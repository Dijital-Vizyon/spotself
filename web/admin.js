const eventForm = document.querySelector("#eventForm");
const uploadForm = document.querySelector("#uploadForm");
const adminEventSelect = document.querySelector("#adminEventSelect");
const eventsEl = document.querySelector("#events");
const eventStatus = document.querySelector("#eventStatus");
const uploadStatus = document.querySelector("#uploadStatus");
const opsStatus = document.querySelector("#opsStatus");
const statsLine = document.querySelector("#statsLine");
const adminToken = document.querySelector("#adminToken");

adminToken.value = sessionStorage.getItem("spotself.adminToken") || "";

window.addEventListener("spotself:locale", () => {
  refreshAdmin().catch((error) => {
    eventStatus.textContent = error.message;
  });
});

async function api(path, options = {}) {
  options.headers = authHeaders(options.headers || {});
  const response = await fetch(path, options);
  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(data.error || "Request failed");
  }
  return data;
}

eventForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const payload = {
    name: document.querySelector("#eventName").value,
    watermark: document.querySelector("#watermark").value,
    retentionDays: Number(document.querySelector("#retention").value),
  };

  try {
    await api("/api/events", {
      method: "POST",
      headers: authHeaders({ "Content-Type": "application/json" }),
      body: JSON.stringify(payload),
    });
    eventForm.reset();
    document.querySelector("#retention").value = 30;
    await loadEvents();
    await loadStats();
  } catch (error) {
    eventStatus.textContent = error.message;
  }
});

uploadForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  uploadStatus.textContent = t("admin.uploading");
  const form = new FormData();
  [...document.querySelector("#photosInput").files].forEach((file) => {
    form.append("photos", file);
  });

  try {
    const eventID = adminEventSelect.value;
    const { photos } = await api(`/api/events/${eventID}/photos`, {
      method: "POST",
      body: form,
    });
    uploadStatus.textContent = t("admin.indexed", { count: photos.length });
    uploadForm.reset();
    await refreshAdmin();
  } catch (error) {
    uploadStatus.textContent = error.message;
  }
});

document.querySelector("#saveToken").addEventListener("click", () => {
  sessionStorage.setItem("spotself.adminToken", adminToken.value.trim());
  opsStatus.textContent = t("admin.tokenSaved");
});

document.querySelector("#purgeExpired").addEventListener("click", async () => {
  try {
    const result = await api("/api/maintenance/purge", {
      method: "POST",
      headers: authHeaders({ "Content-Type": "application/json" }),
      body: "{}",
    });
    opsStatus.textContent = t("admin.purged", { count: result.removedPhotos });
    await refreshAdmin();
  } catch (error) {
    opsStatus.textContent = error.message;
  }
});

eventsEl.addEventListener("click", async (event) => {
  const button = event.target.closest("[data-delete-event]");
  const download = event.target.closest("[data-download-event]");
  if (!button && !download) return;
  try {
    if (button) {
      await api(`/api/events/${encodeURIComponent(button.dataset.deleteEvent)}`, { method: "DELETE" });
      opsStatus.textContent = t("admin.deleted");
      await refreshAdmin();
    } else {
      await downloadEvent(download.dataset.downloadEvent, download.dataset.downloadName);
    }
  } catch (error) {
    opsStatus.textContent = error.message;
  }
});

async function refreshAdmin() {
  await Promise.all([loadEvents(), loadStats()]);
}

async function loadStats() {
  const stats = await api("/api/stats");
  statsLine.textContent = t("admin.stats", {
    events: stats.eventCount,
    photos: stats.photoCount,
    size: formatBytes(stats.totalBytes),
  });
}

async function loadEvents() {
  const { events } = await api("/api/events");
  adminEventSelect.innerHTML = "";
  eventsEl.innerHTML = "";
  eventStatus.textContent = events.length ? "" : t("admin.noEvents");

  events.forEach((event) => {
    adminEventSelect.add(new Option(event.name, event.id));
    eventsEl.append(eventCard(event));
  });
  uploadForm.querySelector("button").disabled = events.length === 0;
  adminEventSelect.disabled = events.length === 0;
}

function eventCard(event) {
  const article = document.createElement("article");
  article.className = "event-card";
  const title = document.createElement("h3");
  title.textContent = event.name;
  const detail = document.createElement("p");
  detail.textContent = t("admin.card", { count: event.photoCount, days: event.retentionDays });
  const code = document.createElement("code");
  code.textContent = event.guestUrl;
  const actions = document.createElement("div");
  actions.className = "event-actions";
  const guest = document.createElement("a");
  guest.href = event.guestUrl;
  guest.textContent = t("admin.openGuest");
  const download = document.createElement("button");
  download.type = "button";
  download.className = "link-button";
  download.dataset.downloadEvent = event.id;
  download.dataset.downloadName = event.slug || event.id;
  download.textContent = t("admin.download");
  const del = document.createElement("button");
  del.type = "button";
  del.className = "link-button danger";
  del.dataset.deleteEvent = event.id;
  del.textContent = t("admin.deleteEvent");
  actions.append(guest, download, del);
  article.append(title, detail, code, actions);
  return article;
}

function formatBytes(bytes) {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  let value = bytes;
  let index = 0;
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index++;
  }
  return `${value.toFixed(index ? 1 : 0)} ${units[index]}`;
}

async function downloadEvent(eventID, name) {
  const response = await fetch(`/api/events/${encodeURIComponent(eventID)}/download`, {
    headers: authHeaders(),
  });
  if (!response.ok) {
    const data = await response.json().catch(() => ({}));
    throw new Error(data.error || "Download failed");
  }
  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `${name || "event"}-photos.zip`;
  document.body.append(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

refreshAdmin().catch((error) => {
  eventStatus.textContent = error.message;
});
