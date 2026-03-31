package main

import (
	"encoding/base64"
	"fmt"
	"github.com/vaultdrift/vaultdrift/internal/crypto"
)

func main() {
	password := "admin"
	salt, err := crypto.GenerateSalt()
	if err != nil {
		panic(err)
	}
	hash := crypto.DeriveKeyArgon2idStyle(password, salt, 65536, 3)
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)
	phc := fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=1$%s$%s", encodedSalt, encodedHash)
	fmt.Println(phc)
}
