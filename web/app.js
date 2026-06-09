const eventSelect = document.querySelector("#eventSelect");
const eventCount = document.querySelector("#eventCount");
const matchForm = document.querySelector("#matchForm");
const matchesEl = document.querySelector("#matches");
const matchStatus = document.querySelector("#matchStatus");
const thresholdInput = document.querySelector("#thresholdInput");
const thresholdValue = document.querySelector("#thresholdValue");

thresholdInput.addEventListener("input", () => {
  thresholdValue.textContent = Number(thresholdInput.value).toFixed(2);
});

window.addEventListener("spotself:locale", () => {
  loadEvents().catch((error) => {
    matchStatus.textContent = error.message;
  });
});

async function api(path, options = {}) {
  const response = await fetch(path, options);
  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(data.error || "Request failed");
  }
  return data;
}

async function loadEvents() {
  const params = new URLSearchParams(location.search);
  const selected = params.get("event");
  const token = params.get("token");
  const events = selected && token
    ? [await api(`/api/events/${encodeURIComponent(selected)}?token=${encodeURIComponent(token)}`)]
    : (await api("/api/events")).events;
  eventCount.textContent = events.length;
  eventSelect.innerHTML = "";

  if (events.length === 0) {
    const option = new Option(t("guest.noEvents"), "");
    eventSelect.add(option);
    eventSelect.disabled = true;
    matchForm.querySelector("button").disabled = true;
    return;
  }

  events.forEach((event) => {
    const label = `${event.name} (${event.photoCount} photos)`;
    const option = new Option(label, event.id);
    option.selected = selected === event.id || selected === event.slug;
    eventSelect.add(option);
  });
}

matchForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  matchesEl.innerHTML = "";
  matchStatus.textContent = t("guest.matching");

  const form = new FormData();
  form.append("selfie", document.querySelector("#selfieInput").files[0]);
  form.append("threshold", thresholdInput.value);

  try {
    const eventID = eventSelect.value;
    const token = new URLSearchParams(location.search).get("token");
    const path = token
      ? `/api/events/${encodeURIComponent(eventID)}/match?token=${encodeURIComponent(token)}`
      : `/api/events/${encodeURIComponent(eventID)}/match`;
    const { matches } = await api(path, {
      method: "POST",
      body: form,
    });
    renderMatches(matches);
  } catch (error) {
    matchStatus.textContent = error.message;
  }
});

function renderMatches(matches) {
  if (!matches.length) {
    matchStatus.textContent = t("guest.noMatches");
    return;
  }
  matchStatus.textContent = t("guest.matchesFound", { count: matches.length });
  matchesEl.replaceChildren(...matches.map(matchCard));
}

function matchCard({ photo, similarity }) {
  const article = document.createElement("article");
  article.className = "photo-card";
  const link = document.createElement("a");
  link.href = photo.url;
  link.download = "";
  const img = document.createElement("img");
  img.src = photo.url;
  img.alt = photo.originalName;
  const body = document.createElement("div");
  const name = document.createElement("strong");
  name.textContent = photo.originalName;
  const score = document.createElement("p");
  score.className = "score";
  score.textContent = t("guest.similar", { score: Math.round(similarity * 100) });
  link.append(img);
  body.append(name, score);
  article.append(link, body);
  return article;
}

loadEvents().catch((error) => {
  matchStatus.textContent = error.message;
});
