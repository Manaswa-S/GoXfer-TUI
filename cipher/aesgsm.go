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

// deriveArgon2ID returns Argon2ID key, with 6 passes over 128MB of memory
// over 1 thread, of length keyLen.
func deriveArgon2ID(pwd, salt []byte, keyLen uint32) (key []byte) {
	return argon2.IDKey(pwd, salt, 6, 128*1024, 1, keyLen)
}

// deriveSalt returns len number of random bytes.
func deriveSalt(len int) (salt []byte) {
	salt = make([]byte, len)
	if n, err := rand.Read(salt); err != nil || n != len {
		panic(fmt.Errorf("failed to generate salt: %v", err))
	}
	return
}

// GetKEK returns a Key Encryption Key that can then be used to wrap the CEK.
func (s *AESGSM) GetKEK(pwd []byte) (kek, salt []byte) {
	salt = deriveSalt(32)
	kek = deriveArgon2ID(pwd, salt, 32)
	return
}

// GetKEKWithSalt is similar to GetKEK but uses given salt.
func (s *AESGSM) GetKEKWithSalt(pwd, salt []byte) (kek []byte) {
	kek = deriveArgon2ID(pwd, salt, 32)
	return
}

// GetCEK returns a Content Encryption Key.
func (s *AESGSM) GetCEK() (cek []byte) {
	cek = deriveSalt(32)
	return
}

// Wrap encrypts CEK using KEK.
// Returns the Wrapped Encryption Key (wek), the nonce and any error.
func (s *AESGSM) Wrap(kek, cek []byte) (wek, nonce []byte, err error) {
	// TODO: we can use a different algo here, just to bring in the diversity
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

// UnWrap decrypts WEK using KEK and the nonce,
// and returns the CEK.
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

// Encrypt encrypts raw using CEK,
// and returns the enc data and nonce generated.
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

// Decrypt decrypts enc using cek and nonce,
// and returns the raw.
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

// EncryptFile encrypts the given 'filePath' using the given 'cek' and saves it to 'savePath'.
// It also returns the nonce and any error.
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

// DecryptFile decrypts the given 'filePath' using the given 'cek' and 'nonce'.
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

// GetSHA generates the SHA checksum for the file at 'path'.
// Returns the SHA as base64 standard encoded string.
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

// GetSHABytes generates the SHA checksum for 'data'.
// Returns the SHA as base64 standard encoded string.
func (s *AESGSM) GetSHABytes(data []byte) (sha string, err error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, bytes.NewReader(data)); err != nil {
		return "", err
	}
	sum := hash.Sum(nil)

	return base64.StdEncoding.EncodeToString(sum), nil
}

// GetHMAC generates the HMAC checksum for file at 'path' using 'key'.
// Returns the HMAC as base64 standard encoded string.
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

// GetHMACBytes generates the HMAC checksum for 'data' using 'key'.
// Returns the HMAC as base64 standard encoded string.
func (s *AESGSM) GetHMACBytes(data, key []byte) (hash string, err error) {
	mac := hmac.New(sha256.New, key)
	if _, err = mac.Write(data); err != nil {
		return "", err
	}
	sum := mac.Sum(nil)

	return base64.StdEncoding.EncodeToString(sum), nil
}
