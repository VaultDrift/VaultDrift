# VaultDrift — BRANDING.md

---

## 🏷️ Brand Identity

| Key | Value |
|-----|-------|
| **Name** | VaultDrift |
| **Tagline** | "Your Files. Your Vault. Your Drift." |
| **Alternative Taglines** | "Zero Dependencies. Total Control." / "One Binary. All Your Files." / "Self-Hosted Sync, Reimagined." |
| **Domain** | vaultdrift.com |
| **GitHub** | github.com/vaultdrift/vaultdrift |
| **License** | Apache 2.0 |
| **Category** | Self-Hosted File Sync & Share Platform |
| **Positioning** | Nextcloud/ownCloud/Seafile killer — single Go binary, zero external deps, E2E encrypted |

---

## 🎨 Brand Concept

**VaultDrift** iki güçlü metaforu birleştiriyor:

- **Vault** (Kasa) — Güvenlik, koruma, E2E encryption, zero-knowledge. Dosyaların güvenle saklandığı aşılmaz bir kasa.
- **Drift** (Akış/Sürüklenme) — Seamless sync, dosyaların cihazlar arasında doğal ve sorunsuz akışı. Suyun akışı gibi — engelsiz, sürekli, güvenilir.

**Brand Personality:**
- **Solid** — Kasa gibi sağlam, güvenilir, kırılmaz
- **Fluid** — Sync akışı gibi pürüzsüz, doğal, engelsiz
- **Minimal** — Tek binary, sıfır bağımlılık, karmaşıklık yok
- **Sovereign** — Self-hosted, verinin sahibi kullanıcı, üçüncü taraf yok

---

## 🎨 Color Palette

### Primary Colors

| Name | Hex | RGB | Usage |
|------|-----|-----|-------|
| **Vault Navy** | `#0F1729` | 15, 23, 41 | Dark backgrounds, primary dark |
| **Drift Blue** | `#3B82F6` | 59, 130, 246 | Primary accent, links, CTAs, active states |
| **Drift Cyan** | `#06B6D4` | 6, 182, 212 | Secondary accent, sync indicators, gradients |
| **Steel Gray** | `#1E293B` | 30, 41, 59 | Card backgrounds (dark mode), sidebar |

### Secondary Colors

| Name | Hex | RGB | Usage |
|------|-----|-----|-------|
| **Vault Silver** | `#94A3B8` | 148, 163, 184 | Muted text, secondary labels |
| **Cloud White** | `#F1F5F9` | 241, 245, 249 | Light mode backgrounds |
| **Pure White** | `#FFFFFF` | 255, 255, 255 | Card backgrounds (light mode), text on dark |
| **Slate 900** | `#0F172A` | 15, 23, 42 | Primary text (light mode) |

### Semantic Colors

| Name | Hex | Usage |
|------|-----|-------|
| **Success Green** | `#22C55E` | Synced, upload complete, connected |
| **Warning Amber** | `#F59E0B` | Conflicts, quota warning, attention |
| **Error Red** | `#EF4444` | Errors, delete actions, disconnected |
| **Info Blue** | `#3B82F6` | Notifications, info badges |
| **Encrypt Purple** | `#A855F7` | E2E encryption indicator, locked states |

### Gradient

```css
/* Primary brand gradient — Vault to Drift */
background: linear-gradient(135deg, #0F1729 0%, #1E293B 40%, #3B82F6 100%);

/* Accent gradient — Drift flow */
background: linear-gradient(90deg, #3B82F6 0%, #06B6D4 100%);

/* Subtle card gradient (dark mode) */
background: linear-gradient(180deg, #1E293B 0%, #0F172A 100%);
```

---

## ✏️ Typography

### Primary Font
- **Inter** — UI, body text, navigation, labels
- Weights: 400 (Regular), 500 (Medium), 600 (SemiBold), 700 (Bold)
- Fallback: `system-ui, -apple-system, sans-serif`

### Monospace Font
- **JetBrains Mono** — Code snippets, file paths, terminal output, checksums
- Weights: 400, 500
- Fallback: `ui-monospace, 'Cascadia Code', 'Fira Code', monospace`

