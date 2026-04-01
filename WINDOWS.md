# VaultDrift Windows Kurulum & Çalıştırma

> **⚠️ Beta**: VaultDrift şu an beta aşamasındadır.

## Hızlı Başlangıç (3 Adım)

### 1. Gereksinimler

- **Go 1.23+** (https://golang.org/dl/)
- **Git** (https://git-scm.com/download/win)
- **PowerShell 5.1+** (Windows 10/11 ile gelir)

### 2. TLS Sertifikası Oluşturma

Yönetici olarak PowerShell açın ve çalıştırın:

```powershell
# Proje dizinine git
cd D:\Codebox\VaultDrift

# Sertifikaları oluştur
powershell -ExecutionPolicy Bypass -File certs\generate-certs.ps1
```

Bu script şunları oluşturur:
- `certs/cert.pem` - TLS sertifikası
- `certs/key.pem` - Özel anahtar
- `data/` - Veri dizini
- `logs/` - Log dizini

> **Not**: Self-signed sertifika kullanıldığı için tarayıcı uyarı gösterecektir. Gelişmiş -> Devam et seçeneğiyle geçebilirsiniz.

### 3. Config'i Düzenle

`config.windows.yaml` dosyasını `config.yaml` olarak kopyalayın:

```bash
copy config.windows.yaml config.yaml
```

**Önemli**: `jwt_secret` değerini değiştirin!

```yaml
auth:
  jwt_secret: BURAYA_RASTGELE_32_KARAKTERLİK_BİR_STRING_YAZIN
```

### 4. Server'ı Derle ve Çalıştır

```bash
# Server'ı derle (CGO gerekli SQLite için)
go build -o vaultdrift-server.exe ./cmd/server

# Admin kullanıcı ile başlat
./vaultdrift-server.exe init --admin-user admin --admin-email admin@example.com

# Server'ı başlat
./vaultdrift-server.exe serve --config config.yaml
```

## Erişim

- **Web UI**: https://localhost:8443
- **Admin**: `admin` / (kurulumda belirtilen şifre)

## CLI Client Kullanımı

```bash
# Client'ı derle
go build -o vaultdrift-cli.exe ./cmd/vaultdrift-cli

# Server ayarla
./vaultdrift-cli.exe config server https://localhost:8443

# Giriş yap (TOTP aktifse --totp-code ekle)
./vaultdrift-cli.exe login

# Dosya listele
./vaultdrift-cli.exe ls

# Dosya yükle
./vaultdrift-cli.exe upload "C:\Users\Siz\Belgeler\dosya.pdf"

# Dosya indir
./vaultdrift-cli.exe download dosya.pdf

# Sync (klasör senkronizasyonu)
./vaultdrift-cli.exe sync "C:\Users\Siz\VaultDrift"

# Daemon mod (değişiklikleri otomatik senkronize et)
./vaultdrift-cli.exe daemon "C:\Users\Siz\VaultDrift"
```

## WebDAV ile Kullanım

VaultDrift'i ağ sürücüsü olarak bağla:

```powershell
# Windows'ta WebDAV bağla
net use Z: https://localhost:8443/webdav

# Şifre sorarsa: kullanıcı adı ve şifreni gir
```

## Config Dosyası Özelleştirme

### HTTP Modu (TLS yok - sadece local test için)

```yaml
server:
  host: 127.0.0.1
  port: 8080
  tls:
    enabled: false
```

### S3 Storage Backend

```yaml
storage:
  backend: s3
  s3:
    endpoint: s3.amazonaws.com
    bucket: vaultdrift-bucket
    region: us-east-1
    access_key: YOUR_ACCESS_KEY
    secret_key: YOUR_SECRET_KEY
```

### Kullanıcı Kaydı Açma

```yaml
users:
  registration_enabled: true   # Herkes kayıt olabilir
  default_quota: 10737418240   # 10GB limit
```

## Log Dosyaları

Log'ları canlı takip et:

```powershell
Get-Content logs/vaultdrift.log -Wait
```

## Sorun Giderme

### "CGO required" hatası
SQLite için CGO gerekli. Go kurulumunuz CGO'yu destekliyorsa sorun olmamalı:

```bash
# CGO desteğini kontrol et
go env CGO_ENABLED
# 1 olmalı
```

### "certificate is not trusted"
Self-signed sertifika normal. Tarayıcıda:
1. `https://localhost:8443` aç
2. "Gelişmiş" veya "Advanced" tıkla
3. "Devam et (localhost)" veya "Proceed" seç

Alternatif olarak `cert.cer` dosyasını yükleyebilirsiniz:
```powershell
# Sertifikayı güvenilir yap
Import-Certificate -FilePath certs\cert.cer -CertStoreLocation Cert:\LocalMachine\Root
```

### Port çakışması
Başka bir uygulama 8443 portunu kullanıyorsa:

```yaml
server:
  port: 8080  # veya başka bir port
```

### Database kilitli
SQLite veritabanı kilitli kalırsa:
1. Server'ı düzgün kapat: `Ctrl+C`
2. Hala sorun varsa `.db` dosyasının yanındaki `-journal` dosyalarını sil

## Docker ile Çalıştırma (Alternatif)

```powershell
# Docker Compose ile
docker-compose up -d

# Veya direkt
docker run -d `
  --name vaultdrift `
  -p 8443:8443 `
  -v ${PWD}/data:/data `
  -e VAULTDRIFT_AUTH_JWT_SECRET=secret `
  vaultdrift/vaultdrift:latest
```

## Önemli Güvenlik Notları

1. **jwt_secret** rastgele ve güçlü bir değer olsun
2. **Production** için gerçek TLS sertifikası kullanın (Let's Encrypt)
3. **Firewall** ayarlarını kontrol edin (port 8443 açık olmalı)
4. **Backup** düzenli alın: `data/` ve `*.db` dosyalarını yedekleyin

## Dosya Yapısı

```
VaultDrift/
├── config.yaml              # Ana config
├── config.windows.yaml      # Windows template
├── vaultdrift-server.exe    # Server binary
├── vaultdrift-cli.exe       # CLI binary
├── certs/                   # TLS sertifikaları
│   ├── cert.pem
│   ├── key.pem
│   └── generate-certs.ps1
├── data/                    # Storage ve database
│   ├── vaultdrift.db
│   └── files/
└── logs/                    # Log dosyaları
    └── vaultdrift.log
```

## Kaynaklar

- [GitHub](https://github.com/vaultdrift/vaultdrift)
- [Dokümantasyon](https://github.com/vaultdrift/vaultdrift#readme)
- [API Referansı](https://github.com/vaultdrift/vaultdrift/blob/main/API.md)

---

*Zero dependencies. One binary. Complete control.*
