package init

import (
	conv "open_im_sdk/internal/controller/conversation_msg"
	"open_im_sdk/internal/controller/friend"
	"open_im_sdk/internal/controller/group"
	ws "open_im_sdk/internal/controller/interaction"
	"open_im_sdk/internal/controller/user"
	"open_im_sdk/internal/open_im_sdk"
	"open_im_sdk/pkg/common"
	"open_im_sdk/pkg/constant"
	"open_im_sdk/pkg/db"
	"open_im_sdk/pkg/log"
	"open_im_sdk/pkg/utils"
	"time"
)

type LoginMgr struct {
	friend       *friend.Friend
	group        *group.Group
	conversation *conv.Conversation
	user         *user.User

	db      *db.DataBase
	ws      *ws.Ws
	msgSync *MsgSync

	heartbeat *Heartbeat

	token       string
	loginUserID string
	listener    ws.ConnListener

	justOnceFlag bool
}

func (u *LoginMgr) login(userID, token string, cb common.Base) {
	log.Info("login start ", userID, token)
	if cb == nil {
		return
	}
	if u.justOnceFlag {
		cb.OnError(constant.ErrLogin.ErrCode, constant.ErrLogin.ErrMsg)
		return
	}
	err := u.checkToken(token)
	if err != nil {
		cb.OnError(constant.ErrTokenInvalid.ErrCode, constant.ErrTokenInvalid.ErrMsg)
		return
	}
	u.justOnceFlag = true

	u.token = token
	u.loginUserID = userID

	db, err := db.NewDataBase(userID)
	if err != nil {
		cb.OnError(constant.ErrDB.ErrCode, constant.ErrDB.ErrMsg)
		log.Error("", "NewDataBase failed ", err.Error())
		return
	}
	u.db = db
	u.ws = ws.NewWs(ws.NewWsRespAsyn(), ws.NewWsConn(u.listener, token, userID))
	u.msgSync = NewMsgSync(db, u.ws, userID)

	u.heartbeat = NewHeartbeat(u.ws, u.msgSync)

	log.Info("ws, forcedSynchronization heartbeat coroutine run ...")
	go u.forcedSynchronization()
	go u.heartbeat.heartbeat()
	go u.run()
	//		go u.timedCloseDB()
	u.forycedSyncReceiveMessageOpt()
	cb.OnSuccess("")

}



func (u *LoginMgr) InitSDK(config string, cb *ws.ConnListener) bool {
	if cb == nil {
		log.Error("callback == nil")
		return false
	}

	log.Info("initSDK LoginState ")

	u.listener = cb
	u.initListenerCh()
	utils.sdkLog("init success, ", config)

	go open_im_sdk.doListener(u)
	return true
}

func (u *LoginMgr) UnInitSDK() {
	u.unInitAll()
	u.closeListenerCh()
}

func (u *LoginMgr) GetVersion() string {
	return "v1.0.5"
}


func (u *LoginMgr) logout(cb Base) {

		u.LoginState = constant.LogoutCmd

		utils.sdkLog("set LoginState ", u.LoginState)

		err := u.closeConn()
		if err != nil {
			if cb != nil {
				cb.OnError(constant.ErrCodeInitLogin, err.Error())
			}
			return
		}
		utils.sdkLog("closeConn ok")

		//err = u.closeDB()
		if err != nil {
			if cb != nil {
				cb.OnError(constant.ErrCodeInitLogin, err.Error())
			}
			return
		}
		utils.sdkLog("close db ok")

		u.loginUserID = ""
		u.token = ""
		time.Sleep(time.Duration(6) * time.Second)
		if cb != nil {
			cb.OnSuccess("")
		}
		utils.sdkLog("logout return")
}


func (u *LoginMgr) GetLoginUser() string {
	if u.LoginState == constant.LoginSuccess {
		return u.loginUserID
	} else {
		return ""
	}
}

func (im *LoginMgr) GetLoginStatus() int {
	return im.LoginState
}
//
//func (u *open_im_sdk.UserRelated) forycedSyncReceiveMessageOpt() {
//	OperationID := utils.operationIDGenerator()
//	resp, err := utils.post2ApiForRead(open_im_sdk.getAllConversationMessageOptRouter, open_im_sdk.paramGetAllConversationMessageOpt{OperationID: OperationID}, u.token)
//	if err != nil {
//		utils.sdkLog("post2Api failed, ", open_im_sdk.getAllConversationMessageOptRouter, OperationID)
//		return
//	}
//	var v open_im_sdk.getReceiveMessageOptResp
//	err = json.Unmarshal(resp, &v)
//	if err != nil {
//		utils.sdkLog("Unmarshal failed ", resp, OperationID)
//		return
//	}
//	if v.ErrCode != 0 {
//		utils.sdkLog("errCode failed, ", v.ErrCode, v.ErrMsg, string(resp), OperationID)
//		return
//	}
//
//	utils.sdkLog("get receive opt ", v)
//	u.receiveMessageOptMutex.Lock()
//	for _, v := range v.Data {
//		if v.Result != 0 {
//			u.receiveMessageOpt[v.ConversationId] = v.Result
//		}
//	}
//	u.receiveMessageOptMutex.Unlock()
//}

func (u *LoginMgr) forcedSynchronization() {
	u.friend.SyncFriendList()
	u.friend.SyncBlackList()
	u.friend.SyncFriendApplication()
	u.friend.SyncSelfFriendApplication()
	u.user.SyncLoginUserInfo()
	u.group.SyncApplyGroupRequest()
	u.group.SyncGroupRequest()
	u.group.SyncJoinedGroupInfo()
	u.group.SyncSelfGroupRequest()
}

func (u *LoginMgr) GetMinSeqSvr() int64 {
	return u.GetMinSeqSvr()
}

func (u *LoginMgr) SetMinSeqSvr(minSeqSvr int64) {
	u.SetMinSeqSvr(minSeqSvr)
}

func (u *LoginMgr)checkToken(token string) error {
	p := ws.NewPostApi(token, constant.SvrConf.ApiAddr)
	_, err := u.user.GetSelfUserInfoFromSvr()
	return utils.Wrap(err, "GetSelfUserInfoFromSvr failed")
}


//func (u *open_im_sdk.UserRelated) kickOnline(msg utils.GeneralWsResp) {
//	utils.sdkLog("kickOnline ", msg.ReqIdentifier, msg.ErrCode, msg.ErrMsg)
//	u.logout(nil)
//	u.cb.OnKickedOffline()
//}
