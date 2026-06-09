# SpotSelf 📸🤖

SpotSelf is a self-hosted, open-source AI photo distribution platform designed for events, festivals, and weddings. Using lightweight and high-performance face recognition algorithms, it allows event attendees to instantly find and download only their own photos via a single selfie or QR code scan.

No more messy shared folders, no more privacy concerns. Maintain 100% ownership of your event media and biometric data.

[Türkçe README](README_TR.md)

## ✨ Features

- **Self-hosted event media:** Store event metadata and uploaded photos on your own filesystem under `SPOTSELF_DATA_DIR`.
- **Token-protected guest links:** Each event gets an unguessable guest token. Guests can only match and view photos through the generated event link.
- **Admin-protected operations:** Create events, upload photos, delete media, download ZIP archives, view stats, and purge expired events with an admin bearer token.
- **Zero-install guest UI:** Attendees open a browser link, upload a selfie, and receive matching photo results.
- **Vanilla web console:** Responsive admin and guest pages with logo support, dark theme, English/Turkish language switching, and no frontend framework.
- **CLI automation:** `spotselfctl` supports health checks, event creation, uploads, matching, stats, deletion, and retention purge.
- **Safe local matcher boundary:** The current implementation uses deterministic image fingerprints. Real face embeddings can be integrated behind `internal/spotself/fingerprint.go`.

## 🛠️ Tech Stack & Architecture

Current implementation:

- **Backend:** Go standard library HTTP server.
- **Frontend:** Vanilla HTML, CSS, and JavaScript.
- **Storage:** Local filesystem plus a JSON manifest.
- **Matching:** Perceptual-style image fingerprinting for a working local baseline.
- **Deployment:** Docker, Docker Compose, or `go run`.

Planned integration points:

- OpenCV, InsightFace, ONNX Runtime, or another embedding engine.
- PostgreSQL with `pgvector` or another vector store.
- S3-compatible object storage such as MinIO or AWS S3.
- Background LiveSync ingestion workers.

## 🚀 Quick Start with Docker

The fastest way to get SpotSelf up and running is using Docker Compose:

```bash
# Clone the repository
git clone https://github.com/Dijital-Vizyon/spotself.git
cd spotself

# Copy environment template
cp .env.example .env

# Set a production admin token before exposing the server
# SPOTSELF_ADMIN_TOKEN=replace-with-a-long-random-token

# Fire up the stack
docker-compose up -d

```

Once running, navigate to `http://localhost:8080/admin`, enter the admin token in the Operations panel, create an event, upload photos, and share the generated guest link.

## 💻 Local Development

SpotSelf also runs without Docker and has no external runtime dependencies beyond Go:

```bash
cp .env.example .env
# Set SPOTSELF_ADMIN_TOKEN before exposing the app.
# For local-only demos, set SPOTSELF_ALLOW_NO_AUTH=true in .env.
go run ./cmd/spotself
```

Open `http://localhost:8080/admin` to create an event and upload photos. Share only the generated guest link; it includes the event access token required for guests to match and view media.

Useful commands:

```bash
make test
make build
make run
```

The current open-source implementation stores event metadata and uploaded photos under `./data`. The matching engine uses a deterministic image fingerprint so the full product works locally; the `internal/spotself/fingerprint.go` boundary is where OpenCV, InsightFace, ONNX Runtime, or a vector database-backed embedding pipeline can be integrated.

## ⚙️ Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `SPOTSELF_ADDR` | `:8080` | HTTP listen address. |
| `SPOTSELF_DATA_DIR` | `./data` | Local storage directory for manifests and uploaded media. |
| `SPOTSELF_PUBLIC_URL` | `http://localhost:8080` | Base URL used when generating guest and download links. |
| `SPOTSELF_MAX_UPLOAD_MB` | `64` | Maximum multipart request size. |
| `SPOTSELF_ADMIN_TOKEN` | empty | Required for production admin APIs. Use a long random value. |
| `SPOTSELF_ALLOW_NO_AUTH` | `false` | Development-only bypass for admin auth. Do not enable on public networks. |
| `SPOTSELF_MAX_IMAGE_PIXELS` | `24000000` | Maximum decoded image dimensions before rejecting uploads/selfies. |

## 🧰 Operations & CLI

SpotSelf includes a small command-line client for photographer workstations and automation:

```bash
go run ./cmd/spotselfctl --url http://localhost:8080 health
go run ./cmd/spotselfctl --url http://localhost:8080 --token "$SPOTSELF_ADMIN_TOKEN" create-event -name "Demo Wedding"
go run ./cmd/spotselfctl --url http://localhost:8080 --token "$SPOTSELF_ADMIN_TOKEN" upload -event <event-id> ./photos/*.jpg
go run ./cmd/spotselfctl --url http://localhost:8080 --token "$SPOTSELF_ADMIN_TOKEN" match -event <event-id> ./selfie.jpg
go run ./cmd/spotselfctl --url http://localhost:8080 --token "$SPOTSELF_ADMIN_TOKEN" stats
go run ./cmd/spotselfctl --url http://localhost:8080 --token "$SPOTSELF_ADMIN_TOKEN" purge
```

Set `SPOTSELF_ADMIN_TOKEN` to enable write/admin APIs. Browser admin users can enter the token in the Operations panel for the current session; CLI users can pass `--token` or export the same environment variable. `SPOTSELF_ALLOW_NO_AUTH=true` is intended only for local development.

Operational API surface:

- Public: `GET /api/health`
- Guest token required: `GET /api/events/{id}?token=...`, `POST /api/events/{id}/match?token=...`, `GET /media/{eventID}/{file}?token=...`
- Admin token required: `GET /api/events`, `POST /api/events`, `GET /api/stats`, `GET /api/events/{id}/photos`, `PATCH /api/events/{id}`, `DELETE /api/events/{id}`, `GET /api/events/{id}/download`, `GET /api/events/{id}/photos/{photoID}`, `DELETE /api/events/{id}/photos/{photoID}`, `POST /api/maintenance/purge`

## 📐 How It Works

1. **Create:** An admin creates an event in the browser console or with `spotselfctl`.
2. **Upload:** The event photographer uploads images through the admin panel or CLI.
3. **Index:** SpotSelf stores the photo and computes its local image fingerprint.
4. **Share:** The admin shares the generated guest link, which includes an event access token.
5. **Match:** The guest uploads a selfie through that link and receives only matching media URLs.

## 🔒 Security & Privacy

Traditional cloud distribution platforms ingest, map, and monetize facial data on public servers. SpotSelf ensures complete isolation:

* Admin APIs require `SPOTSELF_ADMIN_TOKEN` unless development-only no-auth mode is explicitly enabled.
* Guest media access requires a per-event access token generated with the event.
* Uploaded/selfie image dimensions are checked before decoding to reduce image bomb risk.
* Dynamic frontend content is rendered with DOM APIs rather than HTML string injection.
* Auto-purge can delete expired events and indexed media according to each event's retention days.
* Use HTTPS behind a reverse proxy when deploying outside localhost.

## 🤝 Contributing

Contributions are what make the open-source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📄 License

Distributed under the MIT License. See `LICENSE` for more information.

---

Maintained by [Mehmet T. AKALIN](https://github.com/makalin) - Digital Vision.
