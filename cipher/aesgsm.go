package cipher

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/argon2"
)

type AESGSM struct {
} // TODO: Should house configs

func NewAESGSMCipher() *AESGSM {
	return &AESGSM{}
}

//	func derivePBKDF2(pwd, salt []byte, keyLen int) (key []byte) {
//		return pbkdf2.Key(pwd, salt, 204800, keyLen, sha256.New)
//	}
func deriveArgon2ID(pwd, salt []byte, keyLen uint32) (key []byte) {
	return argon2.IDKey(pwd, salt, 6, 128*1024, 1, keyLen)
}
func deriveSalt(len int) (salt []byte) {
	salt = make([]byte, len)
	if n, err := rand.Read(salt); err != nil || n != len {
		panic(fmt.Errorf("failed to generate salt: %v", err))
	}
	return
}

// GetWrappingKey wraps the given 'pwd' using a salt.
// Returns the wrappingKey, salt and any error respectively.
func (s *AESGSM) GetKEK(pwd []byte) (kek, salt []byte, err error) {
	salt = deriveSalt(32)
	kek = deriveArgon2ID(pwd, salt, 32)
	return
}

// GetWrappingKeyUsingSalt wraps the given 'pwd' using the given 'salt'.
// Returns the wrappingKey and any error respectively.
func (s *AESGSM) GetKEKWithSalt(pwd, salt []byte) (kek []byte, err error) {
	return deriveArgon2ID(pwd, salt, 32), nil
}

// GetFileKey returns a random 32 byte long cryptographically safe key.
func (s *AESGSM) GetCEK() (cek []byte, err error) {
	cek = make([]byte, 32)
	if n, err := rand.Read(cek); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to get read for file key: %v", err)
	}
	return
}

// WrapFileKey wraps fileKey using the wrappingkey.
// Returns the wrappedFileKey, the nonce and any error respectively.
// TODO: we can use something different here, just to bring in the diversity
func (s *AESGSM) Wrap(kek, cek []byte) (wek, nonce []byte, err error) {

	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating new cipher block : %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("error wrapping cipher block in GCM : %v", err)
	}

	nonce = deriveSalt(gcm.NonceSize())
	wek = gcm.Seal(nil, nonce, cek, nil)

	return
}

// UpwrapFileKey unwraps wrappedFileKey using wrappingKey and the nonce.
// Returns the fileKey and any error.
func (s *AESGSM) UnWrap(wek, kek, nonce []byte) (cek []byte, err error) {

	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, fmt.Errorf("error creating new cipher block : %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("error wrapping cipher block in GCM : %v", err)
	}

	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("nonce size does not match")
	}

	cek, err = gcm.Open(nil, nonce, wek, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open cipher block : %v", err)
	}

	return
}

func (s *AESGSM) Encrypt(cek, raw []byte) (enc, nonce []byte, err error) {

	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating new cipher block : %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("error wrapping cipher block in GCM : %v", err)
	}

	nonce = deriveSalt(gcm.NonceSize())
	enc = gcm.Seal(nil, nonce, raw, nil)

	return
}

func (s *AESGSM) Decrypt(cek, enc, nonce []byte) (raw []byte, err error) {

	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, fmt.Errorf("error creating new cipher block : %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("error wrapping cipher block in GCM : %v", err)
	}

	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("nonce size does not match")
	}

	raw, err = gcm.Open(nil, nonce, enc, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open cipher block : %v", err)
	}

	return
}

// EncryptFile encrypts the given 'filePath' using the given 'pwd' and saves it to 'savePath'.
// It also returns the nonce and any error.
// Note: The 'pwd' is expected to be cryptographically random and safe, as there is no salt usage
// within EncryptFile.
func (s *AESGSM) EncryptFile(cek []byte, filePath, savePath string) (nonce []byte, err error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	enc, nonce, err := s.Encrypt(cek, raw)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(savePath, enc, 0644)
	if err != nil {
		return nil, err
	}

	return nonce, nil
}

// DecryptFile decrypts the given 'filePath' using the given 'pwd' and 'nonce'.
// It saves the plain file at 'savePath'.
func (s *AESGSM) DecryptFile(cek, nonce []byte, filePath, savePath string) (err error) {
	enc, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	raw, err := s.Decrypt(cek, enc, nonce)
	if err != nil {
		return err
	}

	err = os.WriteFile(savePath, raw, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (s *AESGSM) GetSHA(path string) (sha string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	sum := hash.Sum(nil)

	return base64.StdEncoding.EncodeToString(sum), nil
}

func (s *AESGSM) GetSHABytes(data []byte) (sha string, err error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, bytes.NewReader(data)); err != nil {
		return "", err
	}
	sum := hash.Sum(nil)

	return base64.StdEncoding.EncodeToString(sum), nil
}

func (s *AESGSM) GetHMAC(path string, key []byte) (hash string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	if _, err := io.Copy(mac, file); err != nil {
		return "", err
	}
	sum := mac.Sum(nil)

	return base64.StdEncoding.EncodeToString(sum), nil

}

func (s *AESGSM) GetHMACBytes(data, key []byte) (hash string, err error) {
	mac := hmac.New(sha256.New, key)
	if _, err = mac.Write(data); err != nil {
		return "", err
	}
	sum := mac.Sum(nil)

	return base64.StdEncoding.EncodeToString(sum), nil
}
