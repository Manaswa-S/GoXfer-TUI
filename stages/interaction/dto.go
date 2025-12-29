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

	EncMeta      string `json:"encMetadata"`  // base64(encrypt(MetaWrapper))
	EncMetaNonce string `json:"encMetaNonce"` // base64(nonce from encrypt(MetaWrapper))

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
	FileNonce        string `json:"fileNonce"`
	BKeySalt         string `json:"bKeySalt"`
	PKeySalt         string `json:"pKeySalt"`
	WrappedKey       string `json:"wrappedKey"`
	BWrappedKeyNonce string `json:"bWrappedKey"`
	WrappedKeyNonce  string `json:"wrappedKeyNonce"`
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
	// Encrypted file's path.
	EncPath string
	// Encrypted metadata and it's nonce.
	MetaInfo *EncMeta
	// Encrypted file info and it's nonce.
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

// >>>>>

type GetBucketDataResp struct {
	Name string `json:"name"`
}
