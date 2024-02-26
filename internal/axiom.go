package internal

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"faucet/global"
	"faucet/internal/utils"
	"faucet/persist"
	"faucet/pkg/loggers"
	"faucet/pkg/repo"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/backoff"
	"github.com/Rican7/retry/strategy"
	"github.com/axiomesh/axiom-kit/storage"
	"github.com/axiomesh/axiom-kit/storage/leveldb"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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
	preLockCheck    sync.Mutex
}

type AddressData struct {
	SendTxTime int64   `json:"sendTxTime"`
	TxHash     string  `json:"txHash"`
	Amount     float64 `json:"amount"`
}

func (c *Client) SendTra(net string, address string, amount float64, tweetUrl string) (string, int, error) {
	var (
		txHash string
		err    error
	)
	lowerAddress := strings.ToLower(address)
	// 合法校验：每天每个(net + type + addr)只发一个
	if err := c.checkLimit(net, global.NativeToken, lowerAddress, c.ldb); err != nil {
		if err.Error() == global.AddrPreLockErrMsg {
			return "", global.AddrPreLockErrCode, err
		} else {
			return "", global.ReqWithinDayCode, err
		}
	}

	if tweetUrl != "" {
		code, msg := c.TweetReqCheck(tweetUrl, address)
		if code != global.SUCCESS {
			return "", code, fmt.Errorf(msg)
		}
	}

	txHash, err = sendTxAxm(c, address, amount)
	if err != nil {
		if err.Error() == global.EnoughTokenMsg {
			return "", global.EnoughTokenCode, err
		}
		matched, matchErr := regexp.MatchString("insufficient funds", err.Error())
		if matchErr != nil {
			return "", global.BlockChainCode, fmt.Errorf(global.BlockChainMsg)
		}
		if matched {
			return "", global.InsufficientCode, fmt.Errorf(global.InsufficientMsg)
		}
		return "", global.CommonErrCode, fmt.Errorf(global.CommonErrMsg)
	}
	if checkTxSuccess(c, txHash) {
		if err := putTxData(txHash, c, lowerAddress, global.NativeToken, net); err != nil {
			return "", global.CommonErrCode, fmt.Errorf(global.CommonErrMsg)
		}
	}
	return txHash, global.SUCCESS, nil
}

func (c *Client) PreCheck(net string, address string) (int, error) {
	lowerAddress := strings.ToLower(address)
	// 合法校验：每天每个(net + type + addr)只发一个
	if err := c.precheckLimit(net, global.NativeToken, lowerAddress, c.ldb); err != nil {
		if err.Error() == global.AddrPreLockErrMsg {
			return global.AddrPreLockErrCode, err
		} else {
			return global.ReqWithinDayCode, err
		}
	}
	judge, err := checkBalance(c, address)
	if err != nil && !judge {
		if err.Error() == global.EnoughTokenMsg {
			return global.EnoughTokenCode, err
		} else {
			return global.CommonErrCode, fmt.Errorf(global.CommonErrMsg)
		}
	}
	return global.SUCCESS, nil

}

func putTxData(txHash string, c *Client, address string, typ string, net string) error {
	p := &AddressData{
		SendTxTime: time.Now().Unix(),
		TxHash:     txHash,
		Amount:     c.Config.Axiom.Amount,
	}
	structJSON, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("json marshal failed: %w", err)
	}
	c.ldb.Put(c.construAddressKey(net, typ, address), structJSON)
	return nil
}

func DeleteTxData(c *Client, address string, typ string, net string) error {
	c.ldb.Delete(c.construPreLockAddressKey(net, typ, address))
	return nil
}

func (c *Client) construAddressKey(net string, typ string, address string) []byte {
	// get public_ip and ip with address and net tobe key
	var buffer bytes.Buffer
	buffer.WriteString(address)
	buffer.WriteString("-")
	buffer.WriteString(typ)
	return persist.CompositeKey(net, buffer)
}

func (c *Client) construPreLockAddressKey(net string, typ string, address string) []byte {
	// get public_ip and ip with address and net tobe key
	var buffer bytes.Buffer
	buffer.WriteString(time.Now().Format("2006-01-02"))
	buffer.WriteString("-")
	buffer.WriteString(address)
	buffer.WriteString("-")
	buffer.WriteString(typ)
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
	return persist.CompositeKey(net, buffer)
}

func (c *Client) checkLimit(net string, typ string, address string, ldb storage.Storage) error {
	c.preLockCheck.Lock()
	defer c.preLockCheck.Unlock()
	valuePreLockData := ldb.Get(c.construPreLockAddressKey(net, typ, address))
	if valuePreLockData != nil {
		return fmt.Errorf(global.AddrPreLockErrMsg)
	}
	c.ldb.Put(c.construPreLockAddressKey(net, global.NativeToken, address), []byte("preLock"))
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
		if timeDifference <= oneDayInSeconds {
			return fmt.Errorf(global.ReqWithinDayMsg)
		}
	}
	return nil
}

func (c *Client) precheckLimit(net string, typ string, address string, ldb storage.Storage) error {
	valuePreLockData := ldb.Get(c.construPreLockAddressKey(net, typ, address))
	if valuePreLockData != nil {
		return fmt.Errorf(global.AddrPreLockErrMsg)
	}
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
		if timeDifference <= oneDayInSeconds {
			return fmt.Errorf(global.ReqWithinDayMsg)
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

func (c *Client) Initialize(cfg *repo.Config, configPath string) error {
	c.ctx = context.Background()
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
	if err != nil {
		return err
	}
	private := strings.TrimSpace(string(keyByteAxm))
	privateKeyBytes, err := hex.DecodeString(private)
	if err != nil {
		return fmt.Errorf("Error decoding private key hex:", err)
	}
	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return fmt.Errorf("Error converting to ECDSA private key:", err)
	}
	c.axiomPrivateKey = privateKey
	authAxm := bind.NewKeyedTransactor(privateKey)
	c.axiomAuth = authAxm

	// 初始化leveldb
	leveldb, err := leveldb.New(filepath.Join(configPath, "store"), nil)
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
