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
	s.logger.Log(logger.ErrorLevel, "%v: %v: %v: %v", errf.Type, errf.Error, errf.Message, errf.ReturnRaw)
	return fmt.Errorf("%s", errf.Message)
}

// >>>
func (s *Service) Logout() {
	s.core.DelSession()

}

// >>>
func (s *Service) GetFilesList() ([]FileInfoExtended, error) {
	errMsg := "Failed to get files list. Try Again."
	resp, respBody, err := s.core.Hit(core.Routes.FileList, nil, nil, nil)
	if err != nil {
		return nil, s.emitErr(&errs.Errorf{
			Type:    errs.ErrDependencyFailed,
			Error:   fmt.Errorf("failed to hit files list: %v", err),
			Message: errMsg,
		})
	}
	if resp.StatusCode != http.StatusOK {
		return nil, s.emitErr(&errs.Errorf{
			Type:    errs.ErrBadRequest,
			Error:   fmt.Errorf("failed to hit files list: status not ok"),
			Message: errMsg,
		})
	}

	list := new(GetFilesListResp)
	err = json.Unmarshal(respBody, list)
	if err != nil {
		return nil, s.emitErr(&errs.Errorf{
			Type:    errs.ErrBadRequest,
			Error:   fmt.Errorf("failed to unmarshal files list: %v", err),
			Message: errMsg,
		})
	}

	errMsg = "Failed to decrypt files list."
	files := make([]FileInfoExtended, 0)
	for _, file := range list.Files {
		data, err := s.core.Decrypt(utils.DecodeBase64(file.EncFileInfo), utils.DecodeBase64(file.FileInfoNonce))
		if err != nil {
			return nil, s.emitErr(&errs.Errorf{
				Type:    errs.ErrBadRequest,
				Error:   fmt.Errorf("failed to decrypt file info: %v", err),
				Message: errMsg,
			})
		}

		info := new(FileInfo)
		err = json.Unmarshal(data, info)
		if err != nil {
			return nil, s.emitErr(&errs.Errorf{
				Type:    errs.ErrBadRequest,
				Error:   fmt.Errorf("failed to unmarshal file info: %v", err),
				Message: errMsg,
			})
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

// >>>
func (s *Service) ManageUpload(pwd []byte, rawPath string, progress func(string, int64)) (err error) {
	encPath := filepath.Join(filepath.Dir(rawPath), fmt.Sprintf(".goXfer.%s.enc", filepath.Base(rawPath)))
	defer func() {
		if err = os.Remove(encPath); err != nil {
			// TODO: should not panic
			panic(fmt.Errorf("failed to remove enc file: %v", err))
		}
	}()
	// INIT
	errMsg := "Failed to upload file. Try Again!"
	plan, encReturns, err := s.initUpload(pwd, rawPath, encPath, progress)
	if err != nil {
		return s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to init upload: %v", err),
			Message: errMsg,
		})
	}
	// UPLOAD
	encFile, err := os.Open(encReturns.EncPath)
	if err != nil {
		return s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to open enc file: %v", err),
			Message: errMsg,
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
					Message: errMsg,
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
			Message: errMsg,
		})
	}
	close(successChan)

	// NOTIFY COMPLETION
	errMsg = "Failed to notify upload completion."
	progress("completing upload", 0)
	if err = s.completeUpload(plan.UploadID, encReturns); err != nil {
		return s.emitErr(&errs.Errorf{
			Type:    errs.ErrDependencyFailed,
			Error:   fmt.Errorf("failed to complete file upload: %v", err),
			Message: errMsg,
		})
	}

	return nil
}

func (s *Service) initUpload(pwd []byte, rawPath, encPath string, stage func(string, int64)) (plan *InitUploadResp, encReturns *EncReturns, err error) {
	// TEST UPLOAD
	stage("testing upload", 0)
	upload, err := s.core.TestUpload()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to test upload: %v", err)
	}

	// ENCRYPT FILE
	stage("encrypting file", 0)
	encReturns, err = s.encryptFile(pwd, rawPath, encPath)
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

		EncMeta:      encReturns.MetaInfo.EncMeta,
		EncMetaNonce: encReturns.MetaInfo.MetaNonce,

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

	return nil
}

