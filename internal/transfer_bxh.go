package internal

import (
	"context"
	"github.com/ethereum/go-ethereum/common/math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func sendTxBxh(c *Client, toAddr string, amount int64) (string, error) {
	c.bxhLock.Lock()
	defer c.bxhLock.Unlock()
	client := c.bxhClient

	////余额查询
	//accountBalance, err := contract.BalanceOf(nil, common.HexToAddress("0xFDc7b0d2C02c91cB2916494076a87255051F558d"))
	//if err != nil {
	//	c.logger.Fatalf("get Balances err: %v \n", err)
	//	return "", err
	//}
	//c.logger.Infof("tx sent: %s \n", tx.Hash().Hex())

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
	return signedTx.Hash().Hex(), nil
}
