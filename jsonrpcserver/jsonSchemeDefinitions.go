// EG2s JSONRPC schemas
package jsonrpcserver

type TransmitParams struct {
	Id   int   `json:"id"`
	Data []int `json:"data"`
}

type TransmitMessage struct {
	JsonRpc string         `json:"jsonrpc"`
	Id      string         `json:"id"`
	Method  string         `json:"method"`
	Params  TransmitParams `json:"params"`
}

type ChannelQueryResponse struct {
	JsonRpc string                     `json:"jsonrpc"`
	Method  string                     `json:"method"`
	Params  ChannelQueryResponseParams `json:"params"`
}

type ChannelQueryResponseParams struct {
	Data []int `json:"data"`
	Id   int   `json:"id"`
}

/*
type AttributeChangeAlert struct {
	Params AttributeChangeAlertParams `json:"params"`
	JsonRpc string `json:"jsonrpc"`
	Method string `json:"method"`
}

type AttributeChangeAlertParams struct{
	Value
}
*/
