const translations = {
  en: {
    "nav.guest": "Guest",
    "nav.admin": "Admin",
    "theme.light": "Light",
    "theme.dark": "Dark",
    "lang.en": "EN",
    "lang.tr": "TR",
    "guest.eyebrow": "Self-hosted event photo delivery",
    "guest.title": "Find your event photos with one selfie.",
    "guest.copy": "Choose an event, upload a selfie, and SpotSelf returns the best local matches from the event archive.",
    "guest.activeEvents": "active events",
    "guest.event": "Event",
    "guest.selfie": "Selfie",
    "guest.threshold": "Match threshold",
    "guest.submit": "Find my photos",
    "guest.matches": "Matches",
    "guest.initial": "Upload a selfie to begin.",
    "guest.noEvents": "No events yet",
    "guest.matching": "Matching selfie against indexed photos...",
    "guest.noMatches": "No matches met the current threshold.",
    "guest.matchesFound": "{count} possible matches found.",
    "guest.similar": "{score}% similar",
    "admin.eyebrow": "Admin console",
    "admin.title": "Upload, index, and distribute event photos.",
    "admin.copy": "Create an event, upload images, then share the guest link with attendees.",
    "admin.create": "Create event",
    "admin.eventName": "Event name",
    "admin.eventNamePlaceholder": "Istanbul Summer Wedding",
    "admin.watermark": "Watermark text",
    "admin.optional": "Optional",
    "admin.retention": "Retention days",
    "admin.upload": "Upload photos",
    "admin.photos": "Photos",
    "admin.uploadButton": "Upload and index",
    "admin.events": "Events",
    "admin.noEvents": "No events created yet.",
    "admin.operations": "Operations",
    "admin.apiToken": "Admin API token",
    "admin.saveToken": "Save token",
    "admin.purge": "Purge expired",
    "admin.tokenSaved": "Token saved locally.",
    "admin.purged": "{count} expired photos removed.",
    "admin.deleteEvent": "Delete event",
    "admin.deleted": "Event deleted.",
    "admin.stats": "{events} events, {photos} photos, {size} stored",
    "admin.uploading": "Uploading and indexing photos...",
    "admin.indexed": "{count} photos indexed.",
    "admin.card": "{count} photos indexed. Retention: {days} days.",
    "admin.openGuest": "Open guest page",
    "admin.download": "Download ZIP",
  },
  tr: {
    "nav.guest": "Misafir",
    "nav.admin": "Yönetim",
    "theme.light": "Açık",
    "theme.dark": "Koyu",
    "lang.en": "EN",
    "lang.tr": "TR",
    "guest.eyebrow": "Kendi sunucunuzda etkinlik fotoğraf dağıtımı",
    "guest.title": "Tek selfie ile etkinlik fotoğraflarınızı bulun.",
    "guest.copy": "Etkinliği seçin, selfie yükleyin; SpotSelf arşivdeki en iyi yerel eşleşmeleri listeler.",
    "guest.activeEvents": "aktif etkinlik",
    "guest.event": "Etkinlik",
    "guest.selfie": "Selfie",
    "guest.threshold": "Eşleşme eşiği",
    "guest.submit": "Fotoğraflarımı bul",
    "guest.matches": "Eşleşmeler",
    "guest.initial": "Başlamak için selfie yükleyin.",
    "guest.noEvents": "Henüz etkinlik yok",
    "guest.matching": "Selfie indekslenmiş fotoğraflarla eşleştiriliyor...",
    "guest.noMatches": "Geçerli eşik değerini karşılayan eşleşme yok.",
    "guest.matchesFound": "{count} olası eşleşme bulundu.",
    "guest.similar": "%{score} benzer",
    "admin.eyebrow": "Yönetim paneli",
    "admin.title": "Etkinlik fotoğraflarını yükleyin, indeksleyin ve dağıtın.",
    "admin.copy": "Etkinlik oluşturun, görselleri yükleyin ve misafir bağlantısını katılımcılarla paylaşın.",
    "admin.create": "Etkinlik oluştur",
    "admin.eventName": "Etkinlik adı",
    "admin.eventNamePlaceholder": "İstanbul Yaz Düğünü",
    "admin.watermark": "Filigran metni",
    "admin.optional": "İsteğe bağlı",
    "admin.retention": "Saklama günü",
    "admin.upload": "Fotoğraf yükle",
    "admin.photos": "Fotoğraflar",
    "admin.uploadButton": "Yükle ve indeksle",
    "admin.events": "Etkinlikler",
    "admin.noEvents": "Henüz etkinlik oluşturulmadı.",
    "admin.operations": "Operasyonlar",
    "admin.apiToken": "Yönetici API anahtarı",
    "admin.saveToken": "Anahtarı kaydet",
    "admin.purge": "Süresi dolanları temizle",
    "admin.tokenSaved": "Anahtar yerel olarak kaydedildi.",
    "admin.purged": "{count} süresi dolmuş fotoğraf temizlendi.",
    "admin.deleteEvent": "Etkinliği sil",
    "admin.deleted": "Etkinlik silindi.",
    "admin.stats": "{events} etkinlik, {photos} fotoğraf, {size} saklanıyor",
    "admin.uploading": "Fotoğraflar yükleniyor ve indeksleniyor...",
    "admin.indexed": "{count} fotoğraf indekslendi.",
    "admin.card": "{count} fotoğraf indekslendi. Saklama: {days} gün.",
    "admin.openGuest": "Misafir sayfasını aç",
    "admin.download": "ZIP indir",
  },
};

const defaultLang = navigator.language.toLowerCase().startsWith("tr") ? "tr" : "en";
const appState = {
  lang: localStorage.getItem("spotself.lang") || defaultLang,
  theme: localStorage.getItem("spotself.theme") || "system",
};

function t(key, vars = {}) {
  const value = (translations[appState.lang] && translations[appState.lang][key]) || translations.en[key] || key;
  return value.replace(/\{(\w+)\}/g, (_, name) => vars[name] ?? "");
}

function applyPreferences() {
  document.documentElement.lang = appState.lang;
  document.documentElement.dataset.theme = appState.theme;
  document.querySelectorAll("[data-i18n]").forEach((node) => {
    node.textContent = t(node.dataset.i18n);
  });
  document.querySelectorAll("[data-i18n-placeholder]").forEach((node) => {
    node.placeholder = t(node.dataset.i18nPlaceholder);
  });
  document.querySelectorAll("[data-lang]").forEach((button) => {
    button.classList.toggle("active", button.dataset.lang === appState.lang);
  });
  document.querySelector("#themeToggle").textContent = appState.theme === "dark" ? t("theme.light") : t("theme.dark");
  window.dispatchEvent(new CustomEvent("spotself:locale"));
}

function authHeaders(extra = {}) {
  const token = sessionStorage.getItem("spotself.adminToken");
  if (token) {
    return { ...extra, Authorization: `Bearer ${token}` };
  }
  return extra;
}

document.querySelectorAll("[data-lang]").forEach((button) => {
  button.addEventListener("click", () => {
    appState.lang = button.dataset.lang;
    localStorage.setItem("spotself.lang", appState.lang);
    applyPreferences();
  });
});

document.querySelector("#themeToggle").addEventListener("click", () => {
  appState.theme = appState.theme === "dark" ? "light" : "dark";
  localStorage.setItem("spotself.theme", appState.theme);
  applyPreferences();
});

applyPreferences();
