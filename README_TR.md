# SpotSelf 📸🤖

SpotSelf; düğünler, festivaller ve kurumsal etkinlikler için tasarlanmış, kendi sunucunuzda barındırabileceğiniz (self-hosted) açık kaynaklı bir yapay zeka fotoğraf dağıtım platformudur. Hafif ve yüksek performanslı yüz tanıma algoritmaları sayesinde, etkinlik katılımcılarının tek bir selfie yükleyerek veya QR kod okutarak saniyeler içinde yalnızca kendi fotoğraflarını bulup indirmelerini sağlar.

Karmaşık ortak klasörlere ve gizlilik endişelerine son. Etkinlik medyanızın ve biyometrik verilerinizin kontrolü %100 sizde kalsın.

[SpotSelf Mimarisi veya Arayüz Görseli]

## ✨ Özellikler

- **Kendi sunucunuzda etkinlik medyası:** Etkinlik metadatasını ve yüklenen fotoğrafları `SPOTSELF_DATA_DIR` altında saklar.
- **Token korumalı misafir bağlantıları:** Her etkinlik için tahmin edilemeyen bir erişim anahtarı üretilir. Misafirler yalnızca oluşturulan etkinlik bağlantısı ile eşleşme yapabilir ve medyayı görebilir.
- **Yönetici korumalı operasyonlar:** Etkinlik oluşturma, fotoğraf yükleme, medya silme, ZIP indirme, istatistik görüntüleme ve süre dolumu temizliği bearer token ile korunur.
- **Kurulumsuz misafir arayüzü:** Katılımcılar tarayıcı bağlantısını açar, selfie yükler ve eşleşen fotoğrafları görür.
- **Saf web konsolu:** Logo desteği, koyu tema, Türkçe/İngilizce dil değişimi ve framework kullanmayan responsive arayüz.
- **CLI otomasyonu:** `spotselfctl`; sağlık kontrolü, etkinlik oluşturma, yükleme, eşleşme, istatistik, silme ve saklama süresi temizliği komutlarını içerir.
- **Yerel eşleştirme sınırı:** Mevcut uygulama deterministik görüntü parmak izi kullanır. Gerçek yüz embedding entegrasyonu `internal/spotself/fingerprint.go` arkasına eklenebilir.

## 🛠️ Teknoloji Yığını ve Mimari

Mevcut uygulama:

- **Backend:** Go standart kütüphane HTTP sunucusu.
- **Frontend:** Saf HTML, CSS ve JavaScript.
- **Depolama:** Yerel dosya sistemi ve JSON manifest.
- **Eşleştirme:** Çalışan yerel temel için görüntü parmak izi yaklaşımı.
- **Dağıtım:** Docker, Docker Compose veya `go run`.

Planlanan entegrasyon noktaları:

- OpenCV, InsightFace, ONNX Runtime veya farklı bir embedding motoru.
- PostgreSQL `pgvector` ya da başka bir vektör deposu.
- MinIO veya AWS S3 gibi S3 uyumlu nesne depolama.
- Arka plan LiveSync yükleme işçileri.

## 🚀 Docker ile Hızlı Başlangıç

SpotSelf'i ayağa kaldırmanın en hızlı yolu Docker Compose kullanmaktır:

```bash
# Depoyu klonlayın
git clone https://github.com/Dijital-Vizyon/spotself.git
cd spotself

# Çevre değişkenleri şablonunu kopyalayın
cp .env.example .env

# Sunucuyu dışarı açmadan önce üretim yönetici token'ı belirleyin
# SPOTSELF_ADMIN_TOKEN=uzun-rastgele-bir-token

# Sistemi başlatın
docker-compose up -d

```

Sistem çalıştıktan sonra `http://localhost:8080/admin` adresini açın, Operasyonlar paneline yönetici token'ını girin, etkinlik oluşturun, fotoğrafları yükleyin ve oluşturulan misafir bağlantısını paylaşın.

## 💻 Yerel Geliştirme

SpotSelf, Docker olmadan da çalışır ve Go dışında ek bir çalışma zamanı bağımlılığı gerektirmez:

```bash
cp .env.example .env
# Uygulamayı dışarı açmadan önce SPOTSELF_ADMIN_TOKEN belirleyin.
# Sadece yerel demo için .env içinde SPOTSELF_ALLOW_NO_AUTH=true kullanabilirsiniz.
go run ./cmd/spotself
```

Etkinlik oluşturmak ve fotoğraf yüklemek için `http://localhost:8080/admin` adresini açın. Oluşturulan misafir bağlantısını katılımcılarla paylaşarak selfie yüklemelerini ve eşleşen görselleri görmelerini sağlayabilirsiniz.
Yalnızca oluşturulan misafir bağlantısını paylaşın; bu bağlantı misafirlerin eşleşme yapması ve medyayı görüntülemesi için gereken etkinlik erişim token'ını içerir.

Kullanışlı komutlar:

```bash
make test
make build
make run
```

Mevcut açık kaynak uygulama, etkinlik metadatasını ve yüklenen fotoğrafları `./data` altında saklar. Eşleştirme motoru, ürünün yerelde eksiksiz çalışması için deterministik bir görüntü parmak izi kullanır; OpenCV, InsightFace, ONNX Runtime veya vektör veritabanı destekli embedding hattı `internal/spotself/fingerprint.go` sınırından entegre edilebilir.

## ⚙️ Yapılandırma

