package game

import (
	"encoding/json"
	"fmt"
	"github.com/llr104/LiFrame/core/liFace"
	"github.com/llr104/LiFrame/core/liNet"
	"github.com/llr104/LiFrame/proto"
	"github.com/llr104/LiFrame/server/app"
	"github.com/llr104/LiFrame/utils"
	"time"
)


var Enter EnterGame

func init() {
	Enter = EnterGame{}
}

func ClientConnStart(conn liFace.IConnection) {
	app.MClientData.Inc()
	utils.Log.Info("ClientConnStart:%s", conn.RemoteAddr().String())
}

func ClientConnStop(conn liFace.IConnection) {
	app.MClientData.Dec()

	//修改离线用户
	user, err := conn.GetProperty("userId")
	if err == nil {
		Id := user.(uint32)
		ok, state := GUserMgr.UserIsIn(Id)
		if ok {
			GUserMgr.UserChangeState(Id, GUserStateOffLine, state.SceneId, nil)
			r := game.userOffLine(Id)
			if r {
				GUserMgr.UserChangeState(Id, GUserStateLeave, -1,nil)
			}
		}
	}

	utils.Log.Info("ClientConnStop:%s", conn.RemoteAddr().String())
}

func ShutDown(){
	utils.Log.Info("ShutDown")
	game.shutDown()
}

type EnterGame struct {
	liNet.BaseRouter
}

func (s *EnterGame) NameSpace() string {
	return "*.*"
}

func (s *EnterGame) EveryThingHandle(req liFace.IRequest) {

	//进入请求，授权
	if req.GetMsgName() == proto.GameEnterGameReq{
		reqInfo := proto.EnterGameReq{}
		ackInfo := proto.EnterGameAck{}
		if err := json.Unmarshal(req.GetData(), &reqInfo); err != nil {
			ackInfo.Code = proto.Code_Illegal
			utils.Log.Info("GameEnterGameReq error:%s", err.Error())
		} else {
			//检测进入处理逻辑
			r := game.enterGame(reqInfo)
			if r {
				ackInfo.Code = proto.Code_Success
				req.GetConnection().SetProperty("userId", reqInfo.UserId)
			}else{
				ackInfo.Code = proto.Code_EnterGameError
			}
		}
		data, _ := json.Marshal(ackInfo)
		req.GetConnection().SendMsg(proto.GameEnterGameAck, data)

	}else if req.GetMsgName() == protoLogoutReq{
		//通知场景
		userId, err := req.GetConnection().GetProperty("userId")
		if err == nil {
			d := userId.(uint32)
			game.userLogout(d)
		}

		req.GetConnection().RemoveProperty("userId")
		req.GetConnection().SendMsg(protoLogoutAck, nil)

	} else if req.GetMsgName() == protoHeartBeatReq{
		h := heartBeat{}
		json.Unmarshal(req.GetData(), &h)
		h.ServerTimeStamp = time.Now().UnixNano() / 1e6
		data,_ := json.Marshal(h)
		req.GetConnection().SendMsg(protoHeartBeatAck, data)
	} else{
		//验证连接是否已经授权能进入游戏了
		userId, err := req.GetConnection().GetProperty("userId")
		if err == nil {
			d := userId.(uint32)
			game.gameMessage(d, req.GetMsgName(), req.GetData(), req.GetConnection())
		}else{
			v := req.GetMsgName()
			fmt.Println(v)
			req.GetConnection().SendMsg(proto.AuthError, nil)
		}
	}

}