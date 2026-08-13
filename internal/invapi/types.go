package invapi

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// LoginResponse covers both public auth/login shapes:
//
//	top-level:  {"token":"...","invapi":"...","servers":[1]}
//	nested:     {"result":{"token":"...","servers":["12345"],"invapi":"invapi.hostkey.ru"}}
type LoginResponse struct {
	Token       string          `json:"token"`
	Role        string          `json:"role"`
	TokenExpire int64           `json:"token_expire"`
	Invapi      string          `json:"invapi"`
	Servers     json.RawMessage `json:"servers"`
	Result      *LoginResult    `json:"result"`
}

type LoginResult struct {
	Token       string          `json:"token"`
	Role        string          `json:"role"`
	TokenExpire int64           `json:"token_expire"`
	Invapi      string          `json:"invapi"`
	Servers     json.RawMessage `json:"servers"`
}

func (r LoginResponse) Normalized() (token, invapi, role string, expire int64) {
	token = r.Token
	invapi = r.Invapi
	role = r.Role
	expire = r.TokenExpire
	if r.Result != nil {
		if token == "" {
			token = r.Result.Token
		}
		if invapi == "" {
			invapi = r.Result.Invapi
		}
		if role == "" {
			role = r.Result.Role
		}
		if expire == 0 {
			expire = r.Result.TokenExpire
		}
	}
	return token, invapi, role, expire
}

type OrderInstanceRequest struct {
	Preset            string
	LocationName      string
	OSID              int
	SoftID            int
	TrafficPlan       int
	RootPass          string
	Hostname          string
	SSHKey            string
	PostInstallScript string
	DeployPeriod      string
	DeployNotify      *bool // nil = omit (InvAPI default)
	OwnOS             bool
	RootSize          int
	IPv4Amount        int
	VLAN              int
	PrivateVLAN       int
	CustomDomain      string
	OSTemplate        string
	DeployOptions     string
	ServerID          int               // non-zero => reinstall existing server
	Extra             map[string]string // escape hatch for any InvAPI form field
}

type OrderInstanceResponse struct {
	Result       string `json:"result"`
	Action       string `json:"action"`
	Callback     string `json:"callback"`
	DeployStatus string `json:"deploy_status"`
	ID           int    `json:"id"`
	Invoice      int    `json:"invoice"`
	Status       string `json:"status"`
	OSName       string `json:"os_name"`
	SoftName     string `json:"soft_name"`
	RawBody      string `json:"-"`
}

type CallbackCheckResponse struct {
	Result  string          `json:"result"`
	Action  string          `json:"action"`
	Scope   json.RawMessage `json:"scope"`
	Context json.RawMessage `json:"context"`
	Debug   string          `json:"debug"`
	Key     string          `json:"key"`
}

type CallbackContext struct {
	Action    string `json:"action"`
	ID        string `json:"id"`
	Location  string `json:"location"`
	IP        string `json:"ip"`
	Reinstall int    `json:"reinstall"`
}

type ServerShowResponse struct {
	Result     string          `json:"result"`
	ServerData json.RawMessage `json:"server_data"`
	Hardware   json.RawMessage `json:"hardware"`
	IP         []ServerIP      `json:"IP"`
	Tags       json.RawMessage `json:"tags"`
}

type ServerIP struct {
	IP     string `json:"ip"`
	MainIP int    `json:"main_ip"`
}

type ServerListResponse struct {
	Result  string          `json:"result"`
	Servers json.RawMessage `json:"servers"`
}

type ServerSummary struct {
	ID       int    `json:"id"`
	Status   string `json:"status"`
	Hostname string `json:"hostname"`
	Location string `json:"location"`
}

func (r *ServerListResponse) IDs() ([]int, error) {
	if len(r.Servers) == 0 || string(r.Servers) == "null" {
		return nil, nil
	}
	var asInts []int
	if err := json.Unmarshal(r.Servers, &asInts); err == nil {
		return asInts, nil
	}
	var asObjs []ServerSummary
	if err := json.Unmarshal(r.Servers, &asObjs); err == nil {
		ids := make([]int, 0, len(asObjs))
		for _, s := range asObjs {
			ids = append(ids, s.ID)
		}
		return ids, nil
	}
	var asStrs []string
	if err := json.Unmarshal(r.Servers, &asStrs); err == nil {
		ids := make([]int, 0, len(asStrs))
		for _, s := range asStrs {
			n, err := strconv.Atoi(s)
			if err != nil {
				return nil, err
			}
			ids = append(ids, n)
		}
		return ids, nil
	}
	return nil, fmt.Errorf("unsupported servers payload: %s", truncate([]byte(r.Servers), 256))
}

type UpdateServersResponse struct {
	Result         string          `json:"result"`
	Servers        json.RawMessage `json:"servers"`
	BillingServers []BillingServer `json:"billing_servers"`
	DeployKeysRaw  json.RawMessage `json:"deploy_keys"`
}

// DeployKeysMap normalizes deploy_keys which InvAPI may return as object map or empty array.
func (r *UpdateServersResponse) DeployKeysMap() map[string]string {
	out := map[string]string{}
	if r == nil || len(r.DeployKeysRaw) == 0 || string(r.DeployKeysRaw) == "null" {
		return out
	}
	if err := json.Unmarshal(r.DeployKeysRaw, &out); err == nil {
		return out
	}
	// empty array [] or unexpected shape
	var arr []any
	if err := json.Unmarshal(r.DeployKeysRaw, &arr); err == nil {
		return out
	}
	return out
}

type BillingServer struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Config string `json:"config"`
}

type PresetListResponse struct {
	Result  string   `json:"result"`
	Presets []Preset `json:"presets"`
}

type Preset struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Location    string `json:"location"`
	Description string `json:"description"`
	Locations   string `json:"locations"`
}