### Heading Font
- **Inter** — Same as body, differentiated by weight + size
- H1: 36px / Bold (700)
- H2: 28px / SemiBold (600)
- H3: 22px / SemiBold (600)
- Body: 16px / Regular (400)
- Small: 14px / Regular (400)
- Caption: 12px / Medium (500)

---

## 🖼️ Logo Concept

### Symbol
- **Kasa kapağı + akış** kombinasyonu
- Hexagonal veya shield-shape vault outline
- İçinde sağa doğru akan üç paralel çizgi (drift / data flow)
- Minimal, geometric, tek renk kullanımına uygun
- Favicon'da 16x16'da bile okunabilir olmalı

### Logo Variants
| Variant | Usage |
|---------|-------|
| **Full Logo** | Logo mark + "VaultDrift" wordmark | Website header, README, docs |
| **Logo Mark** | Symbol only | Favicon, app icon, tray icon, small contexts |
| **Wordmark** | "VaultDrift" text only | Inline references, footer |
| **Dark Mode** | White logo on dark background |
| **Light Mode** | Navy logo on light background |
| **Monochrome** | Single color (works in black or white) |

### Tray Icon States
| State | Visual |
|-------|--------|
| **Synced** | Logo mark — static, Drift Blue |
| **Syncing** | Logo mark — animated flow lines (pulse or rotate) |
| **Paused** | Logo mark — gray/muted |
| **Conflict** | Logo mark — Warning Amber overlay dot |
| **Error** | Logo mark — Error Red overlay dot |

---

## 📱 UI Theme Reference

### Dark Mode (Default)
```
Background:       #0F172A (Slate 900)
Surface:          #1E293B (Steel Gray)
Surface Elevated: #334155 (Slate 700)
Border:           #334155 (Slate 700)
Text Primary:     #F1F5F9 (Cloud White)
Text Secondary:   #94A3B8 (Vault Silver)
Text Muted:       #64748B (Slate 500)
Accent:           #3B82F6 (Drift Blue)
Accent Hover:     #2563EB (Blue 600)
```

### Light Mode
```
Background:       #FFFFFF (Pure White)
Surface:          #F1F5F9 (Cloud White)
Surface Elevated: #FFFFFF (Pure White)
Border:           #E2E8F0 (Slate 200)
Text Primary:     #0F172A (Slate 900)
Text Secondary:   #475569 (Slate 600)
Text Muted:       #94A3B8 (Slate 400)
Accent:           #3B82F6 (Drift Blue)
Accent Hover:     #2563EB (Blue 600)
```

### Tailwind 4.1 CSS Variables
```css
@theme {
  --color-vault-navy: #0F1729;
  --color-drift-blue: #3B82F6;
  --color-drift-cyan: #06B6D4;
  --color-steel-gray: #1E293B;
  --color-vault-silver: #94A3B8;
  --color-cloud-white: #F1F5F9;
  --color-encrypt-purple: #A855F7;
  
  --font-sans: 'Inter', system-ui, -apple-system, sans-serif;
  --font-mono: 'JetBrains Mono', ui-monospace, monospace;
}
```

---

## 📝 Copy & Messaging

### One-liner Descriptions

| Context | Copy |
|---------|------|
| **GitHub** | Self-hosted file sync & share. One Go binary. Zero dependencies. E2E encrypted. Replaces Nextcloud/ownCloud/Seafile. |
| **Website Hero** | Your files. Your vault. Your drift. Self-hosted file sync with zero dependencies and end-to-end encryption — in a single binary. |
| **Twitter/X Bio** | Self-hosted file sync & share platform. One Go binary, zero deps, E2E encrypted. The Nextcloud killer. #NOFORKANYMORE |
| **Docker Hub** | Single-binary self-hosted file sync & share with WebDAV, delta sync, E2E encryption, and a modern React UI. |

### Feature Headlines

