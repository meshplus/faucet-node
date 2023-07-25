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
	"github.com/sirupsen/logrus"
)

const (
	nativeToken = "native"
	erc20Token  = "erc20"
	amount      = 1
)

type Client struct {
	Config        *repo.Config
	ctx           context.Context
	ethClient     *ethclient.Client
	ethLock       sync.Mutex
	bscClient     *ethclient.Client
	bscLock       sync.Mutex
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
	// 合法校验：每天每个(net + type + addr)只发一个
	if err := c.checkLimit(net, nativeToken, address, c.ldb); err != nil {
		return "", err
	}
	txHash, err = sendTxBxh(c, address, amount)
	keyAddr := address
	if err != nil {
		deleteTxData(c, keyAddr, nativeToken, net)
		return "", fmt.Errorf("txFailed: %w", err)
	}
	if err := putTxData(txHash, c, keyAddr, nativeToken, net); err != nil {
		return "", fmt.Errorf("putTxDataFailed: %w", err)
	}
	return txHash, nil
}

func (c *Client) SendErc20(net string, address string, contractAddress string) (string, error) {
	var (
		txHash string
		err    error
	)
	// 合法校验：每天每个(net + type + addr)只发一个
	addressKey := address + "-" + contractAddress
	if err := c.checkLimit(net, erc20Token, addressKey, c.ldb); err != nil {
		return "", err
	}
	txHash, err = sendTraEthToken(c, address, erc20Token, amount)
	if err != nil {
		deleteTxData(c, addressKey, erc20Token, net)
		return "", fmt.Errorf("txFailed: %w", err)
	}
	if err := putTxData(txHash, c, addressKey, erc20Token, net); err != nil {
		return "", fmt.Errorf("putTxDataFailed: %w", err)
	}
	return txHash, nil
}

func putTxData(txHash string, c *Client, address string, typ string, net string) error {
	p := &AddressData{
		SendTxTime: time.Now().UnixNano(),
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
	buffer.WriteString("-")
	buffer.WriteString(time.Now().Format("2006-01-02"))
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
	data := ldb.Get(c.construAddressKey(net, typ, address))
	if data != nil {
		return fmt.Errorf("surpass the faucet limit: %s", "1day1amount")
	} else {
		// 占位
		ldb.Put(c.construAddressKey(net, typ, address), []byte("1"))
	}
	return nil
}

func (c *Client) Initialize(configPath string) error {
	c.ctx = context.Background()
	cfg, err := repo.UnmarshalConfig(configPath)
	if err != nil {
		return fmt.Errorf("unmarshal config for plugin :%w", err)
	}
	c.Config = cfg
	// 构建eth+bxh+bsc客户端
	etherCli, err := ethclient.Dial(cfg.Ether.Addr)
	if err != nil {
		return fmt.Errorf("dial ethereum node: %w", err)
	}
	c.ethClient = etherCli
	etherCliBSc, err := ethclient.Dial(cfg.Bsc.Addr)
	if err != nil {
		return fmt.Errorf("dial bsc node: %w", err)
	}
	c.bscClient = etherCliBSc
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
	chainID, err := etherCli.ChainID(c.ctx)
	auth, err := bind.NewKeyedTransactorWithChainID(unlockedKey.PrivateKey, chainID)
	auth.Context = c.ctx
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
