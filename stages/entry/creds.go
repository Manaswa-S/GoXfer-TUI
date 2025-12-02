package entry

import (
	"cmp"
	"encoding/hex"
	"encoding/json"
	"errors"
	"goxfer/tui/consts"
	"goxfer/tui/utils"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/zalando/go-keyring"
)

type CredsManager struct {
}

func NewCredsManager() *CredsManager {
	return &CredsManager{}
}

type CredElem struct {
	Index     string
	Key       string
	CreatedAt int64
	Remember  bool
	Used      int32
}

type Remember struct {
	Key  []byte
	Pass []byte
}

func (s *CredsManager) add(idx string, creds *Remember) {
	newKey := hex.EncodeToString(creds.Key)

	entries := s.read()
	for i, entry := range entries {
		if strings.EqualFold(entry.Key, newKey) {
			entries = append(entries[:i], entries[i+1:]...)
		}
	}

	newEntry := CredElem{
		Index:     idx,
		Key:       newKey,
		CreatedAt: time.Now().Unix(),
		Remember:  true,
		Used:      1,
	}
	entries = append(entries, newEntry)
	s.save(entries)
}

func (s *CredsManager) read() []CredElem {
	entries := make([]CredElem, 0)
	data, err := os.ReadFile(consts.CREDS_FILE_PATH)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return entries
		}
		panic(err)
	}

	valid := json.Valid(data)
	if !valid {
		s.save(entries)
		return entries
	}

	err = json.Unmarshal(data, &entries)
	if err != nil {
		panic(err)
	}

	return entries
}

func (s *CredsManager) save(entries []CredElem) {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(consts.CREDS_FILE_PATH, data, 0600)
	if err != nil {
		panic(err)
	}
}

func (s *CredsManager) Set(creds *Remember) {
	idx, err := utils.Rand(16)
	if err != nil {
		panic(err)
	}
	newIdx := hex.EncodeToString(idx)
	s.add(newIdx, creds)

	err = keyring.Set(consts.SERVICE_NAME_CREDS, newIdx, hex.EncodeToString(creds.Pass))
	if err != nil {
		panic(err)
	}
}

func (s *CredsManager) Get() []Remember {
	entries := s.read()
	slices.SortFunc(entries, func(a, b CredElem) int {
		return cmp.Compare(b.Used, a.Used)
	})

	remember := make([]Remember, 0)
	for _, entry := range entries {
		pass, err := keyring.Get(consts.SERVICE_NAME_CREDS, entry.Index)
		if err != nil {
			continue
		}
		keyBytes, err := hex.DecodeString(entry.Key)
		if err != nil {
			panic(err)
		}
		passBytes, err := hex.DecodeString(pass)
		if err != nil {
			panic(err)
		}
		remember = append(remember, Remember{
			Key:  keyBytes,
			Pass: passBytes,
		})
	}

	return remember
}

func (s *CredsManager) Used(key []byte) {
	entries := s.read()
	keyHex := hex.EncodeToString(key)
	for i, entry := range entries {
		if entry.Key == keyHex {
			entries[i].Used++
			break
		}
	}
	s.save(entries)
}
