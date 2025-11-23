package core

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"goxfer/tui/cipher"
	"goxfer/tui/utils"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// The idea here is that Core houses ephemeral secrets,
// and because you don't want to have 'Get' methods on those
// secrets, you need to have all that functionality within Core
// that would otherwise be performed by the service layer after
// 'Getting' those secrets.
type Core struct {
	domain *url.URL
	client *http.Client
	routes map[RouteKey]*Route
	cipher cipher.Cipher

	bucKey  []byte
	bucPass []byte
	sessID  []byte // the main session id
	sessKey []byte // the main session key
}

func NewCore(domainStr string, cipher cipher.Cipher) (*Core, error) {
	core := &Core{
		client: &http.Client{},
		cipher: cipher,
	}
	err := core.register(domainStr)
	if err != nil {
		return nil, err
	}

	return core, nil
}

// >>>

func (s *Core) setBucket(key, pwd []byte) error {
	if key == nil || pwd == nil {
		return fmt.Errorf("pwd or key cannot be empty")
	}

	s.Clear(s.bucKey)
	s.Clear(s.bucPass)
	s.bucKey = make([]byte, len(key))
	s.bucPass = make([]byte, len(pwd))
	copy(s.bucKey, key)
	copy(s.bucPass, pwd)

	return nil
}

func (s *Core) SetSession(id, key []byte) error {
	if id == nil || key == nil {
		return errors.New("id or key cannot be empty")
	}
	s.Clear(s.sessID)
	s.Clear(s.sessKey)
	s.sessID = make([]byte, len(id))
	s.sessKey = make([]byte, len(key))
	copy(s.sessID, id)
	copy(s.sessKey, key)

	return nil
}

func (s *Core) DelSession() {
	s.Clear(s.sessID)
	s.Clear(s.sessKey)
	s.Clear(s.bucKey)
	s.Clear(s.bucPass)
}

func (s *Core) Clear(sensi []byte) {
	for i := range sensi {
		sensi[i] = 0
	}
	clear(sensi)
}

// >>>
// Network functionality that uses secrets.

func (s *Core) Hit(rKey RouteKey, queries QueryParams, headers HeaderParams, body *BodyParams) (resp *http.Response, respBody []byte, err error) {
	route, ok := s.routes[rKey]
	if !ok {
		return nil, nil, fmt.Errorf("not a valid route")
	}

	urlStr := route.URL
	if len(queries) > 0 {
		url, err := url.Parse(route.URL)
		if err != nil {
			return nil, nil, err
		}
		q := url.Query()
		for key, value := range queries {
			q.Set(string(key), value)
		}
		url.RawQuery = q.Encode()
		urlStr = url.String()
	}

	if body == nil {
		body = &BodyParams{
			ConType: ConType.Nil,
			Body:    nil,
		}
	}
	var reqBody io.Reader
	if len(body.Body) > 0 {
		reqBody = bytes.NewBuffer(body.Body)
	}

	req, err := http.NewRequest(route.Method, urlStr, reqBody)
	if err != nil {
		return nil, nil, err
	}

	if route.Auth {
		err = s.prepareReq(req, body.Body)
		if err != nil {
			return nil, nil, err
		}
	}

	if body.ConType != ConType.Nil {
		req.Header.Set("Content-Type", body.ConType.String())
	}

	if len(headers) > 0 {
		for key, value := range headers {
			req.Header.Set(string(key), value)
		}
	}

	resp, err = s.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err = io.ReadAll(resp.Body)
	return
}

// Caller should close the body.
func (s *Core) Download(rKey RouteKey, queries QueryParams, downloadId string) (resp *http.Response, err error) {
	route, ok := s.routes[rKey]
	if !ok {
		return nil, fmt.Errorf("not a valid route")
	}

	urlStr := route.URL
	if len(queries) > 0 {
		url, err := url.Parse(route.URL)
		if err != nil {
			return nil, err
		}
		q := url.Query()
		for key, value := range queries {
			q.Set(string(key), value)
		}
		url.RawQuery = q.Encode()
		urlStr = url.String()
	}

	var reqBody io.Reader
	req, err := http.NewRequest(route.Method, urlStr, reqBody)
	if err != nil {
		return nil, err
	}

	if route.Auth {
		err = s.prepareReq(req, []byte{})
		if err != nil {
			return nil, err
		}
	}

	req.Header.Set("X-Download-ID", downloadId)

	resp, err = s.client.Do(req)
	if err != nil {
		return nil, err
	}

	return
}

