package interaction

import "time"

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
type InitUploadReq struct {
	UpSpeed  float32 `json:"upSpeed"`
	FileSize int64   `json:"fileSize"`
}
type InitUploadResp struct {
	UploadID      string `json:"uploadID"`
	ChunkSize     int64  `json:"chunkSize"`
	TotalChunks   int64  `json:"totalChunks"`
	ParallelConns int    `json:"parallelConns"`
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

type CompleteUploadReq struct {
	UploadID string `json:"uploadID"` // upload Id

	EncFileInfo      string `json:"encFileInfo"`      // base64(encrypt(FileInfo))
	EncFileInfoNonce string `json:"encFileInfoNonce"` // base64(encrypt(FileInfo))

	EncMeta   string `json:"metadata"`  // base64(encrypt(MetaWrapper))
	MetaNonce string `json:"metaNonce"` // base64(nonce from encrypt(MetaWrapper))

	EncDataChecksum string `json:"dataChecksum"` // sha(Data), sha is already base64()
	EncMetaChecksum string `json:"metaChecksum"` // sha(EncMeta)
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

type EncMeta struct {
	EncMeta   string `json:"metadata"`  // base64(encrypt(MetaWrapper{}))
	MetaNonce string `json:"metaNonce"` // base64(nonce from encrypt(MetaWrapper{}))
}

type metaWrapper struct {
	RawMetaChecksum string   // hmac(Meta) // hmac is already base64()
	Meta            metaData // MetaData{}
}

type metaData struct {
	FileName string // original name of the file
	FileExt  string // original extension of the file
	FileSize int64  // original size of the file

	RawDataChecksum string // hmac(rawPath)
	HasFilePassword bool
	FileCipher      fileCipherData `json:"fileCipherData"`
}

type fileCipherData struct {
	WrappingKeySalt string `json:"wrappingKeySalt"` // base64(salt from GetWrappingKey)
	FileNonce       string `json:"fileNonce"`       // base64(nonce from encrypt(encPath))
	WrappedKey      string `json:"wrappedKey"`      // base64(WrapFileKey())
	WrappedKeyNonce string `json:"wrappedKeyNonce"` // base64(nonce from WrapFileKey())
}

// >>>

type FileInfo struct {
	FileName        string // original name of the file
	FileExt         string // original extension of the file
	FileSize        int64  // original size of the file
	HasFilePassword bool
}
type EncFileInfo struct {
	EncFileInfo   string
	FileInfoNonce string
}

// >>>

type EncDigest struct {
	EncDataChecksum string `json:"dataChecksum"` // sha(Data), sha is already base64()
	EncMetaChecksum string `json:"metaChecksum"` // sha(EncMeta)
}

type EncReturns struct {
	EncPath  string
	MetaInfo *EncMeta
	FileInfo *EncFileInfo
}

// >>>>>>>

type GetFilesListResp struct {
	Files []FilesListItem `json:"files"`
}

type FilesListItem struct {
	CreatedAt     time.Time `json:"createdAt"`
	FileUUID      string    `json:"fileUUID"`
	EncFileInfo   string    `json:"encFileInfo"`
	FileInfoNonce string    `json:"fileInfoNonce"`
}

type FileInfoExtended struct {
	CreatedAt       time.Time
	FileUUID        string
	FileName        string // original name of the file
	FileExt         string // original extension of the file
	FileSize        int64  // original size of the file
	HasFilePassword bool
}

// >>>

type DownloadInitResp struct{}

type DownloadMetaResp struct {
	EncMeta   string `json:"metadata"`  // base64(encrypt(MetaWrapper))
	MetaNonce string `json:"metaNonce"` // base64(nonce from encrypt(MetaWrapper))
}

type DownloadDigestResp struct {
	EncDataChecksum string `json:"dataChecksum"` // sha(Data), sha is already base64()
	EncMetaChecksum string `json:"metaChecksum"` // sha(EncMeta)
}
