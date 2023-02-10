package handle

import (
	"das_register_server/cache"
	"das_register_server/config"
	"das_register_server/http_server/api_code"
	"das_register_server/internal"
	"das_register_server/notify"
	"das_register_server/tables"
	"das_register_server/timer"
	"encoding/json"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/gin-gonic/gin"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type ReqOrderRegister struct {
	ReqAccountSearch
	ReqOrderRegisterBase

	PayChainType  common.ChainType  `json:"pay_chain_type"`
	PayAddress    string            `json:"pay_address"`
	PayTokenId    tables.PayTokenId `json:"pay_token_id"`
	PayType       tables.PayType    `json:"pay_type"`
	CoinType      string            `json:"coin_type"`
	CrossCoinType string            `json:"cross_coin_type"`
	GiftCard      string            `json:"gift_card"`
}

type ReqCheckCoupon struct {
	Coupon string `json:"coupon"`
}
type RespCheckCoupon struct {
	CouponType tables.CouponType `json:"type"`
}
type ReqOrderRegisterBase struct {
	RegisterYears  int    `json:"register_years"`
	InviterAccount string `json:"inviter_account"`
	ChannelAccount string `json:"channel_account"`
}

type RespOrderRegister struct {
	OrderId        string            `json:"order_id"`
	TokenId        tables.PayTokenId `json:"token_id"`
	ReceiptAddress string            `json:"receipt_address"`
	Amount         decimal.Decimal   `json:"amount"`
	CodeUrl        string            `json:"code_url"`
	PayType        tables.PayType    `json:"pay_type"`
}

type AccountAttr struct {
	Length uint8           `json:"length"`
	Amount decimal.Decimal `json:"amount"`
}