| Değişken | Varsayılan | Açıklama |
| --- | --- | --- |
| `SPOTSELF_ADDR` | `:8080` | HTTP dinleme adresi. |
| `SPOTSELF_DATA_DIR` | `./data` | Manifest ve yüklenen medya için yerel depolama dizini. |
| `SPOTSELF_PUBLIC_URL` | `http://localhost:8080` | Misafir ve indirme bağlantıları için temel URL. |
| `SPOTSELF_MAX_UPLOAD_MB` | `64` | Maksimum multipart istek boyutu. |
| `SPOTSELF_ADMIN_TOKEN` | boş | Üretim yönetim API'leri için gereklidir. Uzun ve rastgele bir değer kullanın. |
| `SPOTSELF_ALLOW_NO_AUTH` | `false` | Sadece geliştirme için yönetici kimlik doğrulama bypass'ı. Açık ağlarda kullanmayın. |
| `SPOTSELF_MAX_IMAGE_PIXELS` | `24000000` | Yükleme/selfie görüntüleri için maksimum çözülmüş piksel sayısı. |

## 🧰 Operasyonlar ve CLI

SpotSelf, fotoğrafçı bilgisayarları ve otomasyon işleri için küçük bir komut satırı istemcisi içerir:

```bash
go run ./cmd/spotselfctl --url http://localhost:8080 health
go run ./cmd/spotselfctl --url http://localhost:8080 --token "$SPOTSELF_ADMIN_TOKEN" create-event -name "Demo Düğün"
go run ./cmd/spotselfctl --url http://localhost:8080 --token "$SPOTSELF_ADMIN_TOKEN" upload -event <event-id> ./photos/*.jpg
go run ./cmd/spotselfctl --url http://localhost:8080 --token "$SPOTSELF_ADMIN_TOKEN" match -event <event-id> ./selfie.jpg
go run ./cmd/spotselfctl --url http://localhost:8080 --token "$SPOTSELF_ADMIN_TOKEN" stats
go run ./cmd/spotselfctl --url http://localhost:8080 --token "$SPOTSELF_ADMIN_TOKEN" purge
```

Yazma/yönetim API'lerini kullanmak için `SPOTSELF_ADMIN_TOKEN` belirleyin. Web yönetim paneli kullanıcıları anahtarı Operasyonlar panelinden geçerli oturum için girebilir; CLI kullanıcıları `--token` parametresini verebilir veya aynı ortam değişkenini kullanabilir. `SPOTSELF_ALLOW_NO_AUTH=true` yalnızca yerel geliştirme içindir.

Operasyon API yüzeyi:

- Herkese açık: `GET /api/health`
- Misafir token'ı gerekir: `GET /api/events/{id}?token=...`, `POST /api/events/{id}/match?token=...`, `GET /media/{eventID}/{file}?token=...`
- Yönetici token'ı gerekir: `GET /api/events`, `POST /api/events`, `GET /api/stats`, `GET /api/events/{id}/photos`, `PATCH /api/events/{id}`, `DELETE /api/events/{id}`, `GET /api/events/{id}/download`, `GET /api/events/{id}/photos/{photoID}`, `DELETE /api/events/{id}/photos/{photoID}`, `POST /api/maintenance/purge`

## 📐 Nasıl Çalışır?

1. **Oluşturma:** Yönetici, web konsolundan veya `spotselfctl` ile etkinlik oluşturur.
2. **Yükleme:** Fotoğrafçı görselleri yönetim panelinden veya CLI ile yükler.
3. **İndeksleme:** SpotSelf fotoğrafı saklar ve yerel görüntü parmak izini hesaplar.
4. **Paylaşma:** Yönetici, etkinlik erişim token'ı içeren oluşturulmuş misafir bağlantısını paylaşır.
5. **Eşleştirme:** Misafir bu bağlantı üzerinden selfie yükler ve yalnızca eşleşen medya URL'lerini alır.

## 🔒 Güvenlik ve Gizlilik

Geleneksel bulut tabanlı fotoğraf dağıtım platformları, yüz verilerinizi herkese açık sunucularda işler ve kaydeder. SpotSelf tam izolasyon sağlar:

* Yönetim API'leri, geliştirme amaçlı no-auth modu açık değilse `SPOTSELF_ADMIN_TOKEN` ister.
* Misafir medya erişimi, etkinlik oluşturulurken üretilen etkinlik erişim token'ını gerektirir.
* Görüntü bombası riskini azaltmak için yüklenen/selfie görselleri decode edilmeden önce boyut kontrolünden geçer.
* Dinamik frontend içeriği HTML string enjeksiyonu yerine DOM API'leri ile oluşturulur.
* Otomatik temizleme, etkinlik saklama günlerine göre süresi dolan etkinlikleri ve indekslenmiş medyayı silebilir.
* Localhost dışındaki dağıtımlarda HTTPS'i ters proxy arkasında kullanın.

## 🤝 Katkıda Bulunma

Katkılarınız, açık kaynak topluluğunu öğrenmek, ilham almak ve üretmek için harika bir yer haline getiriyor. Yapacağınız her katkı **büyük bir değer taşır**.

1. Projeyi Fork edin
2. Özellik Dalınızı Oluşturun (`git checkout -b feature/AmazingFeature`)
3. Değişikliklerinizi Commit edin (`git commit -m 'Add some AmazingFeature'`)
4. Dalınızı Push edin (`git push origin feature/AmazingFeature`)
5. Bir Pull Request açın

## 📄 Lisans

MIT Lisansı ile dağıtılmaktadır. Daha fazla bilgi için `LICENSE` dosyasına bakabilirsiniz.

---

[Mehmet T. AKALIN](https://github.com/makalin) - Digital Vision tarafından geliştirilmekte ve yönetilmektedir.
