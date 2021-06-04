package internal

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
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

	value := big.NewInt(1000000000000000000 * amount) // in wei (1 eth)
	gasLimit := uint64(21000)                         // in units
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
	c.logger.Infof("tx sent: %s", signedTx.Hash().Hex())
	return signedTx.Hash().Hex(), nil
}
