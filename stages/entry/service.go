package entry

import (
	"encoding/json"
	"fmt"
	"goxfer/tui/consts/errs"
	"goxfer/tui/core"
	"goxfer/tui/logger"
	"goxfer/tui/stages/auxiliary"
	"goxfer/tui/utils"

	"github.com/bytemare/opaque"
)

type Service struct {
	logger   logger.Logger
	core     *core.Core
	creds    *CredsManager
	settings *auxiliary.Settings
}

func NewService(logger logger.Logger, core *core.Core, creds *CredsManager, settings *auxiliary.Settings) *Service {
	return &Service{
		logger:   logger,
		core:     core,
		creds:    creds,
		settings: settings,
	}
}

func (s *Service) emitErr(errf *errs.Errorf) error {
	s.logger.Log(logger.ErrorLevel, "%v: %v: %v: %v", errf.Type, errf.Error, errf.Message, errf.ReturnRaw)
	return fmt.Errorf("%s", errf.Message)
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

func (s *Service) CreateBucket(pwd []byte, name string) (*CreateBucketS2Resp, error) {
	if recommendation := utils.VerifyPassFormat(pwd); recommendation != "" {
		return nil, s.emitErr(&errs.Errorf{
			Message: recommendation,
		})
	}
	errMsg := "Failed to initiate new bucket. Try Again!"
	// Try not to clear pwd too early as internals of 'opaque' do not copy but
	// use the same instance of pwd, and clearing pwd can cause empty passwords
	defer clear(pwd)

	cipher, err := s.core.NewBucket(pwd)
	if err != nil {
		return nil, s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}

	// TODO: get config from server ??
	// OPAQUE: step 1
	conf := opaque.DefaultConfiguration()
	client, err := conf.Client()
	if err != nil {
		return nil, s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}
	c1 := client.RegistrationInit(pwd)

	s1Req := CreateBucketS1Req{
		S1Req: utils.EncodeBase64(c1.Serialize()),
	}
	s1ReqBytes, err := json.Marshal(s1Req)
	if err != nil {
		return nil, s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}
	_, respBody, err := s.core.Hit(core.Routes.RegistrationInit, nil, nil,
		&core.BodyParams{ConType: core.ConType.JSON, Body: s1ReqBytes})
	if err != nil {
		return nil, s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}
	data := new(CreateBucketS1Resp)
	err = json.Unmarshal(respBody, data)
	if err != nil {
		return nil, s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}

	errMsg = "Failed to finalize new bucket. Try Again!"
	// OPAQUE: step 2
	response, err := client.Deserialize.RegistrationResponse(utils.DecodeBase64(data.S1Resp))
	if err != nil {
		return nil, s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}
	record, _ := client.RegistrationFinalize(response, opaque.ClientRegistrationFinalizeOptions{
		ClientIdentity: []byte(name),
		ServerIdentity: utils.DecodeBase64(data.ServerID),
	})

	s2Req := &CreateBucketS2Req{
		BucName: name,
		S2Req:   utils.EncodeBase64(record.Serialize()),
		ReqID:   data.ReqID,
		Cipher:  utils.EncodeBase64(cipher),
	}
	s2ReqByte, err := json.Marshal(s2Req)
	if err != nil {
		return nil, s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}
	_, respBody, err = s.core.Hit(core.Routes.RegistrationFinal, nil, nil,
		&core.BodyParams{ConType: core.ConType.JSON, Body: s2ReqByte})
	if err != nil {
		return nil, s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}
	finalData := new(CreateBucketS2Resp)
	err = json.Unmarshal(respBody, finalData)
	if err != nil {
		return nil, s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}

	return finalData, nil
}

func (s *Service) FetchOpaqueConfigs() (*GetOpaqueConfigs, error) {
	errMsg := "Failed to get configs from server."
	_, respBody, err := s.core.Hit(core.Routes.LoginConfigs, nil, nil, nil)
	if err != nil {
		return nil, s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}

	configs := new(GetOpaqueConfigs)
	err = json.Unmarshal(respBody, configs)
	if err != nil {
		return nil, s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}

	return configs, nil
}

func (s *Service) OpenBucket(pwd, bucKey []byte) error {
	if recommendation := utils.VerifyBucKeyFormat(bucKey); recommendation != "" {
		return s.emitErr(&errs.Errorf{
			Message: recommendation,
		})
	}
	defer clear(pwd)
	errMsg := "Failed to open bucket."

	configs, err := s.FetchOpaqueConfigs()
	if err != nil {
		return s.emitErr(&errs.Errorf{
			Error:   fmt.Errorf("failed to get opaque configs for opening bucket"),
			Message: errMsg + " " + err.Error(),
		})
	}
	conf, err := opaque.DeserializeConfiguration(utils.DecodeBase64(configs.Config))
	if err != nil {
		return s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}
	client, err := conf.Client()
	if err != nil {
		return s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}

	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	ke1 := client.LoginInit(pwd)

	s1Req := OpenBucketS1Req{
		BucketKey: utils.EncodeBase64(bucKey),
		KE1:       utils.EncodeBase64(ke1.Serialize()),
	}
	s1ReqBytes, err := json.Marshal(s1Req)
	if err != nil {
		return s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}
	_, respBody, err := s.core.Hit(core.Routes.LoginInit, nil, nil,
		&core.BodyParams{ConType: core.ConType.JSON, Body: s1ReqBytes})
	if err != nil {
		return s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}
	s1Resp := new(OpenBucketS1Resp)
	err = json.Unmarshal(respBody, s1Resp)
	if err != nil {
		return s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}

	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	ke2, err := client.Deserialize.KE2(utils.DecodeBase64(s1Resp.KE2))
	if err != nil {
		return s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}
	ke3, _, err := client.LoginFinish(ke2, opaque.ClientLoginFinishOptions{
		ClientIdentity: []byte(s1Resp.ClientID),
		ServerIdentity: utils.DecodeBase64(configs.ServerID),
	})
	if err != nil {
		return s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}

	s2Req := OpenBucketS2Req{
		KE3:     utils.EncodeBase64(ke3.Serialize()),
		LoginID: s1Resp.LoginID,
	}
	s2ReqBytes, err := json.Marshal(s2Req)
	if err != nil {
		return s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}
	_, respBody, err = s.core.Hit(core.Routes.LoginFinish, nil, nil,
		&core.BodyParams{ConType: core.ConType.JSON, Body: s2ReqBytes})
	if err != nil {
		return s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}
	s2Resp := new(OpenBucketS2Resp)
	err = json.Unmarshal(respBody, s2Resp)
	if err != nil {
		return s.emitErr(&errs.Errorf{
			Error:   err,
			Message: errMsg,
		})
	}

	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	err = s.core.SetSession([]byte(s2Resp.SessionID), client.SessionKey())
	if err != nil {
		panic(err)
	}
	err = s.core.OpenBucket(bucKey, pwd, utils.DecodeBase64(s2Resp.Cipher))
	if err != nil {
		panic(err)
	}

	return nil
}

// >>>

func (s *Service) SaveCreds(creds *Remember) {
	s.creds.Set(creds)
}

func (s *Service) GetSavedCreds() []Remember {
	return s.creds.Get()
}

func (s *Service) UsedCreds(key []byte) {
	s.creds.Used(key)
}
