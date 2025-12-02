package interaction

import (
	"context"
	"encoding/json"
	"fmt"
	"goxfer/tui/cipher"
	"goxfer/tui/consts/errs"
	"goxfer/tui/core"
	"goxfer/tui/logger"
	"goxfer/tui/stages/auxiliary"
	"goxfer/tui/utils"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"golang.org/x/sync/errgroup"
)

type Service struct {
	logger   logger.Logger
	core     *core.Core
	cipher   cipher.Cipher
	settings *auxiliary.Settings
}

func NewService(logger logger.Logger, core *core.Core, cipher cipher.Cipher, settings *auxiliary.Settings) *Service {
	return &Service{
		logger:   logger,
		core:     core,
		cipher:   cipher,
		settings: settings,
	}
}

func (s *Service) emitErr(errf *errs.Errorf) error {
	s.logger.Log(logger.ErrorLevel, "%s: %s: %s: %s", errf.Type, errf.Error, errf.Message, errf.ReturnRaw)
	return fmt.Errorf("%s", errf.Message)
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

func (s *Service) Logout() {
	s.core.DelSession()

}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

func (s *Service) GetFilesList() ([]FileInfoExtended, error) {
	resp, respBody, err := s.core.Hit(core.Routes.FileList, nil, nil, nil)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != http.StatusOK {
		panic("not ok")
	}

	list := new(GetFilesListResp)
	err = json.Unmarshal(respBody, list)
	if err != nil {
		panic(err)
	}

	files := make([]FileInfoExtended, 0)
	for _, file := range list.Files {
		data, err := s.core.Decrypt(utils.DecodeBase64(file.EncFileInfo), utils.DecodeBase64(file.FileInfoNonce))
		if err != nil {
			panic(err)
		}

		info := new(FileInfo)
		err = json.Unmarshal(data, info)
		if err != nil {
			panic(err)
		}

		files = append(files, FileInfoExtended{
			CreatedAt:       file.CreatedAt,
			FileUUID:        file.FileUUID,
			FileName:        info.FileName,
			FileExt:         info.FileExt,
			FileSize:        info.FileSize,
			HasFilePassword: info.HasFilePassword,
		})
	}

	return files, nil
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

func (s *Service) ManageUpload(pwd []byte, path string, progress func(string, int64)) (err error) {
	// INIT
	plan, encReturns, err := s.initUpload(pwd, path, progress)
	if err != nil {
		return s.emitErr(&errs.Errorf{
			Type:    errs.ErrDependencyFailed,
			Error:   fmt.Errorf("failed to init upload: %v", err),
			Message: "Failed to upload. Try Again!",
		})
	}
	encFile, err := os.Open(encReturns.EncPath)
	if err != nil {
		return s.emitErr(&errs.Errorf{
			Type:    errs.ErrDependencyFailed,
			Error:   fmt.Errorf("failed to open enc file: %v", err),
			Message: "Failed to upload. Try Again!",
		})
	}
	defer encFile.Close()
	tasks := make(chan int64, plan.TotalChunks)
	errGrp, _ := errgroup.WithContext(context.Background())
	successChan := make(chan int64, 1)
	highestChunkSent := int64(-1)
	go func() {
		for chunk := range successChan {
			if chunk > highestChunkSent {
				progress("", int64((chunk*100)/plan.TotalChunks))
				highestChunkSent = chunk
			}
		}
	}()
	successChan <- 0

	for i := 0; i < plan.ParallelConns; i++ {
		errGrp.Go(func() error {
			if err := s.postPart(plan, encFile, tasks, successChan); err != nil {
				return s.emitErr(&errs.Errorf{
					Type:    errs.ErrDependencyFailed,
					Error:   fmt.Errorf("failed to post file part: %v", err),
					Message: "Failed to upload. Try Again!",
				})
			}
			return nil
		})
	}

	for i := int64(0); i < plan.TotalChunks; i++ {
		tasks <- i
	}
	close(tasks)

	if err = errGrp.Wait(); err != nil {
		return s.emitErr(&errs.Errorf{
			Type:    errs.ErrDependencyFailed,
			Error:   fmt.Errorf("failed to post file part: %v", err),
			Message: "Failed to upload. Try Again!",
		})
	}
	close(successChan)

	// NOTIFY COMPLETION
	progress("completing upload", 0)
	if err = s.completeUpload(plan.UploadID, encReturns); err != nil {
		return s.emitErr(&errs.Errorf{
			Type:    errs.ErrDependencyFailed,
			Error:   fmt.Errorf("failed to complete file upload: %v", err),
			Message: "Failed to upload. Try Again!",
		})
	}

	return nil
}

// TODO: encrypted file should be deleted irrespective of the outcome
func (s *Service) initUpload(pwd []byte, rawPath string, stage func(string, int64)) (plan *InitUploadResp, encReturns *EncReturns, err error) {
	// TEST UPLOAD
	stage("testing upload", 0)
	upload, err := s.core.TestUpload()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to test upload: %v", err)
	}

	// ENCRYPT FILE
	stage("encrypting file", 0)
	encReturns, err = s.encryptFile(pwd, rawPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encrypt file: %v", err)
	}
	clear(pwd)

	// INITIATE UPLOAD
	stage("initiating upload", 0)
	info, err := os.Stat(encReturns.EncPath)
	if err != nil {
		return nil, nil, err
	}
	req := &InitUploadReq{
		UpSpeed:  float32(upload.UpLen) / float32(upload.UpTime),
		FileSize: info.Size(),
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, nil, err
	}
	_, respBody, err := s.core.Hit(core.Routes.UploadInit, nil, nil,
		&core.BodyParams{ConType: core.ConType.JSON, Body: reqData})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init upload: %v", err)
	}
	initResp := new(InitUploadResp)
	if err = json.Unmarshal(respBody, initResp); err != nil {
		return nil, nil, err
	}

	return initResp, encReturns, nil
}

