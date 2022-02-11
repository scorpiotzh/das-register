package example

import (
	"das_register_server/http_server/api_code"
	"das_register_server/http_server/handle"
	"encoding/json"
	"fmt"
	"github.com/DeAccountSystems/das-lib/common"
	"github.com/parnurzeal/gorequest"
	"github.com/scorpiotzh/toolib"
	"testing"
)

const (
	TestUrl = "https://test-register-api.da.systems/v1"
)

func TestTransactionSend(t *testing.T) {
	str := `{"sign_key":"3006b6f61c704cd33d19f6e52bf033b6","sign_list":[{"sign_type":6,"sign_msg":"0x7fc2127e46a95588f28ec3cc069645e8f46d974eee48b6830712bfd8734212d914744d0765a4fbfd8368dc57ff4cfedb82936ee8147285df7b6802ce4e0dc20801"}],"mm_json":null}`

	var req handle.ReqTransactionSend
	if err := json.Unmarshal([]byte(str), &req); err != nil {
		t.Fatal(err)
	}
	url := TestUrl + "/transaction/send"
	var data handle.RespTransactionSend
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println(data.Hash)
}

func doReq(url string, req, data interface{}) error {
	var resp api_code.ApiResp
	resp.Data = &data

	_, _, errs := gorequest.New().Post(url).SendStruct(&req).EndStruct(&resp)
	if errs != nil {
		return fmt.Errorf("%v", errs)
	}
	if resp.ErrNo != api_code.ApiCodeSuccess {
		return fmt.Errorf("%d - %s", resp.ErrNo, resp.ErrMsg)
	}
	return nil
}

func TestEditManager(t *testing.T) {
	url := TestUrl + "/account/edit/manager"

	var req handle.ReqEditManager
	req.ChainType = common.ChainTypeMixin
	req.Address = "0xe1090ce82474cbe0b196d1e62ec349ec05a61076c68d14129265370ca7e051c4"
	req.Account = "1234567871.bit"
	req.RawParam.ManagerChainType = common.ChainTypeMixin
	req.RawParam.ManagerAddress = "0x99c648a7968540a630dc665a676cf90adaeaad923685f03803abd23bc17c5b58"
	var data handle.RespEditManager

	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}

	var signReq handle.ReqSignTx
	signReq.SignInfo = data.SignInfo
	signReq.ChainId = 5
	signReq.Private = ""
	fmt.Println(toolib.JsonString(signReq))

	// curl -X POST http://127.0.0.1:8119/v1/sign/tx -d'{"chain_id":5,"private":"","sign_key":"87c906d6121d7eeea5670a247600cd4c","sign_list":[{"sign_type":5,"sign_msg":"0xad9bc80a25d3753354f085074c271477db309dff75b15f7ab4db9e192a7cf768"}],"mm_json":{"types":{"EIP712Domain":[{"name":"chainId","type":"uint256"},{"name":"name","type":"string"},{"name":"verifyingContract","type":"address"},{"name":"version","type":"string"}],"Action":[{"name":"action","type":"string"},{"name":"params","type":"string"}],"Cell":[{"name":"capacity","type":"string"},{"name":"lock","type":"string"},{"name":"type","type":"string"},{"name":"data","type":"string"},{"name":"extraData","type":"string"}],"Transaction":[{"name":"DAS_MESSAGE","type":"string"},{"name":"inputsCapacity","type":"string"},{"name":"outputsCapacity","type":"string"},{"name":"fee","type":"string"},{"name":"action","type":"Action"},{"name":"inputs","type":"Cell[]"},{"name":"outputs","type":"Cell[]"},{"name":"digest","type":"bytes32"}]},"primaryType":"Transaction","domain":{"chainId":5,"name":"da.systems","verifyingContract":"0x0000000000000000000000000000000020210722","version":"1"},"message":{"DAS_MESSAGE":"EDIT MANAGER OF ACCOUNT 0001.bit","inputsCapacity":"214.9995 CKB","outputsCapacity":"214.9994 CKB","fee":"0.0001 CKB","digest":"","action":{"action":"edit_manager","params":"0x00"},"inputs":[{"capacity":"214.9995 CKB","lock":"das-lock,0x01,0x05c9f53b1d85356b60453f867610888d89a0b667...","type":"account-cell-type,0x01,0x","data":"{ account: 0001.bit, expired_at: 1822199174 }","extraData":"{ status: 0, records_hash: 0x5376adbb69986cf8192a1ab94fe438920e2046f1b450ef9af5a8ad0902890e28 }"}],"outputs":[{"capacity":"214.9994 CKB","lock":"das-lock,0x01,0x05c9f53b1d85356b60453f867610888d89a0b667...","type":"account-cell-type,0x01,0x","data":"{ account: 0001.bit, expired_at: 1822199174 }","extraData":"{ status: 0, records_hash: 0x5376adbb69986cf8192a1ab94fe438920e2046f1b450ef9af5a8ad0902890e28 }"}]}}}'
}

