# VaultDrift TLS Sertifika Oluşturma (Windows PowerShell)
# Yönetici olarak çalıştırın

$certsDir = "$PSScriptRoot\certs"
$dataDir = "$PSScriptRoot\data"
$logsDir = "$PSScriptRoot\logs"

# Dizinleri oluştur
New-Item -ItemType Directory -Force -Path $certsDir | Out-Null
New-Item -ItemType Directory -Force -Path $dataDir | Out-Null
New-Item -ItemType Directory -Force -Path $logsDir | Out-Null

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "VaultDrift TLS Sertifika Oluşturma" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

# Self-signed sertifika oluştur
Write-Host "Self-signed sertifika oluşturuluyor..." -ForegroundColor Yellow

$cert = New-SelfSignedCertificate `
    -DnsName "localhost", "127.0.0.1" `
    -CertStoreLocation "cert:\LocalMachine\My" `
    -KeyAlgorithm RSA `
    -KeyLength 2048 `
    -NotAfter (Get-Date).AddYears(5) `
    -FriendlyName "VaultDrift Local"

Write-Host "Sertifika oluşturuldu: $($cert.Thumbprint)" -ForegroundColor Green

# PFX olarak export
$password = ConvertTo-SecureString -String "vaultdrift" -Force -AsPlainText
$pfxPath = "$certsDir\cert.pfx"

Export-PfxCertificate `
    -Cert $cert `
    -FilePath $pfxPath `
    -Password $password | Out-Null

Write-Host "PFX export edildi: $pfxPath" -ForegroundColor Green

# PEM formatına çevir (PowerShell ile)
$certPath = "$certsDir\cert.pem"
$keyPath = "$certsDir\key.pem"

# Public key export
$certBase64 = [System.Convert]::ToBase64String($cert.RawData, [System.Base64FormattingOptions]::InsertLineBreaks)
@"
-----BEGIN CERTIFICATE-----
$certBase64
-----END CERTIFICATE-----
"@ | Out-File -FilePath $certPath -Encoding ASCII

Write-Host "Certificate export edildi: $certPath" -ForegroundColor Green

# Private key export (PFX'ten)
$pfx = [System.Security.Cryptography.X509Certificates.X509Certificate2]::new($pfxPath, "vaultdrift", [System.Security.Cryptography.X509Certificates.X509KeyStorageFlags]::Exportable)
$privateKey = $pfx.PrivateKey

if ($privateKey) {
    $keyData = $privateKey.ExportPkcs8PrivateKey()
    $keyBase64 = [System.Convert]::ToBase64String($keyData, [System.Base64FormattingOptions]::InsertLineBreaks)
    @"
-----BEGIN PRIVATE KEY-----
$keyBase64
-----END PRIVATE KEY-----
"@ | Out-File -FilePath $keyPath -Encoding ASCII
    Write-Host "Private key export edildi: $keyPath" -ForegroundColor Green
}

Write-Host ""
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "Sertifikalar başarıyla oluşturuldu!" -ForegroundColor Green
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Config ayarları:" -ForegroundColor Yellow
Write-Host "  cert_file: ./certs/cert.pem"
Write-Host "  key_file: ./certs/key.pem"
Write-Host ""
Write-Host "Tarayıcıda güvenlik uyarısı alırsanız:"
Write-Host "  1. cert.cer dosyasını 'Güvenilir Kök Sertifika' olarak kurun, veya"
Write-Host "  2. Gelişmiş -> Devam et (localhost) seçeneğine tıklayın"
Write-Host ""

# Root sertifikası export
$crtPath = "$certsDir\cert.cer"
Export-Certificate -Cert $cert -FilePath $crtPath | Out-Null
Write-Host "Root sertifika export edildi: $crtPath" -ForegroundColor Green
Write-Host "  (Bu dosyayı 'Güvenilir Kök Sertifika' olarak yükleyebilirsiniz)"

Write-Host ""
Read-Host "Devam etmek için Enter'a basın"
