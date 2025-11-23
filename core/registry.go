package core

import (
	"fmt"
	"net/http"
	"net/url"
)

type RouteKey string

var Routes = struct {
	RegistrationInit  RouteKey
	RegistrationFinal RouteKey
	LoginConfigs      RouteKey
	LoginInit         RouteKey
	LoginFinish       RouteKey

	UploadInit     RouteKey
	UploadPart     RouteKey
	UploadComplete RouteKey

	FileList RouteKey

	DownloadInit   RouteKey
	DownloadData   RouteKey
	DownloadMeta   RouteKey
	DownloadDigest RouteKey

	DeleteFile RouteKey

	TestUpload   RouteKey
	TestDownload RouteKey
}{
	RegistrationInit:  "RegInit",
	RegistrationFinal: "RegFinal",
	LoginConfigs:      "LogCfg",
	LoginInit:         "LogInit",
	LoginFinish:       "LogFinish",

	UploadInit:     "UpInit",
	UploadPart:     "UpPart",
	UploadComplete: "UpComp",

	FileList: "FileList",

	DownloadInit:   "DownInit",
	DownloadData:   "DownData",
	DownloadMeta:   "DownMeta",
	DownloadDigest: "DownDigest",

	DeleteFile: "DelFile",

	TestUpload:   "TestUp",
	TestDownload: "TestDown",
}

type QueryKey string

const (
	QUploadID QueryKey = "upload_id"
	QChunkID  QueryKey = "chunk_id"
	QFileID   QueryKey = "file_id"
)

type HeaderKey string

const (
	HDownloadID HeaderKey = "X-Download-ID"
)

type QueryParams map[QueryKey]string

type HeaderParams map[HeaderKey]string

type BodyParams struct {
	ConType ContentType
	Body    []byte
}

type Route struct {
	Method string
	RPath  string
	Auth   bool
	URL    string
}

func (s *Core) register(domainStr string) error {
	domain, err := url.Parse(domainStr)
	if err != nil {
		return err
	}

	s.domain = domain
	s.routes = map[RouteKey]*Route{
		Routes.RegistrationInit: {
			Method: http.MethodPost,
			RPath:  "public/bucket/create/s1",
			Auth:   false,
		},
		Routes.RegistrationFinal: {
			Method: http.MethodPost,
			RPath:  "public/bucket/create/s2",
			Auth:   false,
		},
		Routes.LoginConfigs: {
			Method: http.MethodGet,
			RPath:  "public/bucket/open/config",
			Auth:   false,
		},
		Routes.LoginInit: {
			Method: http.MethodPost,
			RPath:  "public/bucket/open/s1",
			Auth:   false,
		},
		Routes.LoginFinish: {
			Method: http.MethodPost,
			RPath:  "public/bucket/open/s2",
			Auth:   false,
		},

		Routes.UploadInit: {
			Method: http.MethodPost,
			RPath:  "private/file/upload/init",
			Auth:   true,
		},
		Routes.UploadPart: {
			Method: http.MethodPost,
			RPath:  "private/file/upload/part",
			Auth:   true,
		},
		Routes.UploadComplete: {
			Method: http.MethodPost,
			RPath:  "private/file/upload/complete",
			Auth:   true,
		},

		Routes.TestUpload: {
			Method: http.MethodPost,
			RPath:  "public/test/upload",
			Auth:   false,
		},
		Routes.TestDownload: {
			Method: http.MethodGet,
			RPath:  "public/test/download",
			Auth:   false,
		},

		Routes.FileList: {
			Method: http.MethodGet,
			RPath:  "private/file/list",
			Auth:   true,
		},

		Routes.DownloadInit: {
			Method: http.MethodGet,
			RPath:  "private/file/download/init",
			Auth:   true,
		},
		Routes.DownloadData: {
			Method: http.MethodGet,
			RPath:  "private/file/download/data",
			Auth:   true,
		},
		Routes.DownloadMeta: {
			Method: http.MethodGet,
			RPath:  "private/file/download/meta",
			Auth:   true,
		},
		Routes.DownloadDigest: {
			Method: http.MethodGet,
			RPath:  "private/file/download/digest",
			Auth:   true,
		},

		Routes.DeleteFile: {
			Method: http.MethodDelete,
			RPath:  "private/file/delete",
			Auth:   true,
		},
	}

	for _, a := range s.routes {
		rpath, err := url.Parse(a.RPath)
		if err != nil {
			return fmt.Errorf("failed to parse: %s :%v", a.RPath, err)
		}
		a.URL = s.domain.ResolveReference(rpath).String()
	}

	return nil
}

// >>>

type ContentType string

var ConType = struct {
	Nil   ContentType
	JSON  ContentType
	Octet ContentType
}{
	Nil:   "",
	JSON:  "application/json",
	Octet: "application/octet-stream",
}

func (c ContentType) String() string {
	return string(c)
}
