package entry

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
type CreateBucketS1Req struct {
	S1Req string `json:"s1Req"` // base64(reg.Serialize()), opaque step 1 data
}
type CreateBucketS1Resp struct {
	S1Resp   string `json:"s1Resp"`
	ReqID    string `json:"reqID"`
	ServerID string `json:"serverID"`
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
type CreateBucketS2Req struct {
	BucName string `json:"bucName"` // string, bucket name
	S2Req   string `json:"s2Req"`   // base64(record.Serialize()), opaque step 2 record
	ReqID   string `json:"reqID"`   // string, request ID
	Cipher  string `json:"cipher"`  // base64(bucketCipher), cipher for bucket
}
type CreateBucketS2Resp struct {
	BucketKey string `json:"bucketKey"`
	Name      string `json:"name"`
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
type OpenBucketS1Req struct {
	BucketKey string `json:"bucketKey"`
	KE1       string `json:"ke1"`
}
type OpenBucketS1Resp struct {
	KE2      string `json:"ke2"`
	ClientID string `json:"clientID"`
	LoginID  string `json:"loginID"`
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
type OpenBucketS2Req struct {
	KE3     string `json:"ke3"`
	LoginID string `json:"loginID"`
}
type OpenBucketS2Resp struct {
	SessionID  string `json:"sessionID"`
	SessionTTL int64  `json:"sessionTTL"`
	Cipher     string `json:"cipher"` // base64(bucketCipher), cipher for bucket
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
type GetOpaqueConfigs struct {
	ServerID string `json:"serverID"`
	Config   string `json:"config"`
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