func TestEditOwner(t *testing.T) {
	url := TestUrl + "/account/edit/owner"

	var req handle.ReqEditOwner
	req.ChainType = common.ChainTypeMixin
	req.Address = "0x99c648a7968540a630dc665a676cf90adaeaad923685f03803abd23bc17c5b58"
	req.Account = "1234567871.bit"
	req.RawParam.ReceiverChainType = common.ChainTypeMixin
	req.RawParam.ReceiverAddress = "0xe1090ce82474cbe0b196d1e62ec349ec05a61076c68d14129265370ca7e051c4"
	req.EvmChainId = 97

	var data handle.RespEditOwner

	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}

	var signReq handle.ReqSignTx
	signReq.SignInfo = data.SignInfo
	signReq.ChainId = 97
	signReq.Private = ""
	fmt.Println(toolib.JsonString(signReq))
	// curl -X POST http://127.0.0.1:8119/v1/sign/tx
}

func TestEditRecords(t *testing.T) {
	url := TestUrl + "/account/edit/records"

	var req handle.ReqEditRecords
	req.ChainType = common.ChainTypeMixin
	req.Address = "0x99c648a7968540a630dc665a676cf90adaeaad923685f03803abd23bc17c5b58"
	req.Account = "1234567871.bit"
	req.RawParam.Records = []handle.ReqRecord{{
		Key:   "twitter",
		Type:  "profile",
		Label: "33",
		Value: "111",
		TTL:   "300",
	}}
	req.EvmChainId = 5

	var data handle.RespEditRecords

	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
	var signReq handle.ReqSignTx
	signReq.SignInfo = data.SignInfo
	signReq.ChainId = 5
	signReq.Private = ""
	fmt.Println(toolib.JsonString(signReq))
	// curl -X POST http://127.0.0.1:8119/v1/sign/tx
}

func TestBalancePay(t *testing.T) {
	url := TestUrl + "/balance/pay"
	var req handle.ReqBalancePay
	req.EvmChainId = 5
	req.OrderId = "45e215c533894aa2d215b4cf976e6ee5"
	req.ChainType = common.ChainTypeMixin
	req.Address = "0xe1090ce82474cbe0b196d1e62ec349ec05a61076c68d14129265370ca7e051c4"

	var data handle.RespBalancePay

	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
	var signReq handle.ReqSignTx
	signReq.SignInfo = data.SignInfo
	signReq.ChainId = 5
	signReq.Private = ""
	fmt.Println(toolib.JsonString(signReq))
	// curl -X POST http://127.0.0.1:8119/v1/sign/tx
}

func TestReverseDeclare(t *testing.T) {
	url := TestUrl + "/reverse/declare"
	var req handle.ReqReverseDeclare
	req.ChainType = common.ChainTypeMixin
	req.Address = "0xe1090ce82474cbe0b196d1e62ec349ec05a61076c68d14129265370ca7e051c4"
	req.Account = "1234567871.bit"
	req.EvmChainId = 5

	var data handle.RespReverseDeclare
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
	var signReq handle.ReqSignTx
	signReq.SignInfo = data.SignInfo
	signReq.ChainId = 5
	signReq.Private = ""
	fmt.Println(toolib.JsonString(signReq))
}

func TestReverseRedeclare(t *testing.T) {
	url := TestUrl + "/reverse/redeclare"
	var req handle.ReqReverseRedeclare
	req.ChainType = common.ChainTypeMixin
	req.Address = "0xe1090ce82474cbe0b196d1e62ec349ec05a61076c68d14129265370ca7e051c4"
	req.Account = "1234567872.bit"
	req.EvmChainId = 5

	var data handle.RespReverseRedeclare
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
	var signReq handle.ReqSignTx
	signReq.SignInfo = data.SignInfo
	signReq.ChainId = 5
	signReq.Private = ""
	fmt.Println(toolib.JsonString(signReq))
}

func TestReverseRetract(t *testing.T) {
	var req handle.ReqReverseRetract
	req.ChainType = common.ChainTypeMixin
	req.Address = "0xe1090ce82474cbe0b196d1e62ec349ec05a61076c68d14129265370ca7e051c4"
	req.EvmChainId = 5
	url := TestUrl + "/reverse/retract"

	var data handle.RespReverseRetract
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
	var signReq handle.ReqSignTx
	signReq.SignInfo = data.SignInfo
	signReq.ChainId = 5
	signReq.Private = ""
	fmt.Println(toolib.JsonString(signReq))
}

func TestBalanceInfo(t *testing.T) {
	url := TestUrl + "/balance/info"
	var req handle.ReqBalanceInfo
	req.ChainType = common.ChainTypeMixin
	req.Address = "0xe1090ce82474cbe0b196d1e62ec349ec05a61076c68d14129265370ca7e051c4"

	var data handle.RespBalanceInfo
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println(toolib.JsonString(data))
}

func TestReverseLatest(t *testing.T) {
	url := TestUrl + "/reverse/latest"
	var req handle.ReqReverseLatest
	req.ChainType = common.ChainTypeMixin
	req.Address = "0xe1090ce82474cbe0b196d1e62ec349ec05a61076c68d14129265370ca7e051c4"
	var data handle.RespReverseLatest
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println(toolib.JsonString(data))
}
