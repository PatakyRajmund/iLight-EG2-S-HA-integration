package wssconnection

// JSON schemas used when sending JSON RPC messages to EG2s (For LogIn)

type LoginMessage struct {
	JsonRpc string      `json:"jsonrpc"`
	Id      string      `json:"id"`
	Method  string      `json:"method"`
	Params  LoginParams `json:"params"`
}

type LoginParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type MonitorMessage struct {
	JsonRpc string        `json:"jsonrpc"`
	Id      string        `json:"id"`
	Method  string        `json:"method"`
	Params  MonitorParams `json:"params"`
}

type MonitorParams struct {
	Predicate PredicateType `json:"predicate"`
}

type PredicateType struct {
	Type string `json:"type"`
}
