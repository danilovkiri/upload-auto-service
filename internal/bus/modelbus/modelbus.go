// Package modelbus provides models for AMQP transfer objects.

package modelbus

type MsgValidate struct {
	UserID   string `json:"user_id" msgpack:"user_id"`
	FileName string `json:"file_name" msgpack:"file_name"`
}

type MsgProcess struct {
	UserID   string `json:"user_id" msgpack:"user_id"`
	FileName string `json:"file_name" msgpack:"file_name"`
	Barcode  string `json:"barcode" msgpack:"barcode"`
}

type Rsp struct {
	UserID   string `json:"user_id" msgpack:"user_id"`
	FileName string `json:"file_name" msgpack:"file_name"`
	RspType  string `json:"rsp_type" msgpack:"rsp_type"`
	IsReady  bool   `json:"is_ready" msgpack:"is_ready"`
}