| Feature | Headline |
|---------|----------|
| **Single Binary** | One binary to rule them all. No PHP, no Python, no Redis, no Nginx. Just `./vaultdrift serve`. |
| **E2E Encryption** | Zero-knowledge encryption. Your keys, your data. The server sees only encrypted blobs. |
| **Delta Sync** | Changed one paragraph? Sync one chunk. Content-defined chunking transfers only what changed. |
| **WebDAV** | Mount your vault as a drive. Native WebDAV support works with macOS Finder, Windows Explorer, and every file manager. |
| **Deduplication** | Same file twice? Stored once. Chunk-level dedup saves 30-60% storage automatically. |
| **CobaltDB** | Powered by CobaltDB — embedded database engine with B+Tree, WAL, MVCC. No MySQL. No PostgreSQL. No Redis. |
| **Share Links** | Share with a link. Password protect. Set expiry. Limit downloads. QR code included. |
| **Multi-Device** | Desktop, CLI, web, mobile. Your files follow you everywhere with real-time sync. |

### Problem → Solution Copy

```
❌ Nextcloud: PHP + MySQL + Redis + Apache + cron + opcache tuning = weekend gone.
✅ VaultDrift: curl install.sh | sh → done.

❌ Seafile: C + Python + MySQL + Memcached. Good luck compiling.
✅ VaultDrift: Single binary, zero dependencies. go build && ./vaultdrift serve.

❌ ownCloud Infinite Scale: Still feels beta. Multiple microservices.
✅ VaultDrift: Production-ready. One process. One binary. One config file.

❌ Syncthing: Great sync, but no web UI, no share links, no WebDAV.
✅ VaultDrift: Sync + Share + WebDAV + Web UI + E2E encryption = complete package.
```

### README Hero Section
```markdown
# VaultDrift

**Your Files. Your Vault. Your Drift.**

Self-hosted file sync & share platform — built from scratch in Go. 
One binary. Zero dependencies. End-to-end encrypted.

- 🔐 **E2E Encryption** — Zero-knowledge, AES-256-GCM, X25519 key exchange
- ⚡ **Delta Sync** — Content-defined chunking, transfer only what changed
- 🌐 **WebDAV** — Mount as a native drive on any OS
- 🔗 **Share Links** — Password, expiry, download limits, QR codes
- 👥 **Multi-User** — RBAC, quotas, TOTP 2FA
- 📦 **Single Binary** — No PHP, no MySQL, no Redis, no Nginx
- 🗄️ **CobaltDB** — Embedded database, no external DB required
- 💾 **S3 Compatible** — Local storage or AWS/MinIO/R2/B2
- 🖥️ **Modern UI** — React 19, dark/light mode, responsive

Replaces: Nextcloud · ownCloud · Seafile · Syncthing (for share use cases)
```

---

## 🐦 Social Media / X Launch Posts

### Launch Thread (Turkish)

**Post 1 — Hook:**
```
Nextcloud kurulumu: PHP + MySQL + Redis + Apache + cron + opcache tuning.

Benim kurulumum: ./vaultdrift serve

VaultDrift — self-hosted file sync & share. Tek Go binary. Sıfır dependency.

🧵 Neden yaptım, nasıl çalışıyor ↓
```

**Post 2 — Problem:**
```
Self-hosted dosya sync dünyasının hali:

• Nextcloud = PHP performans kabusları, her update bir şeyi bozuyor
• Seafile = C + Python + MySQL + Memcached, compile etmek travma
• ownCloud IS = Hala beta hissi, multiple microservices
• Syncthing = Sync güzel ama WebDAV yok, share link yok, web UI yok

Hepsi ya dependency cehennem ya da yarım çözüm.
```

**Post 3 — Solution:**
```
VaultDrift tek binary'de:

✅ File sync (delta sync, sadece değişen chunk transfer)
✅ E2E encryption (zero-knowledge, sunucu şifreli blob görüyor)
✅ WebDAV server (Finder/Explorer'da native drive olarak mount)
✅ Share links (password, expiry, download limit, QR code)
✅ Modern React UI (dark/light mode)
✅ Multi-user + RBAC
✅ Local + S3 storage

Sıfır external dependency. go build && çalış.
```

**Post 4 — Tech:**
```
Teknik detaylar:

• Content-Defined Chunking (Rabin fingerprint) → dosyalar değişken chunk'lara bölünüyor
• Chunk-level dedup → aynı chunk bir kez saklanıyor (%30-60 tasarruf)
• AES-256-GCM per-file encryption + X25519 key exchange
• Merkle tree + Vector clock sync → gerçek conflict resolution
• CobaltDB embedded (kendi yazdığım DB engine)
• Custom S3 client (AWS SDK yok, sıfırdan Sig V4)

Her şey pure Go, CGo bile yok.
```

