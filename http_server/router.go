package http_server

import (
	"das_register_server/config"
	"das_register_server/http_server/api_code"
	"encoding/json"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/http_api"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"time"
)

func (h *HttpServer) initRouter() {
	originList := []string{
		`https:\/\/[^.]*\.bestdas\.com`,
		`https:\/\/[^.]*\.da\.systems`,
		`https:\/\/bestdas\.com`,
		`https:\/\/da\.systems`,
		`https:\/\/app\.gogodas\.com`,
	}
	if len(config.Cfg.Origins) > 0 {
		toolib.AllowOriginList = append(toolib.AllowOriginList, config.Cfg.Origins...)
	} else {
		toolib.AllowOriginList = append(toolib.AllowOriginList, originList...)
	}
	h.engine.Use(toolib.MiddlewareCors())
	h.engine.Use(sentrygin.New(sentrygin.Options{
		Repanic: true,
	}))
	h.engine.Use(http_api.ReqIdMiddleware())
	v1 := h.engine.Group("v1")
	{
		// cache
		shortExpireTime, longExpireTime, lockTime := time.Second*5, time.Second*15, time.Minute
		shortDataTime, longDataTime := time.Minute*3, time.Minute*10
		cacheHandleShort := toolib.MiddlewareCacheByRedis(h.rc.GetRedisClient(), false, shortDataTime, lockTime, shortExpireTime, respHandle)
		cacheHandleLong := toolib.MiddlewareCacheByRedis(h.rc.GetRedisClient(), false, longDataTime, lockTime, longExpireTime, respHandle)
		//cacheHandleShortCookies := toolib.MiddlewareCacheByRedis(h.rc.GetRedisClient(), true, shortDataTime, lockTime, shortExpireTime, respHandle)

		//v1.POST("/query", cacheHandleShort, h.h.Query)
		//v1.POST("/operate", h.h.Operate)
		// query
		v1.GET("/version", api_code.DoMonitorLog("Version"), cacheHandleShort, h.h.Version)
		v1.POST("/token/list", api_code.DoMonitorLog(api_code.MethodTokenList), cacheHandleLong, h.h.TokenList)
		v1.POST("/config/info", api_code.DoMonitorLog(api_code.MethodConfigInfo), cacheHandleShort, h.h.ConfigInfo)
		v1.POST("/account/list", api_code.DoMonitorLog(api_code.MethodAccountList), cacheHandleLong, h.h.AccountList) // user's not on sale accounts
		v1.POST("/account/mine", api_code.DoMonitorLog(api_code.MethodAccountMine), cacheHandleLong, h.h.AccountMine) // user's accounts by pagination
		v1.POST("/account/detail", api_code.DoMonitorLog(api_code.MethodAccountDetail), cacheHandleLong, h.h.AccountDetail)
		v1.POST("/account/records", api_code.DoMonitorLog(api_code.MethodAccountRecords), cacheHandleShort, h.h.AccountRecords)
		//v1.POST("/reverse/latest", api_code.DoMonitorLog(api_code.MethodReverseLatest), cacheHandleShort, h.h.ReverseLatest)
		//v1.POST("/reverse/list", api_code.DoMonitorLog(api_code.MethodReverseList), cacheHandleShort, h.h.ReverseList)
		v1.POST("/transaction/status", api_code.DoMonitorLog(api_code.MethodTransactionStatus), cacheHandleShort, h.h.TransactionStatus)
		v1.POST("/balance/info", api_code.DoMonitorLog(api_code.MethodBalanceInfo), cacheHandleLong, h.h.BalanceInfo) // balance（712，not 712，sort address）
		v1.POST("/transaction/list", api_code.DoMonitorLog(api_code.MethodTransactionList), cacheHandleLong, h.h.TransactionList)
		v1.POST("/rewards/mine", api_code.DoMonitorLog(api_code.MethodRewardsMine), cacheHandleLong, h.h.RewardsMine)
		v1.POST("/withdraw/list", api_code.DoMonitorLog(api_code.MethodWithdrawList), cacheHandleLong, h.h.WithdrawList)
		v1.POST("/account/search", api_code.DoMonitorLog(api_code.MethodAccountSearch), cacheHandleShort, h.h.AccountSearch)
		v1.POST("/account/registering/list", api_code.DoMonitorLog(api_code.MethodRegisteringList), cacheHandleLong, h.h.RegisteringList)
		v1.POST("/account/order/detail", api_code.DoMonitorLog(api_code.MethodOrderDetail), h.h.OrderDetail)
		v1.POST("/address/deposit", api_code.DoMonitorLog(api_code.MethodAddressDeposit), cacheHandleLong, h.h.AddressDeposit)
		v1.POST("/character/set/list", api_code.DoMonitorLog(api_code.MethodCharacterSetList), cacheHandleLong, h.h.CharacterSetList)
		v1.POST("/account/auction/info", api_code.DoMonitorLog(api_code.MethodAuctionInfo), h.h.GetAccountAuctionInfo)
		v1.POST("/account/auction/price", api_code.DoMonitorLog(api_code.MethodAuctionPrice), h.h.GetAccountAuctionPrice)
		v1.POST("/account/auction/order-status", api_code.DoMonitorLog(api_code.MethodAuctionOrderStatus), h.h.GetAuctionOrderStatus)
		v1.POST("/account/auction/pending-order", api_code.DoMonitorLog(api_code.MethodAuctionPendingOrder), cacheHandleLong, h.h.GetPendingAuctionOrder)
		v1.POST("/account/recommend", api_code.DoMonitorLog("account-recommend"), cacheHandleShort, h.h.AccountRecommend)
		v1.POST("/did/cell/list", api_code.DoMonitorLog("did-cell-list"), cacheHandleShort, h.h.DidCellList)
		v1.POST("/did/cell/upgradable/list", api_code.DoMonitorLog("did-cell-upgradable-list"), cacheHandleShort, h.h.DidCellUpgradableList)
		v1.POST("/did/cell/upgrade/price", api_code.DoMonitorLog("did-cell-upgrade-price"), cacheHandleShort, h.h.DidCellUpgradePrice)
		v1.POST("/did/cell/recyclable/list", api_code.DoMonitorLog("did-cell-recyclable-list"), cacheHandleShort, h.h.DidCellRecyclableList)
		//v1.POST("/did/cell/daslock/list", api_code.DoMonitorLog("did-cell-daslock-list"), cacheHandleShort, h.h.DidCellDasLockList)
		v1.POST("/did/cell/daslock/edit/owner", api_code.DoMonitorLog("did-cell-daslock-edit-owner"), h.h.DidCellDasLockEditOwner)

		// operate
		//v1.POST("/reverse/declare", api_code.DoMonitorLog(api_code.MethodReverseDeclare), h.h.ReverseDeclare)
		//v1.POST("/reverse/redeclare", api_code.DoMonitorLog(api_code.MethodReverseRedeclare), h.h.ReverseRedeclare)
		//v1.POST("/reverse/retract", api_code.DoMonitorLog(api_code.MethodReverseRetract), h.h.ReverseRetract)
		v1.POST("/transaction/send", api_code.DoMonitorLog(api_code.MethodTransactionSend), h.h.TransactionSend)
		v1.POST("/balance/pay", api_code.DoMonitorLog(api_code.MethodBalancePay), h.h.BalancePay)
		v1.POST("/balance/withdraw", api_code.DoMonitorLog(api_code.MethodBalanceWithdraw), h.h.BalanceWithdraw)
		v1.POST("/balance/transfer", api_code.DoMonitorLog(api_code.MethodBalanceTransfer), h.h.BalanceTransfer)
		v1.POST("/balance/deposit", api_code.DoMonitorLog(api_code.MethodBalanceDeposit), h.h.BalanceDeposit)
		v1.POST("/account/edit/manager", api_code.DoMonitorLog(api_code.MethodEditManager), h.h.EditManager)
		//v1.POST("/account/edit/owner", api_code.DoMonitorLog(api_code.MethodEditOwner), h.h.EditOwner)
		v1.POST("/account/edit/owner", api_code.DoMonitorLog(api_code.MethodEditOwner), h.h.DidCellEditOwner)
		//v1.POST("/account/edit/records", api_code.DoMonitorLog(api_code.MethodEditRecords), h.h.EditRecords)
		v1.POST("/account/edit/records", api_code.DoMonitorLog(api_code.MethodEditRecords), h.h.DidCellEditRecord)
		//v1.POST("/account/order/renew", api_code.DoMonitorLog(api_code.MethodOrderRenew), h.h.OrderRenew)
		v1.POST("/account/order/renew", api_code.DoMonitorLog(api_code.MethodOrderRenew), h.h.DidCellRenew)
		v1.POST("/account/order/register", api_code.DoMonitorLog(api_code.MethodOrderRegister), h.h.OrderRegister)
		v1.POST("/account/order/change", api_code.DoMonitorLog(api_code.MethodOrderChange), h.h.OrderChange)
		v1.POST("/account/order/pay/hash", api_code.DoMonitorLog(api_code.MethodOrderPayHash), h.h.OrderPayHash)
		v1.POST("/account/coupon/check", api_code.DoMonitorLog(api_code.MethodOrderCheckCoupon), cacheHandleShort, h.h.CheckCoupon)
		//v1.POST("/account/edit/script", api_code.DoMonitorLog(api_code.MethodEditScript), h.h.EditScript)
		v1.POST("/account/auction/bid", api_code.DoMonitorLog(api_code.MethodAuctionBid), h.h.AccountAuctionBid)
		v1.POST("/did/cell/recycle", api_code.DoMonitorLog("did-cell-recycle"), h.h.DidCellRecycle)
		//v1.POST("/did/cell/edit/owner", api_code.DoMonitorLog("did-cell-edit-owner"), h.h.DidCellEditOwner)
		//v1.POST("/did/cell/renew", api_code.DoMonitorLog("did-cell-renew"), h.h.DidCellRenew)
		//v1.POST("/did/cell/edit/record", api_code.DoMonitorLog("did-cell-edit-record"), h.h.DidCellEditRecord)

		// node rpc
		v1.POST("/node/ckb/rpc", api_code.DoMonitorLog(api_code.MethodCkbRpc), h.h.CkbRpc)
		v1.GET("/test/jenkins", func(c *gin.Context) {
			c.JSON(200, "main--v1.15.3")
		})
	}

	internalV1 := h.internalEngine.Group("v1")
	{
		internalV1.POST("/refund/apply", h.h.RefundApply)
		if config.Cfg.Server.Net != common.DasNetTypeMainNet {
			internalV1.POST("/sign/tx", h.h.SignTx)
		}
		internalV1.POST("/order/info", h.h.OrderInfo)
		internalV1.POST("/account/register", h.h.AccountRegister)
		internalV1.POST("/account/renew", h.h.AccountRenew)
		internalV1.POST("/order/detail", h.h.DasOrderDetail)
		internalV1.POST("/create/coupon", h.h.CreateCoupon)
		internalV1.POST("/unipay/notice", h.h.UniPayNotice)
	}
}

func respHandle(c *gin.Context, res string, err error) {
	if err != nil {
		log.Error("respHandle err:", err.Error())
		c.AbortWithStatusJSON(http.StatusOK, api_code.ApiRespErr(http.StatusInternalServerError, err.Error()))
	} else if res != "" {
		var respMap map[string]interface{}
		_ = json.Unmarshal([]byte(res), &respMap)
		c.AbortWithStatusJSON(http.StatusOK, respMap)
	}
}