func (s *Service) encryptFile(pwd []byte, rawPath, encPath string) (returns *EncReturns, err error) {
	// Generate fkey and encrypt file.
	fkey := s.cipher.GetCEK()
	fileNonce, err := s.cipher.EncryptFile(fkey, rawPath, encPath)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt file: %v", err)
	}

	// Define
	var wedKey, bWrappedKeyNonce, wedKeyNonce []byte
	var bKey, bKeySalt []byte
	var pKey, pKeySalt []byte

	// Irrespective of file password,
	// get a WEK using bucPass.
	bKey, bKeySalt = s.core.GetKEK()
	// Wrap fkey using bkey.
	wedKey, bWrappedKeyNonce, err = s.cipher.Wrap(bKey, fkey)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap: %v", err)
	}

	hasFilePass := false
	// If file password is given,
	if len(pwd) > 0 {
		// Get KEK using pwd
		pKey, pKeySalt = s.cipher.GetKEK(pwd)
		// And wrap wedKeyUBuc again.
		wedKey, wedKeyNonce, err = s.cipher.Wrap(pKey, wedKey)
		if err != nil {
			return nil, fmt.Errorf("failed to wrap: %v", err)
		}
		hasFilePass = true
	}

	fileCipher := fileCipherData{
		FileNonce:        utils.EncodeBase64(fileNonce),
		BKeySalt:         utils.EncodeBase64(bKeySalt),
		PKeySalt:         utils.EncodeBase64(pKeySalt),
		WrappedKey:       utils.EncodeBase64(wedKey),
		BWrappedKeyNonce: utils.EncodeBase64(bWrappedKeyNonce),
		WrappedKeyNonce:  utils.EncodeBase64(wedKeyNonce),
	}

	// >>>
	// All other crypts are to be signed using bkey,
	// cause that's the main key. File password is optional.

	// Get HMAC of raw file using bKey.
	rawDataChksm, err := s.cipher.GetHMAC(rawPath, bKey)
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
	rawMetaChksm, err := s.cipher.GetHMACBytes(metaBytes, bKey)
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

	// TODO:
	clear(fkey)
	clear(fileNonce)
	clear(bKey)
	clear(pKey)
	clear(bKeySalt)
	clear(pKeySalt)
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

// >>>
func (s *Service) ManageDownload(fileId string, pwd []byte, progress func(int64)) (rawPath string, err error) {
	errMsg := "Failed to initiate download. Try again!"
	// INIT
	respInit, _, err := s.core.Hit(core.Routes.DownloadInit, core.QueryParams{
		core.QFileID: fileId,
	}, nil, nil)
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to initiate download: %v", err),
			Message: errMsg,
		})
	}
	if respInit.StatusCode != http.StatusOK {
		return "", s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to initiate download: status not ok: %v", err),
			Message: errMsg,
		})
	}
	downId := respInit.Header.Get("X-Download-ID")
	if downId == "" {
		return "", s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to initiate download: download id not found: %v", err),
			Message: errMsg,
		})
	}

	errMsg = "Failed to download file. Try again!"

	wd, err := os.Getwd()
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to get wd: %v", err),
			Message: errMsg,
		})
	}
	encFile := fmt.Sprintf(".goXfer.%s.enc", fileId)
	encPath := filepath.Join(wd, encFile)
	defer func() {
		if err = os.Remove(encPath); err != nil {
			panic(err)
		}
	}()

	// DATA
	err = s.downloadData(fileId, downId, encPath, progress)
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to download data: %v", err),
			Message: errMsg,
		})
	}

	// META
	encMeta, err := s.downloadMeta(fileId, downId)
	defer func() { encMeta = nil }()
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to download meta: %v", err),
			Message: errMsg,
		})
	}

	// DIGEST
	digest, err := s.downloadDigest(fileId, downId)
	defer func() { digest = nil }()
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to download digest: %v", err),
			Message: errMsg,
		})
	}
	// Everything has been downloaded.
	err = s.checkDownloads(digest, encPath, encMeta)
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}

	errMsg = "Failed to decrypt file. Try again!"
	// Get decrypted and checked metadata
	meta, err := s.getMetadata(encMeta)
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}
	fileCipher := meta.FileCipher

	wedKey := utils.DecodeBase64(fileCipher.WrappedKey)
	if meta.HasFilePassword && len(pwd) > 0 {
		pKey := s.cipher.GetKEKWithSalt(pwd, utils.DecodeBase64(fileCipher.PKeySalt))

		wedKey, err = s.cipher.UnWrap(wedKey, pKey, utils.DecodeBase64(fileCipher.WrappedKeyNonce))
		if err != nil {
			return "", s.emitErr(&errs.Errorf{
				Error:   fmt.Errorf("failed to unwrap: %v", err),
				Message: errMsg,
			})
		}
	}

	bKey := s.core.GetKEKWithSalt(utils.DecodeBase64(fileCipher.BKeySalt))

	fKey, err := s.cipher.UnWrap(wedKey, bKey, utils.DecodeBase64(fileCipher.BWrappedKeyNonce))
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to unwrap: %v", err),
			Message: errMsg,
		})
	}

	rawFile := meta.FileName
	rawPath = filepath.Join(wd, rawFile)
	err = s.cipher.DecryptFile(fKey, utils.DecodeBase64(fileCipher.FileNonce), encPath, rawPath)
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to decrypt: %v", err),
			Message: errMsg,
		})
	}

	// Check raw file
	rawDataChksm, err := s.cipher.GetHMAC(rawPath, bKey)
	if err != nil {
		return "", s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to get HMAC: %v", err),
			Message: errMsg,
		})
	}
	if rawDataChksm != meta.RawDataChecksum {
		return "", s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to match checksums: %v", err),
			Message: errMsg,
		})
	}

	return rawFile, nil
}

