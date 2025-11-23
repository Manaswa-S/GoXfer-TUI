package entry

import (
	"encoding/json"
	"goxfer/tui/consts/errs"
	"goxfer/tui/core"
	"goxfer/tui/stages/auxiliary"
	"goxfer/tui/utils"

	"github.com/bytemare/opaque"
)

type Service struct {
	core     *core.Core
	creds    *CredsManager
	settings *auxiliary.Settings
}

func NewService(core *core.Core, creds *CredsManager, settings *auxiliary.Settings) *Service {
	return &Service{
		core:     core,
		creds:    creds,
		settings: settings,
	}
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

func (s *Service) CreateBucket(pwd []byte, name string) (*CreateBucketS2Resp, *errs.Errorf) {
	if recommendation := utils.VerifyPassFormat(pwd); recommendation != "" {
		return nil, &errs.Errorf{}
	}

	// Try not to clear pwd too early as internals of 'opaque' do not copy but
	// use the same instance of pwd, and clearing pwd can cause empty passwords
	defer clear(pwd)

	cipher, err := s.core.NewBucket(pwd)
	if err != nil {
		return nil, &errs.Errorf{Error: err}
	}

	// TODO: get config from server ??
	// OPAQUE: step 1
	conf := opaque.DefaultConfiguration()
	client, err := conf.Client()
	if err != nil {
		return nil, &errs.Errorf{Error: err}
	}
	c1 := client.RegistrationInit(pwd)

	s1Req := CreateBucketS1Req{
		S1Req: utils.EncodeBase64(c1.Serialize()),
	}
	s1ReqBytes, err := json.Marshal(s1Req)
	if err != nil {
		return nil, &errs.Errorf{Error: err}
	}
	_, respBody, err := s.core.Hit(core.Routes.RegistrationInit, nil, nil,
		&core.BodyParams{ConType: core.ConType.JSON, Body: s1ReqBytes})
	if err != nil {
		return nil, &errs.Errorf{Error: err}
	}
	data := new(CreateBucketS1Resp)
	err = json.Unmarshal(respBody, data)
	if err != nil {
		return nil, &errs.Errorf{Error: err}
	}

	// OPAQUE: step 2
	response, err := client.Deserialize.RegistrationResponse(utils.DecodeBase64(data.S1Resp))
	if err != nil {
		return nil, &errs.Errorf{Error: err}
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
		return nil, &errs.Errorf{Error: err}
	}
	_, respBody, err = s.core.Hit(core.Routes.RegistrationFinal, nil, nil,
		&core.BodyParams{ConType: core.ConType.JSON, Body: s2ReqByte})
	if err != nil {
		return nil, &errs.Errorf{}
	}
	finalData := new(CreateBucketS2Resp)
	err = json.Unmarshal(respBody, finalData)
	if err != nil {
		return nil, &errs.Errorf{Error: err}
	}

	return finalData, nil
}

func (s *Service) FetchOpaqueConfigs() (*GetOpaqueConfigs, *errs.Errorf) {
	_, respBody, err := s.core.Hit(core.Routes.LoginConfigs, nil, nil, nil)
	if err != nil {
		return nil, &errs.Errorf{
			Message: "Get error.",
			Error:   err,
		}
	}

	configs := new(GetOpaqueConfigs)
	err = json.Unmarshal(respBody, configs)
	if err != nil {
		return nil, &errs.Errorf{
			Message: "JSON unmarshall error.",
			Error:   err,
		}
	}

	return configs, nil
}

func (s *Service) OpenBucket(pwd, bucKey []byte) (*OpenBucketS2Resp, *errs.Errorf) {
	if recommendation := utils.VerifyBucKeyFormat(bucKey); recommendation != "" {
		return nil, &errs.Errorf{}
	}

	configs, errf := s.FetchOpaqueConfigs()
	if errf != nil {
		return nil, errf
	}
	conf, err := opaque.DeserializeConfiguration(utils.DecodeBase64(configs.Config))
	if err != nil {
		return nil, &errs.Errorf{}
	}
	client, err := conf.Client()
	if err != nil {
		return nil, &errs.Errorf{
			Message: "Opaque client build error.",
			Error:   err,
		}
	}

	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	ke1 := client.LoginInit(pwd)

	s1Req := OpenBucketS1Req{
		BucketKey: utils.EncodeBase64(bucKey),
		KE1:       utils.EncodeBase64(ke1.Serialize()),
	}
	s1ReqBytes, err := json.Marshal(s1Req)
	if err != nil {
		return nil, &errs.Errorf{
			Message: "Marhal error.",
			Error:   err,
		}
	}
	_, respBody, err := s.core.Hit(core.Routes.LoginInit, nil, nil,
		&core.BodyParams{ConType: core.ConType.JSON, Body: s1ReqBytes})
	if err != nil {
		return nil, &errs.Errorf{
			Message: "Post error.",
			Error:   err,
		}
	}
	s1Resp := new(OpenBucketS1Resp)
	err = json.Unmarshal(respBody, s1Resp)
	if err != nil {
		return nil, &errs.Errorf{
			Message: "S1 response unmarshal error.",
			Error:   err,
		}
	}

	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	ke2, err := client.Deserialize.KE2(utils.DecodeBase64(s1Resp.KE2))
	if err != nil {
		return nil, &errs.Errorf{
			Message: "KE2 deserialize error.",
			Error:   err,
		}
	}
	ke3, _, err := client.LoginFinish(ke2, opaque.ClientLoginFinishOptions{
		ClientIdentity: []byte("test"),
		ServerIdentity: []byte("goxfer-opaque-server-id"),
	})
	if err != nil {
		return nil, &errs.Errorf{
			Message: "Login finish error.",
			Error:   err,
		}
	}

	s2Req := OpenBucketS2Req{
		KE3:     utils.EncodeBase64(ke3.Serialize()),
		LoginID: s1Resp.LoginID,
	}
	s2ReqBytes, err := json.Marshal(s2Req)
	if err != nil {
		return nil, &errs.Errorf{
			Message: "S2 request marhal error.",
			Error:   err,
		}
	}
	_, respBody, err = s.core.Hit(core.Routes.LoginFinish, nil, nil,
		&core.BodyParams{ConType: core.ConType.JSON, Body: s2ReqBytes})
	if err != nil {
		return nil, &errs.Errorf{
			Message: "Post error.",
			Error:   err,
		}
	}
	s2Resp := new(OpenBucketS2Resp)
	err = json.Unmarshal(respBody, s2Resp)
	if err != nil {
		return nil, &errs.Errorf{
			Message: "S2 response unmarshal error.",
			Error:   err,
		}
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
	clear(pwd)

	return &OpenBucketS2Resp{}, nil
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

func (s *Service) SaveCreds(creds Remember) {
	s.creds.Set(creds)
}

func (s *Service) GetSavedCreds() []Remember {
	return s.creds.Get()
}

func (s *Service) UsedCreds(key string) {
	s.creds.Used(key)
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