**Post 5 — CTA:**
```
VaultDrift — Your Files. Your Vault. Your Drift.

⭐ github.com/vaultdrift/vaultdrift
🌐 vaultdrift.com

#NOFORKANYMORE felsefesi: Legacy PHP fork'lamak yerine Go ile sıfırdan yazdım.

Star atın, contribute edin, self-host edin.
Dosyalarınızın sahibi siz olun. ☁️🔐
```

### English Launch Post
```
Introducing VaultDrift ☁️🔐

Self-hosted file sync & share in a single Go binary.

→ Zero dependencies (no PHP, no MySQL, no Redis)
→ E2E encryption (zero-knowledge)
→ Delta sync (content-defined chunking)
→ WebDAV (mount as native drive)
→ Share links with password & expiry
→ Modern React UI with dark mode
→ CobaltDB embedded (no external DB)
→ S3-compatible storage

One binary replaces Nextcloud + ownCloud + Seafile.

⭐ github.com/vaultdrift/vaultdrift

#opensource #selfhosted #golang #NOFORKANYMORE
```

---

## 🖼️ Nano Banana 2 — Logo Prompt

### Primary Logo

```
A modern minimalist logo for "VaultDrift", a self-hosted file sync and share platform. 

The logo combines two concepts: a vault/shield shape and flowing data streams. 

Design: A hexagonal shield outline in deep navy blue (#0F1729), with three horizontal parallel lines inside that curve and flow to the right, representing data drift/sync. The flow lines use a gradient from blue (#3B82F6) to cyan (#06B6D4). 

Style: Geometric, clean, tech-forward. Minimal detail, works at 16x16 favicon size. No text in the logo mark. Flat design, no shadows, no 3D effects. 

Background: Transparent / white. 

Color scheme: Navy (#0F1729), Blue (#3B82F6), Cyan (#06B6D4).
```

### Dark Background Logo

```
A modern minimalist logo for "VaultDrift" on a dark background (#0F172A).

Hexagonal shield outline in white with three flowing horizontal lines inside using a blue (#3B82F6) to cyan (#06B6D4) gradient. The lines represent data flowing/syncing between devices. 

Below the shield: "VaultDrift" wordmark in white, Inter font, semi-bold weight, clean letter spacing.

Style: Geometric, minimal, premium tech aesthetic. Flat design, no shadows. 

Background: Dark navy (#0F172A).
```

### GitHub Social Preview (1280x640)

```
A wide social preview image (1280x640) for "VaultDrift" GitHub repository. 

Left side: VaultDrift logo (hexagonal shield with flowing data lines) in blue-cyan gradient. 

Center-right: Large text "VaultDrift" in white, Inter Bold font. Below it: "Your Files. Your Vault. Your Drift." in silver/gray (#94A3B8), Inter Regular. 

Below tagline: Small icons row representing key features — lock (encryption), arrows (sync), globe (WebDAV), link (share), server (single binary). Icons in Drift Blue (#3B82F6).

Background: Deep gradient from Vault Navy (#0F1729) left to Steel Gray (#1E293B) right, with subtle grid pattern overlay at 5% opacity.

Style: Premium, minimal, developer-focused. No stock photos, no illustrations, pure geometric design.
```

### Infographic — Architecture Overview

```
A technical infographic showing VaultDrift's architecture on dark background (#0F172A).

Title at top: "VaultDrift Architecture" in white Inter Bold.

Center: Layered horizontal blocks representing:
1. Top layer (blue): "Clients" — icons for Web UI, CLI, Desktop, Mobile
2. Middle layer (cyan): "Protocol" — HTTP/2, WebDAV, WebSocket, Sync Protocol
3. Core layer (gradient blue-cyan): "Engine" — VFS, Crypto, Chunker, Sync, Auth
4. Storage layer (navy): "Storage" — Local FS, S3, CobaltDB icons

Connecting lines between layers with subtle glow effects.

Bottom: Key stats — "Zero Dependencies", "Single Binary", "E2E Encrypted", "Delta Sync" in rounded pill badges.

Style: Technical, clean, dark mode, developer-oriented. Geometric shapes, no gradients except specified. Monospace font for technical labels.

Colors: Navy (#0F1729), Blue (#3B82F6), Cyan (#06B6D4), White text, Silver (#94A3B8) for secondary text.
```

