package app

import (
	"context"
	"faucet/internal"
	"faucet/internal/loggers"
	"faucet/internal/utils"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/sirupsen/logrus"
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

type nativeInput struct {
	Net     string `json:"net"`
	Address string `json:"address"`
}

type erc20Input struct {
	Net             string `json:"net"`
	Address         string `json:"address"`
	ContractAddress string `json:"contractAddress"`
}

type response struct {
	Msg  string `json:"msg"`
	Data string `json:"txHash"`
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
	g.router.Use(gin.Recovery()).Use(cors.Default()).Use(g.MaxAllowed(200))
	v1 := g.router.Group("/faucet")
	{
		v1.POST("nativeToken", g.nativeToken)
		//v1.POST("erc20Token", g.erc20Token)
	}

	//used for chainlink oracle
	v2 := g.router.Group("/test")
	{
		v2.GET("/uintToHex", g.uintToHex)
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

func (g *Server) nativeToken(c *gin.Context) {
	res := &response{}
	var nativeInput nativeInput
	if err := c.BindJSON(&nativeInput); err != nil {
		res.Msg = err.Error()
		c.JSON(http.StatusBadRequest, res)
		return
	}
	if add := types.NewAddressByStr(nativeInput.Address); add == nil {
		res.Msg = fmt.Errorf("invalid address: %s", nativeInput.Address).Error()
		c.JSON(http.StatusInternalServerError, res)
		return
	}

	if !strings.EqualFold("bxh", nativeInput.Net) {
		res.Msg = fmt.Errorf("not support net: %s", nativeInput.Net).Error()
		c.JSON(http.StatusInternalServerError, res)
		return
	}

	g.client.GinContext = c
	data, err := g.client.SendTra(nativeInput.Net, nativeInput.Address)
	if err != nil {
		res.Msg = err.Error()
		c.JSON(http.StatusInternalServerError, res)
		return
	}
	res.Msg = "ok"
	res.Data = data
	c.PureJSON(http.StatusOK, res)
}

/*func (g *Server) erc20Token(c *gin.Context) {
	res := &response{}
	var erc20Input erc20Input
	if err := c.BindJSON(&erc20Input); err != nil {
		res.Msg = err.Error()
		c.JSON(http.StatusBadRequest, res)
		return
	}
	if add := types.NewAddressByStr(erc20Input.Address); add == nil {
		res.Msg = fmt.Errorf("invalid address: %s", erc20Input.Address).Error()
		c.JSON(http.StatusInternalServerError, res)
		return
	}
	if !strings.EqualFold("bxh", erc20Input.Net) {
		res.Msg = fmt.Errorf("not support net: %s", erc20Input.Net).Error()
		c.JSON(http.StatusInternalServerError, res)
		return
	}
	if add := types.NewAddressByStr(erc20Input.ContractAddress); add == nil {
		res.Msg = fmt.Errorf("invalid erc20 address: %s", erc20Input.ContractAddress).Error()
		c.JSON(http.StatusInternalServerError, res)
		return
	}
	g.client.GinContext = c
	data, err := g.client.SendErc20(erc20Input.Net, erc20Input.Address, erc20Input.ContractAddress)
	if err != nil {
		res.Msg = err.Error()
		c.JSON(http.StatusInternalServerError, res)
		return
	}
	res.Msg = "ok"
	res.Data = data
	c.PureJSON(http.StatusOK, res)
}*/

func (g *Server) uintToHex(c *gin.Context) {
	res := &response{}
	input := c.Query("num")
	num, err := strconv.Atoi(input)
	if err != nil {
		res.Msg = fmt.Errorf("invalid input: %s", num).Error()
		return
	}
	hexString := fmt.Sprintf("%#x", num)
	res.Msg = "ok"
	res.Data = hexString
	c.PureJSON(http.StatusOK, res)

}

func (g *Server) Stop() error {
	g.client.Close()
	g.cancel()
	g.logger.Infoln("gin service stop")
	return nil
}

// MaxAllowed 限流器
func (g *Server) MaxAllowed(limitValue int64) func(c *gin.Context) {
	limiter := utils.NewLimiter(limitValue)
	g.logger.Infof("limiter.SetMax: %d", limitValue)
	// 返回限流逻辑
	return func(c *gin.Context) {
		if !limiter.Ok() {
			c.AbortWithStatus(http.StatusServiceUnavailable) //超过每秒200，就返回503错误码
			return
		}
		c.Next()
	}
}