// TODO: implement the retry mechanism
func (s *Service) postPart(plan *InitUploadResp, encFile *os.File, tasks, success chan int64) error {
	data := make([]byte, plan.ChunkSize)
	for task := range tasks {
		clear(data)
		n, err := encFile.ReadAt(data, plan.ChunkSize*task)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read enc file: %v", err)
		}
		data = data[:n]

		resp, _, err := s.core.Hit(core.Routes.UploadPart, core.QueryParams{
			core.QUploadID: plan.UploadID,
			core.QChunkID:  strconv.FormatInt(task, 10),
		}, nil, &core.BodyParams{ConType: core.ConType.Octet, Body: data})
		if err != nil {
			return fmt.Errorf("failed to Hit %v: %v", core.Routes.UploadPart, err)
		}
		if resp.StatusCode != http.StatusAccepted {
			return fmt.Errorf("failed to Hit %v: not a 202 Accepted status", core.Routes.UploadPart)
		}
		success <- task
	}
	return nil
}

func (s *Service) completeUpload(uploadId string, encReturns *EncReturns) error {
	encDataChksm, err := s.cipher.GetSHA(encReturns.EncPath)
	if err != nil {
		return fmt.Errorf("failed to get SHA: %v", err)
	}

	encMetaChksm, err := s.cipher.GetSHABytes(utils.DecodeBase64(encReturns.MetaInfo.EncMeta))
	if err != nil {
		return fmt.Errorf("failed to get SHA: %v", err)
	}

	complete := CompleteUploadReq{
		UploadID: uploadId,

		EncFileInfo:      encReturns.FileInfo.EncFileInfo,
		EncFileInfoNonce: encReturns.FileInfo.FileInfoNonce,

		EncMeta:   encReturns.MetaInfo.EncMeta,
		MetaNonce: encReturns.MetaInfo.MetaNonce,

		EncDataChecksum: encDataChksm,
		EncMetaChecksum: encMetaChksm,
	}
	reqBody, err := json.Marshal(complete)
	if err != nil {
		return fmt.Errorf("failed to marshal %v req body: %v", core.Routes.UploadComplete, err)
	}

	resp, _, err := s.core.Hit(core.Routes.UploadComplete, nil, nil,
		&core.BodyParams{ConType: core.ConType.JSON, Body: reqBody})
	if err != nil {
		return fmt.Errorf("failed to Hit %v: %v", core.Routes.UploadComplete, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to Hit %v: status not 200 ok", core.Routes.UploadComplete)
	}

	if err = os.Remove(encReturns.EncPath); err != nil {
		return fmt.Errorf("failed to remove enc file: %v", err)
	}

	return nil
}

func (s *Service) encryptFile(pwd []byte, rawPath string) (returns *EncReturns, err error) {
	var fkey, fileNonce, wkey, wkeySalt, wedKey, wedKeyNonce []byte

	if fkey = s.cipher.GetCEK(); err != nil {
		return nil, fmt.Errorf("failed to get CEK: %v", err)
	}
	encPath := filepath.Join(filepath.Dir(rawPath), fmt.Sprintf(".goXfer.%s.enc", filepath.Base(rawPath)))
	if fileNonce, err = s.cipher.EncryptFile(fkey, rawPath, encPath); err != nil {
		return nil, fmt.Errorf("failed to encrypt file: %v", err)
	}

	hasFilePass := false
	if len(pwd) <= 0 {
		wkey, wkeySalt = s.core.GetKEK()
	} else {
		wkey, wkeySalt = s.cipher.GetKEK(pwd)
		hasFilePass = true
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get KEK: %v", err)
	}

	if wedKey, wedKeyNonce, err = s.cipher.Wrap(wkey, fkey); err != nil {
		return nil, fmt.Errorf("failed to wrap: %v", err)
	}

	fileCipher := fileCipherData{
		WrappingKeySalt: utils.EncodeBase64(wkeySalt),
		FileNonce:       utils.EncodeBase64(fileNonce),
		WrappedKey:      utils.EncodeBase64(wedKey),
		WrappedKeyNonce: utils.EncodeBase64(wedKeyNonce),
	}
	// >>>

	rawDataChksm, err := s.cipher.GetHMAC(rawPath, wkey)
	if err != nil {
		return nil, fmt.Errorf("failed to get HMAC: %v", err)
	}

	stats, err := os.Stat(rawPath)
	if err != nil {
		return nil, err
	}
	meta := &metaData{
		FileName:        stats.Name(),
		FileExt:         filepath.Ext(rawPath),
		FileSize:        stats.Size(),
		RawDataChecksum: rawDataChksm,
		HasFilePassword: hasFilePass,
		FileCipher:      fileCipher,
	}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	rawMetaChksm, err := s.cipher.GetHMACBytes(metaBytes, wkey)
	if err != nil {
		return nil, fmt.Errorf("failed to get HMAC: %v", err)
	}

	wrapper := metaWrapper{
		RawMetaChecksum: rawMetaChksm,
		Meta:            *meta,
	}
	wrapperBytes, err := json.Marshal(wrapper)
	if err != nil {
		return nil, err
	}
	metaEnc, metaNonce, err := s.core.Encrypt(wrapperBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt: %v", err)
	}

	info := FileInfo{
		FileName:        stats.Name(),
		FileExt:         filepath.Ext(rawPath),
		FileSize:        stats.Size(),
		HasFilePassword: hasFilePass,
	}
	infoBytes, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}
	infoEnc, infoNonce, err := s.core.Encrypt(infoBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt: %v", err)
	}

	clear(fkey)
	clear(fileNonce)
	clear(wkey)
	clear(wkeySalt)
	clear(wedKey)
	clear(wedKeyNonce)

	return &EncReturns{
		EncPath: encPath,
		MetaInfo: &EncMeta{
			EncMeta:   utils.EncodeBase64(metaEnc),
			MetaNonce: utils.EncodeBase64(metaNonce),
		},
		FileInfo: &EncFileInfo{
			EncFileInfo:   utils.EncodeBase64(infoEnc),
			FileInfoNonce: utils.EncodeBase64(infoNonce),
		},
	}, nil
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

func (s *Service) ManageDownload(fileId string, pwd []byte, progress func(int64)) (string, error) {
	// INIT
	respInit, _, err := s.core.Hit(core.Routes.DownloadInit, core.QueryParams{
		core.QFileID: fileId,
	}, nil, nil)
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrDependencyFailed,
			Error:   fmt.Errorf("failed to initiate download: %v", err),
			Message: "Failed to initiate download. Try again!",
		})
	}
	if respInit.StatusCode != http.StatusOK {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrBadRequest,
			Error:   fmt.Errorf("failed to initiate download: status not ok: %v", err),
			Message: "Failed to initiate download. Try again!",
		})
	}
	downId := respInit.Header.Get("X-Download-ID")
	if downId == "" {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrInternalServer,
			Error:   fmt.Errorf("failed to initiate download: download id not found: %v", err),
			Message: "Failed to initiate download. Try again!",
		})
	}

	// DATA
	dataPath, err := s.downloadData(fileId, downId, progress)
	defer func() {
		if err = os.Remove(dataPath); err != nil {
			panic(err)
		}
	}()
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrInternalServer,
			Error:   fmt.Errorf("failed to download data: %v", err),
			Message: "Failed to download file. Try again!",
		})
	}

	// META
	encMeta, err := s.downloadMeta(fileId, downId)
	defer func() { encMeta = nil }()
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrInternalServer,
			Error:   fmt.Errorf("failed to download meta: %v", err),
			Message: "Failed to download file. Try again!",
		})
	}

	// DIGEST
	digest, err := s.downloadDigest(fileId, downId)
	defer func() { digest = nil }()
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrInternalServer,
			Error:   fmt.Errorf("failed to download digest: %v", err),
			Message: "Failed to download file. Try again!",
		})
	}

	encDataChksm, err := s.cipher.GetSHA(dataPath)
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrInternalServer,
			Error:   fmt.Errorf("failed to get SHA: %v", err),
			Message: "Failed to verify file. Try again!",
		})
	}
	if encDataChksm != digest.EncDataChecksum {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrInternalServer,
			Error:   fmt.Errorf("enc data checksums did not match : %v", err),
			Message: "Failed to verify file. Try again!",
		})
	}

	encMetaChksm, err := s.cipher.GetSHABytes(utils.DecodeBase64(encMeta.EncMeta))
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrInternalServer,
			Error:   fmt.Errorf("failed to get SHA: %v", err),
			Message: "Failed to verify file. Try again!",
		})
	}
	if encMetaChksm != digest.EncMetaChecksum {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrInternalServer,
			Error:   fmt.Errorf("enc meta checksums did not match : %v", err),
			Message: "Failed to verify file. Try again!",
		})
	}

	metaWrapperBytes, err := s.core.Decrypt(utils.DecodeBase64(encMeta.EncMeta),
		utils.DecodeBase64(encMeta.MetaNonce))
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrDependencyFailed,
			Error:   fmt.Errorf("failed to decrypt meta: %v", err),
			Message: "Failed to decrypt file. Try again!",
		})
	}

	metaWrapper := new(metaWrapper)
	if err = json.Unmarshal(metaWrapperBytes, metaWrapper); err != nil {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrDependencyFailed,
			Error:   fmt.Errorf("failed to unmarshal meta: %v", err),
			Message: "Failed to decrypt file. Try again!",
		})
	}

	fileCipher := metaWrapper.Meta.FileCipher
	meta := metaWrapper.Meta

	var wkey []byte
	if metaWrapper.Meta.HasFilePassword {
		wkey = s.cipher.GetKEKWithSalt(pwd, utils.DecodeBase64(fileCipher.WrappingKeySalt))
	} else {
		wkey = s.core.GetKEKWithSalt(utils.DecodeBase64(fileCipher.WrappingKeySalt))
	}
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrDependencyFailed,
			Error:   fmt.Errorf("failed to get KEK: %v", err),
			Message: "Failed to decrypt file. Try again!",
		})
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrDependencyFailed,
			Error:   fmt.Errorf("failed to marshal meta: %v", err),
			Message: "Failed to decrypt file. Try again!",
		})
	}
	rawMetaChksm, err := s.cipher.GetHMACBytes(metaBytes, wkey)
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrDependencyFailed,
			Error:   fmt.Errorf("failed to marshal meta: %v", err),
			Message: "Failed to decrypt file. Try again!",
		})
	}
	if rawMetaChksm != metaWrapper.RawMetaChecksum {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrDependencyFailed,
			Error:   fmt.Errorf("failed to match meta checksums: %v", err),
			Message: "Failed to decrypt file. Try again!",
		})
	}

	fkey, err := s.cipher.UnWrap(utils.DecodeBase64(fileCipher.WrappedKey), wkey, utils.DecodeBase64(fileCipher.WrappedKeyNonce))
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrDependencyFailed,
			Error:   fmt.Errorf("failed to unwrap: %v", err),
			Message: "Failed to decrypt file. Try again!",
		})
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrDependencyFailed,
			Error:   fmt.Errorf("failed to get wd: %v", err),
			Message: "Failed to decrypt file. Try again!",
		})
	}

	fileName := metaWrapper.Meta.FileName
	savePath := filepath.Join(wd, fileName)
	err = s.cipher.DecryptFile(fkey, utils.DecodeBase64(fileCipher.FileNonce), dataPath, savePath)
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrDependencyFailed,
			Error:   fmt.Errorf("failed to decrypt: %v", err),
			Message: "Failed to decrypt file. Try again!",
		})
	}

	rawDataChksm, err := s.cipher.GetHMAC(savePath, wkey)
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrDependencyFailed,
			Error:   fmt.Errorf("failed to get HMAC: %v", err),
			Message: "Failed to decrypt file. Try again!",
		})
	}
	if rawDataChksm != meta.RawDataChecksum {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrDependencyFailed,
			Error:   fmt.Errorf("failed to match checksums: %v", err),
			Message: "Failed to decrypt file. Try again!",
		})
	}

	stat, err := os.Stat(savePath)
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Type:    errs.ErrDependencyFailed,
			Error:   fmt.Errorf("failed to get stats: %v", err),
			Message: "Failed to decrypt file. Try again!",
		})
	}

	return stat.Name(), nil
}