### Infographic — Competitive Comparison

```
A comparison infographic "VaultDrift vs The Rest" on dark background (#0F172A).

Left column: VaultDrift logo with green checkmarks (✅) for features.
Right columns: Nextcloud, ownCloud, Seafile, Syncthing logos (simplified/generic) with red X (❌) or yellow warning (⚠️).

Feature rows:
- Single Binary: VaultDrift ✅, others ❌
- Zero Dependencies: VaultDrift ✅, others ❌
- E2E Encryption: VaultDrift ✅, mixed for others
- Delta Sync: VaultDrift ✅, mixed
- WebDAV: VaultDrift ✅, mixed
- Share Links: VaultDrift ✅, Syncthing ❌
- Setup Time: VaultDrift "< 1 min", others "15-30+ min"
- Memory Usage: VaultDrift "~50MB", Nextcloud "500MB+"

Style: Clean table/grid layout, dark mode, color-coded cells. Green for VaultDrift advantages, red for competitor weaknesses.

Colors: Navy background, Blue/Cyan accents, Green (#22C55E) for positive, Red (#EF4444) for negative, Amber (#F59E0B) for partial.
```

---

## 📦 Brand Assets Checklist

| Asset | Format | Sizes | Status |
|-------|--------|-------|--------|
| Logo Mark | SVG, PNG | 512, 256, 128, 64, 32, 16 | ⬜ |
| Full Logo (horizontal) | SVG, PNG | 512h, 256h | ⬜ |
| Full Logo (stacked) | SVG, PNG | 512h, 256h | ⬜ |
| Favicon | ICO, PNG | 32x32, 16x16 | ⬜ |
| Apple Touch Icon | PNG | 180x180 | ⬜ |
| PWA Icons | PNG | 192x192, 512x512 | ⬜ |
| GitHub Social Preview | PNG | 1280x640 | ⬜ |
| Twitter/X Card Image | PNG | 1200x628 | ⬜ |
| OG Image | PNG | 1200x630 | ⬜ |
| Docker Hub Logo | PNG | 512x512 | ⬜ |
| Tray Icon (synced) | PNG, ICO | 32x32, 16x16 | ⬜ |
| Tray Icon (syncing) | PNG, ICO | 32x32, 16x16 (animated) | ⬜ |
| Tray Icon (paused) | PNG, ICO | 32x32, 16x16 | ⬜ |
| Tray Icon (conflict) | PNG, ICO | 32x32, 16x16 | ⬜ |
| Architecture Infographic | PNG | 1920x1080 | ⬜ |
| Comparison Infographic | PNG | 1920x1080 | ⬜ |
| README Banner | PNG | 1200x300 | ⬜ |

---

## 🌐 Website (vaultdrift.com) Sections

```
1. Hero
   - Tagline: "Your Files. Your Vault. Your Drift."
   - Subtitle: Self-hosted file sync & share in a single Go binary.
   - CTA: "Get Started" → install docs, "View on GitHub" → repo
   - Animated logo/hero visual

2. Problem
   - "Self-hosted file sync shouldn't require a PhD in DevOps."
   - Cards showing Nextcloud/ownCloud/Seafile pain points

3. Features
   - 6-8 feature cards with icons (Lucide-style)
   - Single Binary, E2E Encryption, Delta Sync, WebDAV, Share Links, Modern UI, S3 Storage, CobaltDB

4. How It Works
   - 3-step: Download → Configure → Serve
   - Terminal animation showing install + first run

5. Comparison Table
   - VaultDrift vs Nextcloud vs ownCloud vs Seafile vs Syncthing

6. Architecture
   - Simplified architecture diagram
   - Tech highlights (CDC, Merkle sync, zero-knowledge)

7. Documentation
   - Quick links to install, API, CLI, WebDAV, encryption docs

8. Footer
   - GitHub, Twitter/X, License, Docs links
   - "Made with Go. Zero dependencies. #NOFORKANYMORE"
```

---

*VaultDrift — Your Files. Your Vault. Your Drift.*
*Built from scratch. Zero dependencies. Total sovereignty.*
