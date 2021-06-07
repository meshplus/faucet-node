package app

import (
	"context"
	"faucet/internal"
	"faucet/internal/loggers"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
)

//2. api：input： net，contractAddress，address； output：0，hash
//3. 验证leveldb， key：address； value：[timestamp, net（eth，bxh），amount, contartAddress] , 每天发一个
//4. 调用对应测试网交易

type Server struct {
	router *gin.Engine
	logger logrus.FieldLogger
	client *internal.Client

	ctx    context.Context
	cancel context.CancelFunc
}

type ComeInput struct {
	Net     string `json:"net"`
	Address string `json:"address"`
}

type response struct {
	Msg  []byte `json:"msg"`
	Data []byte `json:"txHash"`
}

func NewServer(client *internal.Client) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	return &Server{
		router: router,
		client: client,
		ctx:    ctx,
		cancel: cancel,
		logger: loggers.Logger(loggers.ApiServer),
	}, nil
}

func (g *Server) Start() error {
	g.router.Use(gin.Recovery())
	v1 := g.router.Group("/v1")
	{
		v1.POST("come", g.come)
	}

	go func() {
		g.logger.Infoln("start gin success")
		err := g.router.Run(fmt.Sprintf(":%s", g.client.Config.Network.Port))
		if err != nil {
			panic(err)
		}
		<-g.ctx.Done()
	}()
	return nil
}

//func (g *Server) Stop() error {
//	g.cancel()
//	g.logger.Infoln("gin service stop")
//	return nil
//}

func (g *Server) come(c *gin.Context) {
	res := &response{}
	var comeInput ComeInput
	if err := c.BindJSON(&comeInput); err != nil {
		res.Msg = []byte(err.Error())
		c.JSON(http.StatusBadRequest, res)
		return
	}
	data, err := g.client.SendTra(comeInput.Net, comeInput.Address)
	if err != nil {
		res.Msg = []byte(err.Error())
		c.JSON(http.StatusInternalServerError, res)
		return
	}
	res.Msg = []byte("ok")
	res.Data = []byte(data)
	c.PureJSON(http.StatusOK, res)
}

func (g *Server) Stop() error {
	g.client.Close()
	g.cancel()
	g.logger.Infoln("gin service stop")
	return nil
}