func (h *HttpHandle) RpcCheckCouponr(p json.RawMessage, apiResp *api_code.ApiResp) {
	var req []ReqCheckCoupon
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

	if err = h.doCheckCoupon(&req[0], apiResp); err != nil {
		log.Error("doOrderRegister err:", err.Error())
	}
}
func (h *HttpHandle) CheckCoupon(ctx *gin.Context) {
	var (
		funcName = "OrderRegister"
		clientIp = GetClientIp(ctx)
		req      ReqCheckCoupon
		apiResp  api_code.ApiResp
		err      error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, toolib.JsonString(req))

	if err = h.doCheckCoupon(&req, &apiResp); err != nil {
		log.Error("doOrderRegister err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doCheckCoupon(req *ReqCheckCoupon, apiResp *api_code.ApiResp) error {
	var resp RespCheckCoupon
	if req.Coupon == "" {
		apiResp.ApiRespErr(api_code.ApiCodeCouponInvalid, "params invalid")
		return nil
	}
	res := h.checkCoupon(req.Coupon)

	if res == nil {
		apiResp.ApiRespErr(api_code.ApiCodeCouponInvalid, "gift card not found")
		return nil
	}
	resp.CouponType = res.CouponType
	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) RpcOrderRegister(p json.RawMessage, apiResp *api_code.ApiResp) {
	var req []ReqOrderRegister
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

	if err = h.doOrderRegister(&req[0], apiResp); err != nil {
		log.Error("doOrderRegister err:", err.Error())
	}
}

func (h *HttpHandle) OrderRegister(ctx *gin.Context) {
	var (
		funcName = "OrderRegister"
		clientIp = GetClientIp(ctx)
		req      ReqOrderRegister
		apiResp  api_code.ApiResp
		err      error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, toolib.JsonString(req))

	if err = h.doOrderRegister(&req, &apiResp); err != nil {
		log.Error("doOrderRegister err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doOrderRegister(req *ReqOrderRegister, apiResp *api_code.ApiResp) error {
	var resp RespOrderRegister

	if req.Address == "" || req.Account == "" {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return nil
	}
	if yes := req.PayTokenId.IsTokenIdCkbInternal(); yes {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("pay token id [%s] invalid", req.PayTokenId))
		return nil
	}

	addressHex, err := h.dasCore.Daf().NormalToHex(core.DasAddressNormal{
		ChainType:     req.ChainType,
		AddressNormal: req.Address,
		Is712:         true,
	})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "address NormalToHex err")
		return fmt.Errorf("NormalToHex err: %s", err.Error())
	}
	req.ChainType, req.Address = addressHex.ChainType, addressHex.AddressHex

	if !checkChainType(req.ChainType) {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("chain type [%d] invalid", req.ChainType))
		return nil
	}

	if err := h.checkSystemUpgrade(apiResp); err != nil {
		return fmt.Errorf("checkSystemUpgrade err: %s", err.Error())
	}

	if ok := internal.IsLatestBlockNumber(config.Cfg.Server.ParserUrl); !ok {
		apiResp.ApiRespErr(api_code.ApiCodeSyncBlockNumber, "sync block number")
		return fmt.Errorf("sync block number")
	}

	if err := h.rc.RegisterLimitLockWithRedis(req.ChainType, req.Address, "register", req.Account, time.Second*10); err != nil {
		if err == cache.ErrDistributedLockPreemption {
			apiResp.ApiRespErr(api_code.ApiCodeOperationFrequent, "the operation is too frequent")
			return nil
		}
	}

	// check un pay
	maxUnPayCount := int64(200)
	if config.Cfg.Server.Net != common.DasNetTypeMainNet {
		maxUnPayCount = 200
	}
	if unPayCount, err := h.dbDao.GetUnPayOrderCount(req.ChainType, req.Address); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to check order count")
		return nil
	} else if unPayCount > maxUnPayCount {
		log.Info("GetUnPayOrderCount:", req.ChainType, req.Address, unPayCount)
		apiResp.ApiRespErr(api_code.ApiCodeOperationFrequent, "the operation is too frequent")
		return nil
	}

	//if exi := h.rc.RegisterLimitExist(req.ChainType, req.Address, req.Account, "1"); exi {
	//	apiResp.ApiRespErr(api_code.ApiCodeOperationFrequent, "the operation is too frequent")
	//	return fmt.Errorf("AccountActionLimitExist: %d %s %s", req.ChainType, req.Address, req.Account)
	//}

	// order check
	if err := h.checkOrderInfo(req.CoinType, req.CrossCoinType, &req.ReqOrderRegisterBase, apiResp); err != nil {
		return fmt.Errorf("checkOrderInfo err: %s", err.Error())
	}
	if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// account check
	h.checkAccountCharSet(&req.ReqAccountSearch, apiResp)
	if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}
	// base check
	_, status, _, _ := h.checkAccountBase(&req.ReqAccountSearch, apiResp)
	if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}
	if status != tables.SearchStatusRegisterAble {
		switch status {
		case tables.SearchStatusUnAvailableAccount:
			apiResp.ApiRespErr(api_code.ApiCodeUnAvailableAccount, "unavailable account")
		case tables.SearchStatusReservedAccount:
			apiResp.ApiRespErr(api_code.ApiCodeReservedAccount, "reserved account")
		case tables.SearchStatusRegisterNotOpen:
			apiResp.ApiRespErr(api_code.ApiCodeNotOpenForRegistration, "registration is not open")
		default:
			apiResp.ApiRespErr(api_code.ApiCodeAccountAlreadyRegister, "account already register")
		}
		return nil
	}
	// self order
	status, _ = h.checkAddressOrder(&req.ReqAccountSearch, apiResp, false)
	if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	} else if status != tables.SearchStatusRegisterAble {
		apiResp.ApiRespErr(api_code.ApiCodeAccountAlreadyRegister, "account registering")
		return nil
	}
	// registering check
	status = h.checkOtherAddressOrder(&req.ReqAccountSearch, apiResp)
	if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	} else if status >= tables.SearchStatusRegistering {
		apiResp.ApiRespErr(api_code.ApiCodeAccountAlreadyRegister, "account registering")
		return nil
	}

	// create order
	h.doRegisterOrder(req, apiResp, &resp)
	if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}
	// cache
	// _ = h.rc.SetRegisterLimit(req.ChainType, req.Address, req.Account, "1", time.Second*30)
	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) checkOrderInfo(coinType, crossCoinType string, req *ReqOrderRegisterBase, apiResp *api_code.ApiResp) error {
	if req.RegisterYears <= 0 || req.RegisterYears > config.Cfg.Das.MaxRegisterYears {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("register years[%d] invalid", req.RegisterYears))
		return nil
	}
	if req.InviterAccount != "" {
		accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.InviterAccount))
		acc, err := h.dbDao.GetAccountInfoByAccountId(accountId)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "search inviter account fail")
			return fmt.Errorf("GetAccountInfoByAccountId err: %s", err.Error())
		} else if acc.Id == 0 {
			apiResp.ApiRespErr(api_code.ApiCodeInviterAccountNotExist, "inviter account not exist")
			return nil
		} else if acc.Status == tables.AccountStatusOnCross {
			apiResp.ApiRespErr(api_code.ApiCodeOnCross, "account on cross")
			return nil
		} else if strings.EqualFold(acc.Owner, "0x0000000000000000000000000000000000000000") {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "inviter account owner is 0x0")
			return nil
		}
	}

	if req.ChannelAccount != "" {
		accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.ChannelAccount))
		acc, err := h.dbDao.GetAccountInfoByAccountId(accountId)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "search channel account fail")
			return fmt.Errorf("GetAccountInfoByAccountId err: %s", err.Error())
		} else if acc.Id == 0 || acc.Status == tables.AccountStatusOnCross || acc.IsExpired() {
			//apiResp.ApiRespErr(api_code.ApiCodeChannelAccountNotExist, "channel account not exist")
			//return nil
			req.ChannelAccount = ""
		}
	}
	if coinType != "" {
		if ok, _ := regexp.MatchString("^(0|[1-9][0-9]*)$", coinType); !ok {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("CoinType [%s] is invalid", coinType))
			return nil
		}
	}
	if crossCoinType != "" {
		if crossCoinType != string(common.CoinTypeEth) {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("CrossCoinType [%s] is invalid", coinType))
			return nil
		}
		//if ok, _ := regexp.MatchString("^(0|[1-9][0-9]*)$", crossCoinType); !ok {
		//	apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("CrossCoinType [%s] is invalid", coinType))
		//	return nil
		//}
	}
	return nil
}

