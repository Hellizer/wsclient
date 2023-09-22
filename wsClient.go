package wsclient

import (
	"errors"
	"fmt"
	"net/http"

	log "github.com/Hellizer/lightlogger"
	"github.com/gorilla/websocket"
)

type MsgHandler func(msg WSMessage)

type WSMessage struct {
	MsgType int    //
	RawMsg  []byte //
	Err     error  //
}

type WsClient struct {
	conn       *websocket.Conn //
	stopWaiter chan (int)      //
	msgHandler MsgHandler      //
	endpoint   string          //
}

func NewClient(handler MsgHandler) *WsClient {
	ws := new(WsClient)
	ws.stopWaiter = make(chan int)
	ws.msgHandler = handler
	return ws
}

func (ws *WsClient) Open(endpoint string) error {
	if ws.conn != nil {
		return errors.New("already connected")
	}
	if endpoint == "" {
		return errors.New("no endpoint")
	}
	if ws.stopWaiter == nil {
		ws.stopWaiter = make(chan int)
	}
	var resp *http.Response
	var err error
	log.Print(1, log.LogInfo, "ws open", endpoint)
	ws.conn, resp, err = websocket.DefaultDialer.Dial(endpoint, nil)
	if err != nil {
		return err
	}
	//log.Print(5, log.LogInfo, "ws open", fmt.Sprintf("response staus code %d", resp.StatusCode))
	//log.Print(5, log.LogInfo, "ws open", fmt.Sprintf("response  %v", resp.Body))
	if resp.StatusCode != 101 {
		log.Print(5, log.LogError, "ws open", fmt.Sprintf("response code %d", resp.StatusCode))
		return errors.New("socket open error")
	}
	log.Print(5, log.LogInfo, "ws open", "web socket opened")
	return nil
}

func (ws *WsClient) Serve() {
	var t int
	var bmsg []byte
	var err error
	go func() {
		for {
			t, bmsg, err = ws.conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, 1000) {

				} else {
					log.Print(1, log.LogError, "ws serve", fmt.Sprintf("error: %v", err))
					ws.msgHandler(WSMessage{MsgType: t, RawMsg: bmsg, Err: err})
				}
				break
			}
			ws.msgHandler(WSMessage{MsgType: t, RawMsg: bmsg, Err: err})
		}
		close(ws.stopWaiter)
		ws.stopWaiter = nil
	}()
}

func (ws *WsClient) SendText(req []byte) error {
	if ws.conn == nil {
		return errors.New("not connected")
	}
	err := ws.conn.WriteMessage(websocket.TextMessage, req)
	if err != nil {
		log.Print(1, log.LogError, "ws send", "send message error")
		return err
	}
	return nil
}

func (ws *WsClient) Close() error {
	if ws.conn == nil {
		return errors.New("not connected")
	}
	err := ws.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Print(1, log.LogError, "ws close", "closing error while send close message")
		return err
	}
	<-ws.stopWaiter
	err = ws.conn.Close()
	if err != nil {
		log.Print(1, log.LogError, "ws close", "closing error while close connection")
		return err
	}
	ws.conn = nil
	return nil
}
