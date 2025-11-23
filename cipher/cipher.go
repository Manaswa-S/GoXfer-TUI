package cipher

// CEK: Content Encryption Key
// KEK: Key Encryption Key
// WEK: Wrapped Encryption Key
type Cipher interface {
	GetKEK(pwd []byte) (kek, salt []byte, err error)
	GetKEKWithSalt(pwd, salt []byte) (kek []byte, err error)

	GetCEK() (cek []byte, err error)
	Wrap(kek, cek []byte) (wek, nonce []byte, err error)
	UnWrap(wek, kek, nonce []byte) (cek []byte, err error)

	Encrypt(cek, raw []byte) (enc, nonce []byte, err error)
	EncryptFile(cek []byte, filePath, savePath string) (nonce []byte, err error)
	Decrypt(cek, enc, nonce []byte) (raw []byte, err error)
	DecryptFile(cek, nonce []byte, filePath, savePath string) (err error)

	GetSHA(path string) (sha string, err error)
	GetSHABytes(data []byte) (sha string, err error)
	GetHMAC(path string, key []byte) (hash string, err error)
	GetHMACBytes(data, key []byte) (hash string, err error)
}