func (s *Service) downloadData(fileId, downId, encPath string, progress func(int64)) (err error) {
	resp, err := s.core.Download(core.Routes.DownloadData,
		core.QueryParams{core.QFileID: fileId}, downId)
	if err != nil {
		return err
	}
	fullSize, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return err
	}

	out, err := os.Create(encPath)
	if err != nil {
		return err
	}

	buf := make([]byte, 64*1024)
	var downloaded int64
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			nw, err := out.Write(buf[:n])
			if err != nil {
				return err
			}
			downloaded += int64(nw)
			progress((downloaded * 100) / fullSize)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
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

func (s *Service) checkDownloads(digest *EncDigest, encPath string, encMeta *EncMeta) error {
	// Check encrypted data
	encDataChksm, err := s.cipher.GetSHA(encPath)
	if err != nil {
		return s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to get SHA: %v", err),
			Message: "Failed to verify file. Try again!",
		})
	}
	if encDataChksm != digest.EncDataChecksum {
		return s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("enc data checksums did not match : %v", err),
			Message: "Failed to verify file. Try again!",
		})
	}

	// Check encrypted metadata
	encMetaChksm, err := s.cipher.GetSHABytes(utils.DecodeBase64(encMeta.EncMeta))
	if err != nil {
		return s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to get SHA: %v", err),
			Message: "Failed to verify file. Try again!",
		})
	}
	if encMetaChksm != digest.EncMetaChecksum {
		return s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("enc meta checksums did not match : %v", err),
			Message: "Failed to verify file. Try again!",
		})
	}

	return nil
}

func (s *Service) getMetadata(encMeta *EncMeta) (*metaData, error) {
	metaWrapperBytes, err := s.core.Decrypt(utils.DecodeBase64(encMeta.EncMeta),
		utils.DecodeBase64(encMeta.MetaNonce))
	if err != nil {
		return nil, s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to decrypt meta: %v", err),
			Message: "Failed to decrypt file. Try again!",
		})
	}
	metaWrapper := new(metaWrapper)
	if err = json.Unmarshal(metaWrapperBytes, metaWrapper); err != nil {
		return nil, s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to unmarshal meta: %v", err),
			Message: "Failed to decrypt file. Try again!",
		})
	}
	meta := metaWrapper.Meta

	bKey := s.core.GetKEKWithSalt(utils.DecodeBase64(meta.FileCipher.BKeySalt))

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to marshal meta: %v", err),
			Message: "Failed to decrypt file. Try again!",
		})
	}
	rawMetaChksm, err := s.cipher.GetHMACBytes(metaBytes, bKey)
	if err != nil {
		return nil, s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to marshal meta: %v", err),
			Message: "Failed to decrypt file. Try again!",
		})
	}
	if rawMetaChksm != metaWrapper.RawMetaChecksum {
		return nil, s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to match meta checksums: %v", err),
			Message: "Failed to decrypt file. Try again!",
		})
	}
	clear(bKey)

	return &meta, nil
}

// >>>
func (s *Service) DeleteFile(fileId string) (err error) {
	errMsg := "Failed to delete file. Try Again!"
	resp, _, err := s.core.Hit(core.Routes.DeleteFile,
		core.QueryParams{core.QFileID: fileId}, nil, nil)
	if err != nil {
		return s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to delete file: %v", err),
			Message: errMsg,
		})
	}

	if resp.StatusCode != http.StatusOK {
		return s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to delete file: status not ok"),
			Message: errMsg,
		})
	}

	return nil
}
