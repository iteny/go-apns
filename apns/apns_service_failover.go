package apns

import (
	log "github.com/blackbeans/log4go"
	"go-apns/entry"
	"time"
)

//接受错误的响应并触发重发
func (self *ApnsClient) onErrorResponseRecieve(responseChannel chan *entry.Response) {

	ch := make(chan *entry.Message, 1000)
	//启动重发任务
	go self.resend(ch)

	//开始启动
	for self.running {
		//顺序处理每一个连接的错误数据发送
		resp := <-responseChannel
		//只有 prcessing error 和 shutdown的两种id才会进行重发
		switch resp.Status {

		case entry.RESP_SHUTDOWN, entry.RESP_ERROR, entry.RESP_UNKNOW:

			//只有这三种才重发
			msg, succ := self.storage.Remove(resp.Identifier, 0, func(id uint32, msg *entry.Message) bool {
				expiredTime := int64(entry.UmarshalExpiredTime(msg))

				//过滤掉 不是当前连接ID的消息 或者 当前相同ID的消息 或者 (有过期时间结果已经过期的消息)
				return msg.ProcessId != resp.ProccessId ||
					id == resp.Identifier ||
					(0 != expiredTime && (time.Now().Unix()-expiredTime >= 0))

			})

			//如果删除成功并且消息不为空则重发
			if succ && nil != msg {
				ch <- msg
			}
			log.DebugLog("apns_debug", "APNSCLIENT|onErrorResponseRecieve|ERROR|%s|%d", msg, resp.Status)

		case entry.RESP_INVALID_TOKEN, entry.RESP_INVALID_TOKEN_SIZE:
			//将错误的token记录在存储中，备后续的过滤使用
			msg := self.storage.Get(resp.Identifier)
			if nil != msg {
				//从msg中拿出token用于记录
				token := entry.UmarshalToken(msg)
				self.storeInvalidToken(token)
				log.DebugLog("apns_debug", "APNSCLIENT|INVALID TOKEN|%s", resp.Identifier)
			}
		}

	}
}

//重发逻辑
func (self *ApnsClient) resend(ch chan *entry.Message) {

	for self.running {
		select {
		case <-time.After(5 * time.Second):
		case msg := <-ch:
			//发送之......
			self.sendMessage(msg)
			self.resendCounter.Incr(1)
			log.DebugLog("apns_debug", "APNSCLIENT|RESEND|%s\n", msg)
		}
	}

}

func (self *ApnsClient) storeInvalidToken(token string) {
	//这里是里面最后存储不合法的token
	log.Warn("APNSCLIENT|UnImplement StoreInvalidToken|%s\n", token)
}
