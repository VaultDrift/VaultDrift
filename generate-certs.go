// +build ignore

// generate-certs.go - Self-signed TLS sertifika üreten basit Go programı
// Windows/Mac/Linux'ta çalışır, admin yetkisi gerektirmez
//
// Kullanım: go run generate-certs.go
// veya:     go build -o generate-certs.exe generate-certs.go && generate-certs.exe

package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

func main() {
	// Dizin oluştur
	certsDir := "certs"
	dataDir := "data"
	logsDir := "logs"

	for _, dir := range []string{certsDir, dataDir, logsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("Dizin oluşturma hatası: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("==========================================")
	fmt.Println("VaultDrift TLS Sertifika Oluşturma")
	fmt.Println("==========================================")
	fmt.Println()

	// ECC P-256 anahtar çifti oluştur (daha hızlı ve modern)
	fmt.Println("ECC P-256 anahtar çifti oluşturuluyor...")
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Printf("Anahtar oluşturma hatası: %v\n", err)
		os.Exit(1)
	}

	// Sertifika template'i
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"VaultDrift"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // 1 yıl
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	// Self-signed sertifika oluştur
	fmt.Println("Self-signed sertifika oluşturuluyor...")
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		fmt.Printf("Sertifika oluşturma hatası: %v\n", err)
		os.Exit(1)
	}

	// Sertifikayı PEM olarak kaydet
	certPath := filepath.Join(certsDir, "cert.pem")
	certFile, err := os.Create(certPath)
	if err != nil {
		fmt.Printf("Sertifika dosyası oluşturma hatası: %v\n", err)
		os.Exit(1)
	}
	defer certFile.Close()

	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		fmt.Printf("PEM encoding hatası: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Sertifika kaydedildi: %s\n", certPath)

	// Private key'i PEM olarak kaydet
	keyPath := filepath.Join(certsDir, "key.pem")
	keyFile, err := os.Create(keyPath)
	if err != nil {
		fmt.Printf("Anahtar dosyası oluşturma hatası: %v\n", err)
		os.Exit(1)
	}
	defer keyFile.Close()

	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		fmt.Printf("Anahtar marshal hatası: %v\n", err)
		os.Exit(1)
	}

	if err := pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyBytes}); err != nil {
		fmt.Printf("PEM encoding hatası: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Private key kaydedildi: %s\n", keyPath)

	fmt.Println()
	fmt.Println("==========================================")
	fmt.Println("Sertifikalar başarıyla oluşturuldu!")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("Config ayarları:")
	fmt.Printf("  cert_file: ./%s/cert.pem\n", certsDir)
	fmt.Printf("  key_file: ./%s/key.pem\n", certsDir)
	fmt.Println()
	fmt.Println("Tarayıcıda güvenlik uyarısı alırsanız:")
	fmt.Println("  Gelişmiş -> Devam et (localhost) seçeneğine tıklayın")
	fmt.Println()
	fmt.Println("Not: Bu self-signed sertifika sadece local geliştirme içindir.")
	fmt.Println("Production için gerçek bir sertifika (Let's Encrypt vb.) kullanın.")
	fmt.Println()
}
