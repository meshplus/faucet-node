package internal

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/backoff"
	"github.com/Rican7/retry/strategy"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
)

func sendTxBxh(c *Client, toAddr string, amount int64) (string, error) {
	c.bxhLock.Lock()
	defer c.bxhLock.Unlock()
	client := c.bxhClient

	fromAddress := c.bxhAuth.From
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		c.logger.Error(err)
		return "", err
	}

	value := big.NewInt(math.BigPow(10, 18).Int64() * amount) // in wei (1 eth)
	gasLimit := uint64(21000)                                 // in units
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		c.logger.Error(err)
		return "", err
	}
	toAddress := common.HexToAddress(toAddr)
	var data []byte
	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, data)

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		c.logger.Error(err)
		return "", err
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), c.bxhPrivateKey)
	if err != nil {
		c.logger.Error(err)
		return "", err
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		c.logger.Error(err)
		return "", err
	}
	c.logger.Infof("bxh tx sent: %s", signedTx.Hash().Hex())

	err = retry.Retry(func(attempt uint) error {
		receipt, err := client.TransactionReceipt(context.Background(), signedTx.Hash())
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

	if err != nil && err.Error() == "faucet transfer failed" {
		return "", err
	}
	return signedTx.Hash().Hex(), nil
}
