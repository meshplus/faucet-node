package internal

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"faucet/internal/loggers"
	"faucet/internal/repo"
	"faucet/internal/utils"
	"faucet/persist"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/backoff"
	"github.com/Rican7/retry/strategy"
	"github.com/axiomesh/axiom-kit/storage"
	"github.com/axiomesh/axiom-kit/storage/leveldb"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const (
	nativeToken = "native"
	amount      = 0.5
)

type Client struct {
	Config          *repo.Config
	ctx             context.Context
	axiomClient     *ethclient.Client
	axiomLock       sync.Mutex
	axiomAuth       *bind.TransactOpts
	axiomPrivateKey *ecdsa.PrivateKey
	ldb             storage.Storage
	logger          logrus.FieldLogger
	GinContext      *gin.Context
}

type AddressData struct {
	SendTxTime int64   `json:"sendTxTime"`
	TxHash     string  `json:"txHash"`
	Amount     float64 `json:"amount"`
}

func (c *Client) SendTra(net string, address string) (string, error) {
	var (
		txHash string
		err    error
	)
	// 合法校验：每天每个(net + type + addr)只发一个
	if err := c.checkLimit(net, nativeToken, address, c.ldb); err != nil {
		return "", err
	}
	txHash, err = sendTxAxm(c, address, amount)
	keyAddr := address
	if err != nil {
		return "", err
	}
	if checkTxSuccess(c, txHash) {
		if err := putTxData(txHash, c, keyAddr, nativeToken, net); err != nil {
			return "", fmt.Errorf("putTxDataFailed: %w", err)
		}
	}
	return txHash, nil
}

func putTxData(txHash string, c *Client, address string, typ string, net string) error {
	p := &AddressData{
		SendTxTime: time.Now().Unix(),
		TxHash:     txHash,
		Amount:     amount,
	}
	structJSON, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("json marshal failed: %w", err)
	}
	c.ldb.Put(c.construAddressKey(net, typ, address), structJSON)
	return nil
}

func deleteTxData(c *Client, address string, typ string, net string) error {
	c.ldb.Delete(c.construAddressKey(net, typ, address))
	return nil
}

func (c *Client) construAddressKey(net string, typ string, address string) []byte {
	// get public_ip and ip with address and net tobe key
	var buffer bytes.Buffer
	buffer.WriteString(address)
	buffer.WriteString("-")
	buffer.WriteString(typ)
	c.logger.Infof("construKey: %s ", buffer)
	return persist.CompositeKey(net, buffer)
}

func (c *Client) construIpKey(net string) []byte {
	// get public_ip and ip with address and net tobe key
	var buffer bytes.Buffer
	buffer.WriteString(time.Now().Format("2006-01-02"))
	buffer.WriteString("-")
	buffer.WriteString(utils.ClientPublicIP(c.GinContext.Request))
	buffer.WriteString("-")
	buffer.WriteString(utils.ClientIp(c.GinContext.Request))
	c.logger.Infof("construKey: %s ", buffer)
	return persist.CompositeKey(net, buffer)
}

func (c *Client) checkLimit(net string, typ string, address string, ldb storage.Storage) error {
	value := ldb.Get(c.construAddressKey(net, typ, address))
	if value != nil {
		data := AddressData{}
		if err := json.Unmarshal(value, &data); err != nil {
			return fmt.Errorf("unmarshal error")
		}
		// 获取当前时间的 Unix 时间戳
		currentUnixTime := time.Now().Unix()

		// 计算时间差（以秒为单位）
		timeDifference := currentUnixTime - data.SendTxTime

		// 定义一天的秒数
		oneDayInSeconds := int64(24 * 60 * 60)

		// 比较时间差与一天的秒数
		if timeDifference < oneDayInSeconds {
			return fmt.Errorf("The address has recently received test tokens")
		}
	}
	return nil
}

func checkTxSuccess(c *Client, txHash string) bool {
	client := c.axiomClient
	err := retry.Retry(func(attempt uint) error {
		receipt, err := client.TransactionReceipt(context.Background(), common.HexToHash(txHash))
		if err != nil {
			return err
		}
		if err == nil && receipt != nil {
			if receipt.Status == types.ReceiptStatusFailed {
				return fmt.Errorf("faucet transfer failed")
			}
		}
		return nil
	}, strategy.Limit(3), strategy.Backoff(backoff.Fibonacci(200*time.Millisecond)))
	if err != nil {
		return false
	}
	return true

}

func (c *Client) Initialize(configPath string) error {
	c.ctx = context.Background()
	cfg, err := repo.UnmarshalConfig(configPath)
	if err != nil {
		return fmt.Errorf("unmarshal config for plugin :%w", err)
	}
	c.Config = cfg
	// 构建axiom客户端
	axiomClient, err := ethclient.Dial(cfg.Axiom.AxiomAddr)
	if err != nil {
		return fmt.Errorf("dial axiom node: %w", err)
	}
	c.axiomClient = axiomClient

	// 构建auth_axm
	keyPathAxm := filepath.Join(configPath, cfg.Axiom.AxiomKeyPath)
	keyByteAxm, err := ioutil.ReadFile(keyPathAxm)
	psdPathAxm := filepath.Join(configPath, cfg.Axiom.AxiomPassword)
	passwordAxm, err := ioutil.ReadFile(psdPathAxm)
	unlockedKeyAxm, err := keystore.DecryptKey(keyByteAxm, strings.TrimSpace(string(passwordAxm)))
	authAxm := bind.NewKeyedTransactor(unlockedKeyAxm.PrivateKey)
	c.axiomAuth = authAxm
	c.axiomPrivateKey = unlockedKeyAxm.PrivateKey
	// 初始化leveldb
	leveldb, err := leveldb.New(filepath.Join(c.Config.RepoRoot, "store"))
	if err != nil {
		return fmt.Errorf("create tm-leveldb: %w", err)
	}
	c.ldb = leveldb
	c.logger = loggers.Logger(loggers.ApiServer)
	return nil
}
func (c *Client) Close() {
	c.ldb.Close()
	c.axiomClient.Close()
}
