package core

import (
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type TestUploadResult struct {
	UpTime int64
	UpLen  int64
}
type TestDownloadResult struct {
	DownTime int64
	DownLen  int64
}

func (c *Core) TestUpload() (*TestUploadResult, error) {
	url, ok := c.routes[Routes.TestUpload]
	if !ok {
		return nil, fmt.Errorf("invalid route")
	}
	client := &http.Client{Timeout: 0}

	upLen := int64(4 * 1024 * 1024)
	upData := make([]byte, upLen)
	rand.Read(upData)

	pr, pw := io.Pipe()
	req, err := http.NewRequest(url.Method, url.URL, pr)
	if err != nil {
		return nil, err
	}
	go func() {
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
	}()

	_, err = fmt.Fprintf(pw, "%d\n", upLen)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	_, err = pw.Write(upData)
	if err != nil {
		return nil, err
	}
	uploadTime := time.Since(start).Milliseconds()

	return &TestUploadResult{
		UpTime: uploadTime,
		UpLen:  upLen,
	}, nil
}

func (c *Core) TestDownload() (*TestDownloadResult, error) {
	url, ok := c.routes[Routes.TestDownload]
	if !ok {
		return nil, fmt.Errorf("invalid route")
	}
	client := &http.Client{Timeout: 0}

	req, err := http.NewRequest(url.Method, url.URL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	startTime, err := strconv.ParseInt(resp.Header.Get("Start-Time"), 10, 64)
	if err != nil {
		return nil, err
	}

	written, err := io.Copy(io.Discard, resp.Body)
	if err != nil && err != io.EOF {
		return nil, err
	}
	downTime := time.Now().UnixMilli() - startTime

	return &TestDownloadResult{
		DownTime: downTime,
		DownLen:  written,
	}, nil
}