// // Caller should close the body.
// func (s *Core) Download(rKey RouteKey, queries QueryParams, conType ContentType, body []byte) (resp *http.Response, err error) {
// 	route, ok := s.routes[rKey]
// 	if !ok {
// 		return nil, fmt.Errorf("not a valid route")
// 	}

// 	urlStr := route.URL
// 	if len(queries) > 0 {
// 		url, err := url.Parse(route.URL)
// 		if err != nil {
// 			return nil, err
// 		}
// 		q := url.Query()
// 		for key, value := range queries {
// 			q.Set(string(key), value)
// 		}
// 		url.RawQuery = q.Encode()
// 		urlStr = url.String()
// 	}

// 	var reqBody io.Reader
// 	if len(body) > 0 {
// 		reqBody = bytes.NewBuffer(body)
// 	}

// 	req, err := http.NewRequest(route.Method, urlStr, reqBody)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if route.Auth {
// 		err = s.prepareReq(req, body)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	if conType != ConType.Nil {
// 		req.Header.Set("Content-Type", conType.String())
// 	}

// 	resp, err = s.client.Do(req)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return
// }

func (s *Core) prepareReq(req *http.Request, body []byte) error {
	if s.sessKey == nil || s.sessID == nil {
		return fmt.Errorf("session key or id used before assignment")
	}

	ts := strconv.FormatInt(time.Now().Unix(), 10)

	meta := fmt.Sprintf("%s\n%s\n%s\n%s",
		req.Method,
		req.URL.Path,
		req.URL.RawQuery,
		ts,
	)
	metaHash, err := hash([]byte(meta), s.sessKey)
	if err != nil {
		return err
	}

	bodyHash, err := hash(body, s.sessKey)
	if err != nil {
		return err
	}

	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Session-ID", string(s.sessID))
	req.Header.Set("X-Req-Signature", utils.EncodeBase64(metaHash))
	req.Header.Set("X-Body-Signature", utils.EncodeBase64(bodyHash))

	return nil
}

func hash(data []byte, key []byte) ([]byte, error) {
	mac := hmac.New(sha256.New, key)
	if _, err := mac.Write(data); err != nil {
		return nil, err
	}
	return mac.Sum(nil), nil
}

// >>>
// Cipher functionality that uses secrets.

type BucketCipher struct {
	KEKSalt  []byte
	WEK      []byte
	WEKNonce []byte
}

func (s *Core) NewBucket(pwd []byte) ([]byte, error) {
	bkey, err := s.cipher.GetCEK()
	if err != nil {
		return nil, err
	}

	kek, kekSalt, err := s.cipher.GetKEK(pwd)
	if err != nil {
		return nil, err
	}

	wek, wekNonce, err := s.cipher.Wrap(kek, bkey)
	if err != nil {
		return nil, err
	}

	cipher := BucketCipher{
		KEKSalt:  kekSalt,
		WEK:      wek,
		WEKNonce: wekNonce,
	}
	cipherBytes, err := json.Marshal(cipher)
	if err != nil {
		return nil, err
	}

	return cipherBytes, nil
}

func (s *Core) OpenBucket(key, pwd, cipherData []byte) error {

	cipher := new(BucketCipher)
	err := json.Unmarshal(cipherData, cipher)
	if err != nil {
		return err
	}

	kek, err := s.cipher.GetKEKWithSalt(pwd, cipher.KEKSalt)
	if err != nil {
		return err
	}

	cek, err := s.cipher.UnWrap(cipher.WEK, kek, cipher.WEKNonce)
	if err != nil {
		return fmt.Errorf("unwrap failed: %v", err)
	}

	err = s.setBucket(key, cek)
	if err != nil {
		return err
	}

	return nil
}

func (s *Core) Encrypt(data []byte) (enc, nonce []byte, err error) {
	if len(s.bucPass) == 0 {
		return nil, nil, fmt.Errorf("s.bucPass is len 0")
	}
	return s.cipher.Encrypt(s.bucPass, data)
}

func (s *Core) Decrypt(enc, nonce []byte) (data []byte, err error) {
	return s.cipher.Decrypt(s.bucPass, enc, nonce)
}

func (s *Core) GetKEK() (kek []byte, salt []byte, err error) {
	return s.cipher.GetKEK(s.bucPass)
}

func (s *Core) GetKEKWithSalt(salt []byte) (kek []byte, err error) {
	return s.cipher.GetKEKWithSalt(s.bucPass, salt)
}

func (s *Core) DecryptFile(nonce []byte, filePath, savePath string) (err error) {
	return s.cipher.DecryptFile(s.bucPass, nonce, filePath, savePath)
}
