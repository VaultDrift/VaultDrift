@echo off
chcp 65001 >nul
echo ==========================================
echo VaultDrift Windows TLS Sertifika Oluşturma
echo ==========================================
echo.

REM Dizin oluştur
if not exist certs mkdir certs
if not exist data mkdir data
if not exist logs mkdir logs

REM Self-signed sertifika oluştur (PowerShell ile)
echo Self-signed TLS sertifikası oluşturuluyor...
powershell -Command "$cert = New-SelfSignedCertificate -DnsName 'localhost', '127.0.0.1' -CertStoreLocation cert:\LocalMachine\My -NotAfter (Get-Date).AddYears(5); $pwd = ConvertTo-SecureString -String 'vaultdrift' -Force -AsPlainText; Export-PfxCertificate -Cert $cert -FilePath 'certs/cert.pfx' -Password $pwd; Export-Certificate -Cert $cert -FilePath 'certs/cert.cer'"

REM PFX'ten PEM'e dönüştür (OpenSSL gerekli) veya PowerShell ile export
powershell -Command "
$pwd = 'vaultdrift'
$pfx = Get-PfxCertificate -FilePath 'certs/cert.pfx' -Password (ConvertTo-SecureString -String $pwd -Force -AsPlainText)

# Private key export
$cert = Get-ChildItem -Path Cert:\LocalMachine\My | Where-Object { $_.Thumbprint -eq (Get-PfxCertificate -FilePath 'certs/cert.pfx').Thumbprint }

# Not: Windows'ta PEM export için OpenSSL daha kolay
# Alternatif: cert.pem ve key.pem'i manuel oluştur
"

echo.
echo ==========================================
echo ALTERNATİF: OpenSSL ile (Önerilen)
echo ==========================================
echo.
echo OpenSSL kuruluysa şu komutları çalıştırın:
echo.
echo openssl req -x509 -nodes -days 365 -newkey rsa:2048 ^
echo   -keyout certs/key.pem -out certs/cert.pem ^
echo   -subj "/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
echo.

pause
