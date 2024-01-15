package websocket

import (
	"errors"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"sync"
	"time"
	"trpc.group/trpc-go/trpc-go/log"
)

// 客户端读写消息
type wsMessage struct {
	messageType int
	data        []byte
}

// 客户端连接
type wsConnection struct {
	wsSocket *websocket.Conn // 底层websocket
	inChan   chan *wsMessage // 读队列
	outChan  chan *wsMessage // 写队列

	mutex     sync.Mutex // 避免重复关闭管道
	isClosed  bool
	closeChan chan byte // 关闭通知
}

// WsServer : websocket服务
type WsServer struct {
	// 定义一个 upgrade 类型用于升级 http 为 websocket
	upgrade       *websocket.Upgrader
	listener      net.Listener
	addr          string
	wsConnections map[string]*wsConnection
}

var ws *WsServer

// NewWsServer websocket监听得地址加端口如0.0.0.0:8081
func NewWsServer(addr string) *WsServer {
	ws = &WsServer{}
	ws.addr = addr
	ws.wsConnections = make(map[string]*wsConnection)
	ws.upgrade = &websocket.Upgrader{
		ReadBufferSize:  4096, //指定读缓存区大小
		WriteBufferSize: 1024, // 指定写缓存区大小
		// 检测请求来源
		CheckOrigin: func(r *http.Request) bool {
			if r.Method != "GET" {
				log.Errorf("method is not GET")
				return false
			}
			if r.URL.Path != "/rcc/msg" {
				log.Errorf("path error")
				return false
			}
			return true
		},
	}
	return ws
}

// Start 服务启动
func (self *WsServer) Start() (err error) {
	self.listener, err = net.Listen("tcp", self.addr)
	if err != nil {
		log.Errorf("net listen error: %v", err)
		return
	}
	err = http.Serve(self.listener, self)
	if err != nil {
		log.Errorf("http serve error: %v", err)
		return
	}
	return nil
}

// ServeHTTP web
func (self *WsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 收到 http 请求后 升级 协议
	conn, err := self.upgrade.Upgrade(w, r, nil)
	if err != nil {
		log.Trace("websocket error:", err)
		return
	}
	log.Infof("client connect : %v", conn.RemoteAddr())
	wsConn := &wsConnection{
		wsSocket:  conn,
		inChan:    make(chan *wsMessage, 1000),
		outChan:   make(chan *wsMessage, 1000),
		closeChan: make(chan byte),
		isClosed:  false,
	}
	log.Infof("ws con is (%#v)", wsConn)
	self.wsConnections[conn.RemoteAddr().String()] = wsConn
	log.Infof("init after wsConMap is %#v", self.wsConnections)
	// 保活心跳
	go wsConn.procLoop()
	// 读协程
	go wsConn.wsReadLoop()
	// 写协程
	go wsConn.wsWriteLoop()
}

// Handle websocket对外发送消息 */
func (self *WsServer) Handle(data []byte) error {
	log.Debugf("send msg ! value: %v ", string(data))
	log.Debugf("connections now is %v", self.wsConnections)

	for _, c := range self.wsConnections {
		if err := c.wsWrite(websocket.TextMessage, data); err != nil {
			log.Errorf("send ws data fail. c is %v ", c)
			c.wsClose()
			break
		}
	}
	return nil
}

func (wsConn *wsConnection) wsReadLoop() {
	for {
		// 读一个message
		msgType, data, err := wsConn.wsSocket.ReadMessage()
		if err != nil {
			goto error
		}
		req := &wsMessage{
			msgType,
			data,
		}
		// 放入请求队列
		select {
		case wsConn.inChan <- req:
		case <-wsConn.closeChan:
			goto closed
		}
	}
error:
	wsConn.wsClose()
closed:
}

func (wsConn *wsConnection) wsWriteLoop() {
	for {
		select {
		// 取一个应答
		case msg := <-wsConn.outChan:
			// 写给websocket
			if err := wsConn.wsSocket.WriteMessage(msg.messageType, msg.data); err != nil {
				goto error
			}
		case <-wsConn.closeChan:
			goto closed
		}
	}
error:
	wsConn.wsClose()
closed:
}

func (wsConn *wsConnection) procLoop() {
	// 启动一个gouroutine发送心跳
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := wsConn.wsWrite(websocket.TextMessage, []byte("heartbeat from server")); err != nil {
					log.Errorf("heartbeat fail")
					wsConn.wsClose()
					break
				}
			}
		}
	}()

	//处理websocket从客户端发送的指令，此处为预留不做处理
	for {
		msg, err := wsConn.wsRead()
		if err != nil {
			log.Errorf("read fail")
			break
		}
		log.Info(string(msg.data))
		err = wsConn.wsWrite(msg.messageType, msg.data)
		if err != nil {
			log.Errorf("write fail")
			break
		}
	}
}

func (wsConn *wsConnection) wsWrite(messageType int, data []byte) error {
	select {
	case wsConn.outChan <- &wsMessage{messageType, data}:
	case <-wsConn.closeChan:
		return errors.New("websocket closed")
	}
	return nil
}

func (wsConn *wsConnection) wsRead() (*wsMessage, error) {
	select {
	case msg := <-wsConn.inChan:
		return msg, nil
	case <-wsConn.closeChan:
	}
	return nil, errors.New("websocket closed")
}

func (wsConn *wsConnection) wsClose() {
	//强制关闭websocket连接
	wsConn.wsSocket.Close()

	wsConn.mutex.Lock()
	defer wsConn.mutex.Unlock()
	if !wsConn.isClosed {
		log.Infof("client disconnect : %v", wsConn.wsSocket.RemoteAddr().String())
		delete(ws.wsConnections, wsConn.wsSocket.RemoteAddr().String())
		wsConn.isClosed = true
		close(wsConn.closeChan)
	}
}
