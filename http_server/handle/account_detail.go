package handle

import (
	"bytes"
	"das_register_server/config"
	"time"

	//"das_register_server/http_server/api_code"
	"das_register_server/tables"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/minio/blake2b-simd"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"net/http"
	"strings"
)

type ReqAccountDetail struct {
	Account string `json:"account"`
}

type RespAccountDetail struct {
	Account              string                  `json:"account"`
	Owner                string                  `json:"owner"`
	OwnerChainType       common.ChainType        `json:"owner_chain_type"`
	Manager              string                  `json:"manager"`
	ManagerChainType     common.ChainType        `json:"manager_chain_type"`
	RegisteredAt         int64                   `json:"registered_at"`
	ExpiredAt            int64                   `json:"expired_at"`
	Status               tables.SearchStatus     `json:"status"`
	AccountPrice         decimal.Decimal         `json:"account_price"`
	BaseAmount           decimal.Decimal         `json:"base_amount"`
	ConfirmProposalHash  string                  `json:"confirm_proposal_hash"`
	EnableSubAccount     tables.EnableSubAccount `json:"enable_sub_account"`
	RenewSubAccountPrice uint64                  `json:"renew_sub_account_price"`
	Nonce                uint64                  `json:"nonce"`
	CustomScript         string                  `json:"custom_script"`
	PremiumPercentage    decimal.Decimal         `json:"premium_percentage"`
	PremiumBase          decimal.Decimal         `json:"premium_base"`
	ReRegisterTime       uint64                  `json:"re_register_time"`
}

func (h *HttpHandle) RpcAccountDetail(p json.RawMessage, apiResp *api_code.ApiResp) {
	var req []ReqAccountDetail
	err := json.Unmarshal(p, &req)
	if err != nil {
		log.Error("json.Unmarshal err:", err.Error())
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return
	} else if len(req) == 0 {
		log.Error("len(req) is 0")
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return
	}

	if err = h.doAccountDetail(&req[0], apiResp); err != nil {
		log.Error("doAccountDetail err:", err.Error())
	}
}

func (h *HttpHandle) AccountDetail(ctx *gin.Context) {
	var (
		funcName = "AccountDetail"
		clientIp = GetClientIp(ctx)
		req      ReqAccountDetail
		apiResp  api_code.ApiResp
		err      error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, ctx)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, toolib.JsonString(req), ctx)

	if err = h.doAccountDetail(&req, &apiResp); err != nil {
		log.Error("doAccountDetail err:", err.Error(), funcName, clientIp, ctx)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) getAccountPrice(accLen uint8, args, account string, isRenew bool) (baseAmount, accountPrice decimal.Decimal, err error) {
	builder, err := h.dasCore.ConfigCellDataBuilderByTypeArgsList(common.ConfigCellTypeArgsPrice, common.ConfigCellTypeArgsAccount)
	if err != nil {
		err = fmt.Errorf("ConfigCellDataBuilderByTypeArgsList err: %s", err.Error())
		return
	}
	//accLen := common.GetAccountLength(account)
	//if accLen == 0 {
	//	accLen = common.GetAccountLength(account)
	//}
	if accLen == 0 {
		err = fmt.Errorf("accLen is 0")
		return
	}
	newPrice, renewPrice, err := builder.AccountPrice(accLen)
	if err != nil {
		err = fmt.Errorf("AccountPrice err: %s", err.Error())
		return
	}

	quoteCell, err := h.dasCore.GetQuoteCell()
	if err != nil {
		err = fmt.Errorf("GetQuoteCell err: %s", err.Error())
		return
	}
	quote := quoteCell.Quote()

	if args == "" {
		args = "0x03"
	}
	basicCapacity, err := builder.BasicCapacityFromOwnerDasAlgorithmId(args)
	if err != nil {
		err = fmt.Errorf("BasicCapacity err: %s", err.Error())
		return
	}
	preparedFeeCapacity, err := builder.PreparedFeeCapacity()
	if err != nil {
		err = fmt.Errorf("PreparedFeeCapacity err: %s", err.Error())
		return
	}
	log.Info("BasicCapacity:", basicCapacity, "PreparedFeeCapacity:", preparedFeeCapacity, "Quote:", quote, "Price:", newPrice, renewPrice)

	basicCapacity = basicCapacity/common.OneCkb + uint64(len([]byte(account))) + preparedFeeCapacity/common.OneCkb
	baseAmount, _ = decimal.NewFromString(fmt.Sprintf("%d", basicCapacity))
	decQuote, _ := decimal.NewFromString(fmt.Sprintf("%d", quote))
	decUsdRateBase := decimal.NewFromInt(common.UsdRateBase)
	baseAmount = baseAmount.Mul(decQuote).DivRound(decUsdRateBase, 6)

	if isRenew {
		accountPrice, _ = decimal.NewFromString(fmt.Sprintf("%d", renewPrice))
		accountPrice = accountPrice.DivRound(decUsdRateBase, 2)
	} else {
		accountPrice, _ = decimal.NewFromString(fmt.Sprintf("%d", newPrice))
		accountPrice = accountPrice.DivRound(decUsdRateBase, 2)
	}
	return
}

