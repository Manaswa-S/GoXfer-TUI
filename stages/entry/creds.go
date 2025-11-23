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
	Key  string
	Pass string
}

func (s *CredsManager) add(idx []byte, creds Remember) {
	entries := s.read()
	for i, entry := range entries {
		if entry.Key == creds.Key {
			entries = append(entries[:i], entries[i+1:]...)
		}
	}

	newEntry := CredElem{
		Index:     hex.EncodeToString(idx),
		Key:       creds.Key,
		CreatedAt: time.Now().Unix(),
		Remember:  true,
		Used:      1,
	}
	entries = append(entries, newEntry)
	s.save(entries)
}

func (s *CredsManager) read() []CredElem {
	data, err := os.ReadFile(consts.CREDS_FILE_PATH)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []CredElem{}
		}
		panic(err)
	}

	entries := make([]CredElem, 0)
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

func (s *CredsManager) Set(creds Remember) {
	idx, err := utils.Rand(16)
	if err != nil {
		panic(err)
	}

	s.add(idx, creds)

	err = keyring.Set(consts.SERVICE_NAME_CREDS, hex.EncodeToString(idx), creds.Pass)
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
		remember = append(remember, Remember{
			Key:  entry.Key,
			Pass: pass,
		})
	}

	return remember
}

func (s *CredsManager) Used(key string) {
	entries := s.read()
	for i, entry := range entries {
		if entry.Key == key {
			entries[i].Used++
			break
		}
	}
	s.save(entries)
}
