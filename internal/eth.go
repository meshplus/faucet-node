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
	"math/big"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/backoff"
	"github.com/Rican7/retry/strategy"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	types2 "github.com/ethereum/go-ethereum/core/types"
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
	bscClient     *ethclient.Client
	bscLock       sync.Mutex
	bxhClient     *ethclient.Client
	bxhLock       sync.Mutex
	ethAuth       *bind.TransactOpts
	bscAuth       *bind.TransactOpts
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

func (c *Client) SendTra(net string, address string, erc20Addr string) (string, error) {
	var (
		txHash string
		err    error
	)

	// 合法校验：每天每个(public_ip + ip + addr)只发一个
	/*if err := c.isValid(net, address, c.ldb, erc20Addr); err != nil {
		return "", err
	}*/
	switch net {
	case "bxh":
		txHash, err = sendTxBxh(c, address, 1)
	case "erc20":
		txHash, err = sendTraEthToken(c, address, erc20Addr, 1)
	case "bsc":
		txHash, err = sendTraBscToken(c, address, erc20Addr, 1)
	case "nft":
		txHash, err = mintNftToken(c, address, erc20Addr)
	default:
		return "", fmt.Errorf("invalid net: %s", net)
	}
	keyAddr := address + erc20Addr
	if err != nil {
		deleteTxData(c, keyAddr, net)
		return "", fmt.Errorf("txFailed: %w", err)
	}
	if err := putTxData(txHash, c, keyAddr, net); err != nil {
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
	return nil
}

func deleteTxData(c *Client, address string, net string) error {
	c.ldb.Delete(c.construAddressKey(net, address))
	return nil
}

func (c *Client) isValid(net string, address string, ldb storage.Storage, erc20Addr string) error {
	// address格式校验xx
	if add := types.NewAddressByStr(address); add == nil {
		return fmt.Errorf("invalid address: %s", address)
	}
	if !strings.EqualFold("bxh", net) && types.NewAddressByStr(erc20Addr) == nil {
		return fmt.Errorf("invalid erc20Addr: %s", erc20Addr)
	}
	// 合法校验：每天每个addr只发一个
	keyAddr := address + erc20Addr
	return c.checkLimit(keyAddr, ldb, net)

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
	if data != nil {
		//p := &AddressData{}
		//err := json.Unmarshal([]byte(data), &p)
		//if err != nil || len(p.TxHash) == 0 {
		//	return true
		//}
		return fmt.Errorf("surpass the faucet limit: %s", "1day1amount")
	} else {
		// 占位
		ldb.Put(c.construAddressKey(net, address), []byte("1"))
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
	auth.GasLimit = 100000
	auth.GasPrice = big.NewInt(50000)
	c.ethAuth = auth
	// 构建auth_bsc
	/*chainIDBsc, err := etherCliBSc.ChainID(c.ctx)
	authBsc, err := bind.NewKeyedTransactorWithChainID(unlockedKey.PrivateKey, chainIDBsc)
	authBsc.Context = c.ctx
	authBsc.GasFeeCap = nil
	authBsc.GasTipCap = nil
	authBsc.GasPrice, _ = c.bscClient.SuggestGasPrice(c.ctx)
	c.bscAuth = authBsc*/

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

func GetRecept(c *ethclient.Client, txHash common.Hash) error {
	var err error
	var receipt *types2.Receipt
	err = retry.Retry(func(attempt uint) error {
		receipt, err = c.TransactionReceipt(context.Background(), txHash)
		if err != nil {
			return err
		}
		return nil
	}, strategy.Limit(5), strategy.Backoff(backoff.Fibonacci(500*time.Millisecond)))
	if err != nil {
		return err
	}
	if receipt.Status != types2.ReceiptStatusSuccessful {
		return fmt.Errorf("交易执行错误")
	}
	return nil
}
