package native

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"goxfer/tui/logger"
	"io"
	"os"
	"sync"
	"time"
)

type Native struct {
	filePath string
	maxSize  int64
	maxTime  int64
	logs     chan *logger.Log
	sessid   string
	writer   io.Writer
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func New(filePath string, maxSize int64, maxTime int64) (*Native, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	native := &Native{
		filePath: filePath,
		maxSize:  maxSize,
		maxTime:  maxTime,
		logs:     make(chan *logger.Log, 25),
		writer:   file,
	}

	native.ctx, native.cancel = context.WithCancel(context.Background())
	native.wg = sync.WaitGroup{}
	native.wg.Add(1)
	go func() {
		defer native.wg.Done()
		if err := native.worker(native.ctx); err != nil {
			panic(err)
		}
	}()

	return native, nil
}

func (s *Native) With(id string) {
	s.sessid = id
}

func (s *Native) Stop() {
	s.cancel()
	s.wg.Wait()
}

func (s *Native) Rotate() error {
	stats, err := os.Stat(s.filePath)
	if err != nil {
		return err
	}

	// TODO:
	if stats.Size() > s.maxSize {
		file, err := os.Open(s.filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		temp, err := os.Create(s.filePath + ".tmp")
		if err != nil {
			return err
		}
		defer temp.Close()

		logs := make([][]byte, 0)
		currSize := int64(0)
		dec := json.NewDecoder(file)
		for {
			log := new(logger.Log)
			if err := dec.Decode(log); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return err
			}
			logBytes, err := json.MarshalIndent(log, "", " ")
			if err != nil {
				return err
			}
			logs = append(logs, logBytes)
			currSize += int64(len(logBytes))
		}

		threshold := s.maxSize / 2 // keep half
		start := 0
		for currSize > threshold {
			currSize -= int64(len(logs[start]))
			start++
		}

		for _, logBytes := range logs[start:] {
			_, err = temp.Write(append(logBytes, []byte("\n")...))
			if err != nil {
				return err
			}
		}

		if err = os.Remove(s.filePath); err != nil {
			return err
		}
		if err = os.Rename(s.filePath+".tmp", s.filePath); err != nil {
			return err
		}

		file.Close()
		temp.Close()
	}

	// TODO: the max time rotation

	s.writer, err = os.OpenFile(s.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (s *Native) Log(level logger.Level, msg string, args ...any) {
	s.logs <- &logger.Log{
		SessionID: s.sessid,
		Level:     level,
		Time:      time.Now().UnixMilli(),
		Message:   msg,
		Args:      args,
	}
}

func (s *Native) worker(ctx context.Context) error {
	processLog := func(log *logger.Log) error {
		log.Message = fmt.Sprintf(log.Message, log.Args...)
		log.Args = nil
		// Keep json, not text.
		// Storage is cheap, debug time ain't.
		bytes, err := json.MarshalIndent(log, "", " ")
		if err != nil {
			return err
		}

		_, err = s.writer.Write(append(bytes, []byte("\n")...))
		if err != nil {
			return err
		}
		return nil
	}

outer:
	for {
		select {
		case <-ctx.Done():
			break outer
		case log := <-s.logs:
			if err := processLog(log); err != nil {
				return err
			}
		}
	}

	for {
		select {
		case log := <-s.logs:
			if err := processLog(log); err != nil {
				return err
			}
		default:
			return nil
		}
	}
}