func (s *Service) downloadData(fileId, downId string, progress func(int64)) (outPath string, err error) {
	resp, err := s.core.Download(core.Routes.DownloadData,
		core.QueryParams{core.QFileID: fileId}, downId)
	if err != nil {
		return "", err
	}
	fullSize, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return "", err
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	outFile := fmt.Sprintf(".goXfer.%s.enc", fileId)
	outPath = filepath.Join(wd, outFile)
	out, err := os.Create(outPath)
	if err != nil {
		return "", err
	}

	buf := make([]byte, 64*1024)
	var downloaded int64
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			nw, err := out.Write(buf[:n])
			if err != nil {
				return "", err
			}
			downloaded += int64(nw)
			progress((downloaded * 100) / fullSize)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	return outPath, nil
}

func (s *Service) downloadMeta(fileId, downId string) (*EncMeta, error) {
	_, respBody, err := s.core.Hit(core.Routes.DownloadMeta,
		core.QueryParams{core.QFileID: fileId},
		core.HeaderParams{core.HDownloadID: downId}, nil)
	if err != nil {
		return nil, err
	}

	meta := new(DownloadMetaResp)
	err = json.Unmarshal(respBody, meta)
	if err != nil {
		return nil, err
	}

	return &EncMeta{
		EncMeta:   meta.EncMeta,
		MetaNonce: meta.MetaNonce,
	}, nil
}

func (s *Service) downloadDigest(fileId, downId string) (*EncDigest, error) {
	_, respBody, err := s.core.Hit(core.Routes.DownloadDigest,
		core.QueryParams{core.QFileID: fileId},
		core.HeaderParams{core.HDownloadID: downId}, nil)
	if err != nil {
		return nil, err
	}

	digest := new(DownloadDigestResp)
	err = json.Unmarshal(respBody, digest)
	if err != nil {
		return nil, err
	}

	return &EncDigest{
		EncDataChecksum: digest.EncDataChecksum,
		EncMetaChecksum: digest.EncMetaChecksum,
	}, nil
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

func (s *Service) DeleteFile(fileId string) (err error) {

	resp, _, err := s.core.Hit(core.Routes.DeleteFile,
		core.QueryParams{core.QFileID: fileId}, nil, nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		panic("status not ok")
	}

	return nil
}