func (h *HttpHandle) checkDutchAuction(expiredAt uint64) (status tables.SearchStatus, reRegisterTime uint64, err error) {
	nowTime := uint64(time.Now().Unix())
	//27 days : watting for bid
	//h.dasCore.ConfigCellDataBuilderByTypeArgsList()
	builderConfigCell, err := h.dasCore.ConfigCellDataBuilderByTypeArgs(common.ConfigCellTypeArgsAccount)
	if err != nil {
		err = fmt.Errorf("ConfigCellDataBuilderByTypeArgs err: %s", err.Error())
		return
	}

	gracePeriodTime, err := builderConfigCell.ExpirationGracePeriod()
	if err != nil {
		err = fmt.Errorf("ExpirationGracePeriod err: %s", err.Error())
		return
	}

	auctionPeriodTime, err := builderConfigCell.ExpirationAuctionPeriod()
	if err != nil {
		err = fmt.Errorf("ExpirationAuctionPeriod err: %s", err.Error())
		return
	}

	deliverPeriodTime, err := builderConfigCell.ExpirationDeliverPeriod()
	if err != nil {
		err = fmt.Errorf("ExpirationDeliverPeriod err: %s", err.Error())
		return
	}
	// nowTime-117*24*3600 < expiredAt && expiredAt < nowTime-90*24*3600
	if nowTime-uint64(gracePeriodTime)-uint64(auctionPeriodTime) < expiredAt && expiredAt < nowTime-uint64(gracePeriodTime) {
		status = tables.SearchStatusOnDutchAuction
		return
	}
	//3 days : can`t bid or register or renew, watting for recycle by keeper
	//nowTime-120*24*3600 < expiredAt && expiredAt < nowTime-117*24*3600
	if nowTime-uint64(gracePeriodTime)-uint64(auctionPeriodTime)-uint64(deliverPeriodTime) < expiredAt && expiredAt < nowTime-uint64(gracePeriodTime)-uint64(auctionPeriodTime) {
		status = tables.SearchStatusAuctionRecycling
		reRegisterTime = expiredAt + uint64(gracePeriodTime+auctionPeriodTime+deliverPeriodTime)
		return
	}
	return
}

