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

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/sirupsen/logrus"
)

type Client struct {
	Config        *repo.Config
	ctx           context.Context
	ethClient     *ethclient.Client
	ethLock       sync.Mutex
	bxhClient     *ethclient.Client
	bxhLock       sync.Mutex
	ethAuth       *bind.TransactOpts
	bxhAuth       *bind.TransactOpts
	bxhPrivateKey *ecdsa.PrivateKey
	ldb           storage.Storage
	logger        logrus.FieldLogger
	GinContext    *gin.Context
}

type AddressData struct {
	SendTxTime int64  `json:"sendTxTime"`
	TxHash     string `json:"txHash"`
	Amount     int64  `json:"amount"`
}

func (c *Client) SendTra(net string, address string) (string, error) {
	var (
		txHash string
		err    error
	)

	// 合法校验：每天每个(public_ip + ip + addr)只发一个
	if err := c.isValid(net, address, c.ldb); err != nil {
		return "", err
	}
	switch net {
	case "bxh":
		txHash, err = sendTxBxh(c, address, 1)
	case "erc20":
		txHash, err = sendTraEthToken(c, address, 1)
	default:
		return "", fmt.Errorf("invalid net: %s", net)
	}
	if err != nil {
		deleteTxData(c, address, net)
		return "", fmt.Errorf("txFailed: %w", err)
	}
	if err := putTxData(txHash, c, address, net); err != nil {
		return "", fmt.Errorf("putTxDataFailed: %w", err)
	}
	return txHash, nil
}

func putTxData(txHash string, c *Client, address string, net string) error {
	p := &AddressData{
		SendTxTime: time.Now().UnixNano(),
		TxHash:     txHash,
		Amount:     1,
	}
	structJSON, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("json marshal failed: %w", err)
	}
	c.ldb.Put(c.construAddressKey(net, address), structJSON)
	c.ldb.Put(c.construIpKey(net), structJSON)
	return nil
}

func deleteTxData(c *Client, address string, net string) error {
	c.ldb.Delete(c.construAddressKey(net, address))
	c.ldb.Delete(c.construIpKey(net))
	return nil
}

func (c *Client) isValid(net string, address string, ldb storage.Storage) error {
	// address格式校验
	if add := types.NewAddressByStr(address); add == nil {
		return fmt.Errorf("invalid address: %s", address)
	}
	// 合法校验：每天每个addr只发一个
	return c.checkLimit(address, ldb, net)

}

func (c *Client) construAddressKey(net string, address string) []byte {
	// get public_ip and ip with address and net tobe key
	var buffer bytes.Buffer
	buffer.WriteString(time.Now().Format("2006-01-02"))
	buffer.WriteString("-")
	buffer.WriteString(address)
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

func (c *Client) checkLimit(address string, ldb storage.Storage, net string) error {
	data := ldb.Get(c.construAddressKey(net, address))
	data2 := ldb.Get(c.construIpKey(net))
	if data != nil || data2 != nil {
		//p := &AddressData{}
		//err := json.Unmarshal([]byte(data), &p)
		//if err != nil || len(p.TxHash) == 0 {
		//	return true
		//}
		return fmt.Errorf("surpass the faucet limit: %s", "1day1amount")
	} else {
		// 占位
		ldb.Put(c.construAddressKey(net, address), []byte("1"))
		ldb.Put(c.construIpKey(net), []byte("1"))
	}
	return nil
}

func (c *Client) Initialize(configPath string) error {
	cfg, err := repo.UnmarshalConfig(configPath)
	if err != nil {
		return fmt.Errorf("unmarshal config for plugin :%w", err)
	}
	c.Config = cfg
	// 构建eth+bxh客户端
	etherCli, err := ethclient.Dial(cfg.Ether.Addr)
	if err != nil {
		return fmt.Errorf("dial ethereum node: %w", err)
	}
	c.ethClient = etherCli
	bxhClient, err := ethclient.Dial(cfg.Bxh.BxhAddr)
	if err != nil {
		return fmt.Errorf("dial bxh node: %w", err)
	}
	c.bxhClient = bxhClient
	// 构建auth_eth
	keyPath := filepath.Join(configPath, cfg.Ether.KeyPath)
	keyByte, err := ioutil.ReadFile(keyPath)
	psdPath := filepath.Join(configPath, cfg.Ether.Password)
	password, err := ioutil.ReadFile(psdPath)
	unlockedKey, err := keystore.DecryptKey(keyByte, strings.TrimSpace(string(password)))
	auth := bind.NewKeyedTransactor(unlockedKey.PrivateKey)
	c.ethAuth = auth
	// 构建auth_bxh
	keyPathBxh := filepath.Join(configPath, cfg.Bxh.BxhKeyPath)
	keyByteBxh, err := ioutil.ReadFile(keyPathBxh)
	psdPathBxh := filepath.Join(configPath, cfg.Bxh.BxhPassword)
	passwordBxh, err := ioutil.ReadFile(psdPathBxh)
	unlockedKeyBxh, err := keystore.DecryptKey(keyByteBxh, strings.TrimSpace(string(passwordBxh)))
	authBxh := bind.NewKeyedTransactor(unlockedKeyBxh.PrivateKey)
	c.bxhAuth = authBxh
	c.bxhPrivateKey = unlockedKeyBxh.PrivateKey
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
	c.ethClient.Close()
	c.bxhClient.Close()
}
