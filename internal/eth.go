package internal

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"faucet/internal/loggers"
	"faucet/internal/repo"
	"faucet/persist"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
	"time"
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
}

type AddressData struct {
	SendTxTime int64  `json:"sendTxTime"`
	TxHash     string `json:"txHash"`
	Amount     int64  `json:"amount"`
}

func (c *Client) SendTra(net string, address string) (string, error) {
	// 合法校验：每天每个addr只发一个
	if !isValid(net, address, c.ldb) {
		return "", fmt.Errorf("surpass the faucet limit: %s", "1day1amount")
	}
	switch net {
	case "bxh":
		txHash, err := sendTxBxh(c, address, 1)
		if err != nil {
			c.ldb.Delete(construKey(persist.BxhAddressKey, address))
			return "", fmt.Errorf("txFailed: %w", err)
		}
		if err := putTxData(txHash, c, address, persist.BxhAddressKey); err != nil {
			return "", fmt.Errorf("putTxDataFailed: %w", err)
		}
		return txHash, nil
	case "erc20":
		txHash, err := sendTraEthToken(c, address, 1)
		if err != nil {
			c.ldb.Delete(construKey(persist.Erc20AddressKey, address))
			return "", fmt.Errorf("txFailed: %w", err)
		}
		if err := putTxData(txHash, c, address, persist.Erc20AddressKey); err != nil {
			return "", fmt.Errorf("putTxDataFailed: %w", err)
		}
		return txHash, nil
	default:
		return "", fmt.Errorf("invalid net: %s", net)
	}
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
	c.ldb.Put(construKey(net, address), structJSON)
	return nil
}

func isValid(net string, address string, ldb storage.Storage) bool {
	// 合法校验：每天每个addr只发一个
	switch net {
	case "bxh":
		return checkLimit(address, ldb, persist.BxhAddressKey)
	case "erc20":
		return checkLimit(address, ldb, persist.Erc20AddressKey)
	default:
		return false
	}
}

func construKey(pre string, address string) []byte {
	var buffer bytes.Buffer
	buffer.WriteString(time.Now().Format("2006-01-02"))
	buffer.WriteString("-")
	buffer.WriteString(address)
	return persist.CompositeKey(pre, buffer)
}

func checkLimit(address string, ldb storage.Storage, pre string) bool {
	data := ldb.Get(construKey(pre, address))
	if data != nil {
		//p := &AddressData{}
		//err := json.Unmarshal([]byte(data), &p)
		//if err != nil || len(p.TxHash) == 0 {
		//	return true
		//}
		return false
	} else {
		// 占位
		ldb.Put(construKey(pre, address), []byte("1"))
	}
	return true
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