func (h *HttpHandle) doAccountDetail(req *ReqAccountDetail, apiResp *api_code.ApiResp) error {
	var resp RespAccountDetail
	resp.Account = req.Account
	resp.Status = tables.SearchStatusRegisterAble
	resp.PremiumPercentage = config.Cfg.Stripe.PremiumPercentage
	resp.PremiumBase = config.Cfg.Stripe.PremiumBase

	// acc
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	acc, err := h.dbDao.GetAccountInfoByAccountId(accountId)
	if err != nil && err != gorm.ErrRecordNotFound {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "search account err")
		return fmt.Errorf("SearchAccount err: %s", err.Error())
	}

	// check sub account
	count := strings.Count(req.Account, ".")
	if count == 1 && acc.Id > 0 {
		//now < expired_at + 90 + 27 => expired_at > now-90-27
		//expired_at+90 < now => expired_at < now - 90
		//now > expired_at+90+27
		//now < expired_at+90+30
		if status, reRegisterTime, err := h.checkDutchAuction(acc.ExpiredAt); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "checkDutchAuction err")
			return fmt.Errorf("checkDutchAuction err: %s", err.Error())
		} else if status != 0 {
			resp.Status = status
			resp.ReRegisterTime = reRegisterTime
			apiResp.ApiRespOK(resp)
			return nil
		}

		accOutpoint := common.String2OutPointStruct(acc.Outpoint)
		accTx, er := h.dasCore.Client().GetTransaction(h.ctx, accOutpoint.TxHash)
		if er != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, er.Error())
			return fmt.Errorf("GetTransaction err: %s", er.Error())
		}
		mapAcc, er := witness.AccountIdCellDataBuilderFromTx(accTx.Transaction, common.DataTypeNew)
		if er != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, er.Error())
			return fmt.Errorf("AccountIdCellDataBuilderFromTx err: %s", er.Error())
		}
		accBuilder, ok := mapAcc[acc.AccountId]
		if !ok {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "mapAcc is nil")
			return fmt.Errorf("AccountCellDataBuilderMapFromTx mapAcc is nil")
		}

		// price
		var err error
		resp.BaseAmount, resp.AccountPrice, err = h.getAccountPrice(uint8(accBuilder.AccountChars.Len()), "", req.Account, true)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "get account price err")
			return fmt.Errorf("getAccountPrice err: %s", err.Error())
		}
	}
	if acc.Id > 0 {
		resp.Status = acc.FormatAccountStatus()
		resp.ExpiredAt = int64(acc.ExpiredAt) * 1e3
		resp.RegisteredAt = int64(acc.RegisteredAt) * 1e3
		resp.OwnerChainType = acc.OwnerChainType

		ownerNormal, err := h.dasCore.Daf().HexToNormal(core.DasAddressHex{
			DasAlgorithmId: acc.OwnerAlgorithmId,
			AddressHex:     acc.Owner,
			IsMulti:        false,
			ChainType:      acc.OwnerChainType,
		})
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "owner address HexToNormal err")
			return fmt.Errorf("HexToNormal err: %s", err.Error())
		}

		resp.Owner = ownerNormal.AddressNormal
		resp.ManagerChainType = acc.ManagerChainType

		managerNormal, err := h.dasCore.Daf().HexToNormal(core.DasAddressHex{
			DasAlgorithmId: acc.ManagerAlgorithmId,
			AddressHex:     acc.Manager,
			IsMulti:        false,
			ChainType:      acc.ManagerChainType,
		})
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "manager address HexToNormal err")
			return fmt.Errorf("HexToNormal err: %s", err.Error())
		}
		resp.Manager = managerNormal.AddressNormal
		resp.ConfirmProposalHash = acc.ConfirmProposalHash
		resp.EnableSubAccount = acc.EnableSubAccount
		resp.RenewSubAccountPrice = acc.RenewSubAccountPrice
		resp.Nonce = acc.Nonce

		// check custom-script
		if acc.EnableSubAccount == tables.AccountEnableStatusOn {
			subAccLiveCell, err := h.dasCore.GetSubAccountCell(acc.AccountId)
			if err != nil {
				apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
				return nil
			}
			detailSub := witness.ConvertSubAccountCellOutputData(subAccLiveCell.OutputData)
			defaultCS := make([]byte, 33)
			if len(detailSub.CustomScriptArgs) > 0 && bytes.Compare(defaultCS, detailSub.CustomScriptArgs) != 0 {
				resp.CustomScript = common.Bytes2Hex(detailSub.CustomScriptArgs)
			}
		}

		apiResp.ApiRespOK(resp)
		return nil
	}

	if count == 1 {
		// reserve account
		accountName := strings.ToLower(strings.TrimSuffix(req.Account, common.DasAccountSuffix))
		accountName = common.Bytes2Hex(common.Blake2b([]byte(accountName))[:20])

		if _, ok := h.mapReservedAccounts[accountName]; ok {
			resp.Status = tables.SearchStatusReservedAccount
			apiResp.ApiRespOK(resp)
			return nil
		}

		// unavailable account
		if _, ok := h.mapUnAvailableAccounts[accountName]; ok {
			resp.Status = tables.SearchStatusUnAvailableAccount
			apiResp.ApiRespOK(resp)
			return nil
		}
	}

	// account not exist
	apiResp.ApiRespErr(api_code.ApiCodeAccountNotExist, fmt.Sprintf("account [%s] not exist", req.Account))
	return nil
}

func Blake256AndFourBytesBigEndian(data []byte) (uint32, error) {
	bys, err := Blake256(data)
	if err != nil {
		return 0, err
	}
	bytesBuffer := bytes.NewBuffer(bys[0:4])
	var res uint32
	if err = binary.Read(bytesBuffer, binary.BigEndian, &res); err != nil {
		return 0, err
	}
	return res, nil
}

func Blake256(data []byte) ([]byte, error) {
	tmpConfig := &blake2b.Config{
		Size:   32,
		Person: []byte("2021-07-22 12:00"),
	}
	hash, err := blake2b.New(tmpConfig)
	if err != nil {
		return nil, err
	}
	hash.Write(data)
	return hash.Sum(nil), nil
}