func (h *HttpHandle) doRegisterOrder(req *ReqOrderRegister, apiResp *api_code.ApiResp, resp *RespOrderRegister) {
	// pay amount
	addrHex := core.DasAddressHex{
		DasAlgorithmId: req.ChainType.ToDasAlgorithmId(true),
		AddressHex:     req.Address,
		IsMulti:        false,
		ChainType:      req.ChainType,
	}
	args, err := h.dasCore.Daf().HexToArgs(addrHex, addrHex)
	if err != nil {
		log.Error("HexToArgs err: ", err.Error())
		apiResp.ApiRespErr(api_code.ApiCodeError500, "HexToArgs err")
		return
	}
	accLen := uint8(len(req.AccountCharStr))
	if tables.EndWithDotBitChar(req.AccountCharStr) {
		accLen -= 4
	}

	var coupon *tables.TableCoupon
	amountTag := 1
	if req.GiftCard != "" {
		if req.RegisterYears != 1 {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
			return
		}

		coupon = h.checkCoupon(req.GiftCard)
		if coupon == nil {
			apiResp.ApiRespErr(api_code.ApiCodeCouponInvalid, "gift card not found")
			return
		}

		accountAttr := AccountAttr{
			Length: accLen,
		}
		if res := h.checkCouponType(accountAttr, coupon); !res {
			apiResp.ApiRespErr(api_code.ApiCodeCouponInvalid, "gift card type err")
			return
		}

		req.InviterAccount = ""
		req.ChannelAccount = ""

		if err := h.rc.GetCouponLockWithRedis(coupon.Code, time.Minute*1); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeOperationFrequent, "the gift card operation is too frequent")
			return
		}
		amountTag = 0
	}

	amountTotalUSD, amountTotalCKB, amountTotalPayToken, err := h.getOrderAmount(accLen, common.Bytes2Hex(args), req.Account, req.InviterAccount, req.RegisterYears, false, req.PayTokenId)
	if err != nil {
		log.Error("getOrderAmount err: ", err.Error())
		apiResp.ApiRespErr(api_code.ApiCodeError500, "get order amount fail")
		return
	}

	if amountTotalUSD.Cmp(decimal.Zero) != amountTag || amountTotalCKB.Cmp(decimal.Zero) != amountTag || amountTotalPayToken.Cmp(decimal.Zero) != amountTag {
		log.Error("order amount err:", amountTotalUSD, amountTotalCKB, amountTotalPayToken)
		apiResp.ApiRespErr(api_code.ApiCodeError500, "get order amount fail")
		return
	}

	inviterAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.InviterAccount))
	if _, ok := config.Cfg.InviterWhitelist[inviterAccountId]; ok {
		req.ChannelAccount = req.InviterAccount
	}
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	orderContent := tables.TableOrderContent{
		AccountCharStr: req.AccountCharStr,
		InviterAccount: req.InviterAccount,
		ChannelAccount: req.ChannelAccount,
		RegisterYears:  req.RegisterYears,
		AmountTotalUSD: amountTotalUSD,
		AmountTotalCKB: amountTotalCKB,
	}

	contentDataStr, err := json.Marshal(&orderContent)
	if err != nil {
		log.Error("json marshal err:", err.Error())
		apiResp.ApiRespErr(api_code.ApiCodeError500, "json marshal fail")
		return
	}

	// check balance
	if req.PayTokenId == tables.TokenIdDas {
		dasLock, _, err := h.dasCore.Daf().HexToScript(addrHex)
		if err != nil {
			log.Error("HexToArgs err: ", err.Error())
			apiResp.ApiRespErr(api_code.ApiCodeError500, "HexToArgs err")
			return
		}

		fee := common.OneCkb
		needCapacity := amountTotalPayToken.BigInt().Uint64()
		_, _, err = h.dasCore.GetBalanceCells(&core.ParamGetBalanceCells{
			DasCache:          h.dasCache,
			LockScript:        dasLock,
			CapacityNeed:      needCapacity + fee,
			CapacityForChange: common.DasLockWithBalanceTypeOccupiedCkb,
			SearchOrder:       indexer.SearchOrderDesc,
		})
		if err != nil {
			checkBalanceErr(err, apiResp)
			return
		}
	}

	order := tables.TableDasOrderInfo{
		Id:                0,
		OrderType:         tables.OrderTypeSelf,
		OrderId:           "",
		AccountId:         accountId,
		Account:           req.Account,
		Action:            common.DasActionApplyRegister,
		ChainType:         req.ChainType,
		Address:           req.Address,
		Timestamp:         time.Now().UnixNano() / 1e6,
		PayTokenId:        req.PayTokenId,
		PayType:           req.PayType,
		PayAmount:         amountTotalPayToken,
		Content:           string(contentDataStr),
		PayStatus:         tables.TxStatusDefault,
		HedgeStatus:       tables.TxStatusDefault,
		PreRegisterStatus: tables.TxStatusDefault,
		OrderStatus:       tables.OrderStatusDefault,
		RegisterStatus:    tables.RegisterStatusConfirmPayment,
		CoinType:          req.CoinType,
		CrossCoinType:     req.CrossCoinType,
	}
	order.CreateOrderId()

	resp.OrderId = order.OrderId
	resp.TokenId = req.PayTokenId
	resp.PayType = req.PayType
	resp.Amount = order.PayAmount
	resp.CodeUrl = ""
	if coupon == nil {
		if addr, ok := config.Cfg.PayAddressMap[order.PayTokenId.ToChainString()]; !ok {
			apiResp.ApiRespErr(api_code.ApiCodeError500, fmt.Sprintf("not supported [%s]", order.PayTokenId))
			return
		} else {
			resp.ReceiptAddress = addr
		}
	}

	if coupon != nil {
		order.PayStatus = tables.TxStatusSending
		err := h.dbDao.CreateCouponOrder(&order, coupon.Code)
		if err := h.rc.DeleteCouponLockWithRedis(coupon.Code); err != nil {
			log.Error("delete coupon redis lock error : ", err.Error())
		}
		if err != nil {
			log.Error("CreateOrder err:", err.Error())
			apiResp.ApiRespErr(api_code.ApiCodeError500, "create order fail")
			return
		}
	} else {
		if err := h.dbDao.CreateOrder(&order); err != nil {
			log.Error("CreateOrder err:", err.Error())
			apiResp.ApiRespErr(api_code.ApiCodeError500, "create order fail")
			return
		}
	}

	// notify
	go func() {
		notify.SendLarkOrderNotify(&notify.SendLarkOrderNotifyParam{
			Key:        config.Cfg.Notify.LarkRegisterKey,
			Action:     "new register order",
			Account:    order.Account,
			OrderId:    order.OrderId,
			ChainType:  order.ChainType,
			Address:    order.Address,
			PayTokenId: order.PayTokenId,
			Amount:     order.PayAmount,
		})
	}()
	return
}
func (h *HttpHandle) getOrderAmount(accLen uint8, args, account, inviterAccount string, years int, isRenew bool, payTokenId tables.PayTokenId) (amountTotalUSD decimal.Decimal, amountTotalCKB decimal.Decimal, amountTotalPayToken decimal.Decimal, e error) {
	// pay token
	if payTokenId == tables.TokenCoupon {
		amountTotalUSD = decimal.Zero
		amountTotalCKB = decimal.Zero
		amountTotalPayToken = decimal.Zero
		return
	}
	payToken := timer.GetTokenInfo(payTokenId)
	if payToken.TokenId == "" {
		e = fmt.Errorf("not supported [%s]", payTokenId)
		return
	}
	//
	quoteCell, err := h.dasCore.GetQuoteCell()
	if err != nil {
		e = fmt.Errorf("GetQuoteCell err: %s", err.Error())
		return
	}
	quote := quoteCell.Quote()
	decQuote := decimal.NewFromInt(int64(quote)).Div(decimal.NewFromInt(common.UsdRateBase))
	// base price
	baseAmount, accountPrice, err := h.getAccountPrice(accLen, args, account, isRenew)
	if err != nil {
		e = fmt.Errorf("getAccountPrice err: %s", err.Error())
		return
	}
	if isRenew {
		baseAmount = decimal.Zero
	}
	accountPrice = accountPrice.Mul(decimal.NewFromInt(int64(years)))
	if inviterAccount != "" {
		builder, err := h.dasCore.ConfigCellDataBuilderByTypeArgsList(common.ConfigCellTypeArgsPrice)
		if err != nil {
			e = fmt.Errorf("ConfigCellDataBuilderByTypeArgsList err: %s", err.Error())
			return
		}
		discount, _ := builder.PriceInvitedDiscount()
		decDiscount := decimal.NewFromInt(int64(discount)).Div(decimal.NewFromInt(common.PercentRateBase))
		accountPrice = accountPrice.Mul(decimal.NewFromInt(1).Sub(decDiscount))
	}
	amountTotalUSD = accountPrice

	log.Info("before Premium:", account, isRenew, amountTotalUSD, baseAmount, accountPrice)
	if config.Cfg.Das.Premium.Cmp(decimal.Zero) == 1 {
		amountTotalUSD = amountTotalUSD.Mul(config.Cfg.Das.Premium.Add(decimal.NewFromInt(1)))
	}
	if config.Cfg.Das.Discount.Cmp(decimal.Zero) == 1 {
		amountTotalUSD = amountTotalUSD.Mul(config.Cfg.Das.Discount)
	}
	amountTotalUSD = amountTotalUSD.Add(baseAmount)
	log.Info("after Premium:", account, isRenew, amountTotalUSD, baseAmount, accountPrice)

	amountTotalUSD = amountTotalUSD.Mul(decimal.NewFromInt(100)).Ceil().DivRound(decimal.NewFromInt(100), 2)
	amountTotalCKB = amountTotalUSD.Div(decQuote).Mul(decimal.NewFromInt(int64(common.OneCkb))).Ceil()
	amountTotalPayToken = amountTotalUSD.Div(payToken.Price).Mul(decimal.New(1, payToken.Decimals)).Ceil()

	log.Info("getOrderAmount:", amountTotalUSD, amountTotalCKB, amountTotalPayToken)
	if payToken.TokenId == tables.TokenIdCkb {
		amountTotalPayToken = amountTotalCKB
	} else if payToken.TokenId == tables.TokenIdMatic || payToken.TokenId == tables.TokenIdBnb || payToken.TokenId == tables.TokenIdEth {
		log.Info("amountTotalPayToken:", amountTotalPayToken.String())
		decCeil := decimal.NewFromInt(1e6)
		amountTotalPayToken = amountTotalPayToken.DivRound(decCeil, 6).Ceil().Mul(decCeil)
		log.Info("amountTotalPayToken:", amountTotalPayToken.String())
	}
	return
}

func (h *HttpHandle) checkCoupon(code string) (coupon *tables.TableCoupon) {
	salt := config.Cfg.Server.CouponEncrySalt
	if salt == "" {
		log.Error("GetCoupon err: config coupon_encry_salt is empty")
		return
	}
	code = couponEncry(code, salt)
	res, err := h.dbDao.GetCouponByCode(code)
	if err != nil {
		log.Error("GetCoupon err:", err.Error())
		return
	}
	if res.Id == 0 || res.OrderId != "" {
		return
	}

	nowTime := time.Now().Unix()
	if nowTime < res.StartAt.Unix() || nowTime > res.ExpiredAt.Unix() {
		return
	}
	return &res
}

func (h *HttpHandle) checkCouponType(accountAttr AccountAttr, coupon *tables.TableCoupon) bool {
	if coupon.CouponType == tables.CouponType4bit && accountAttr.Length == 4 {
		return true
	}
	if coupon.CouponType == tables.CouponType5bit && accountAttr.Length >= 5 {
		return true
	}
	return false
}
