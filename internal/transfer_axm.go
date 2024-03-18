package internal

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/axiomesh/faucet/global"
)

func sendTxAxm(c *Client, toAddr string, amount float64) (string, error) {
	c.axiomLock.Lock()
	defer c.axiomLock.Unlock()
	client := c.axiomClient

	fromAddress := c.axiomAuth.From
	// 余额查询
	balanceNow, err := client.BalanceAt(context.Background(), common.HexToAddress(toAddr), nil)
	if err != nil {
		c.logger.Error(err)
		return "", err
	}
	limit := floatToEtherBigInt(c.Config.Axiom.ClaimLimit)
	if balanceNow.Cmp(limit) >= 0 {
		return "", fmt.Errorf(global.EnoughTokenMsg)
	}

	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		c.logger.Error(err)
		return "", err
	}

	value := floatToEtherBigInt(amount)         // in wei (1 eth)
	gasLimit := uint64(c.Config.Axiom.GasLimit) // in units
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

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), c.axiomPrivateKey)
	if err != nil {
		c.logger.Error(err)
		return "", err
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", err
	}
	c.logger.Infof("axm tx sent: %s", signedTx.Hash().Hex())

	return signedTx.Hash().Hex(), nil
}

func checkBalance(c *Client, toAddr string) (bool, error) {
	client := c.axiomClient
	// 余额查询
	balanceNow, err := client.BalanceAt(context.Background(), common.HexToAddress(toAddr), nil)
	if err != nil {
		c.logger.Error(err)
		return false, err
	}
	limit := floatToEtherBigInt(c.Config.Axiom.ClaimLimit)
	if balanceNow.Cmp(limit) >= 0 {
		return false, fmt.Errorf(global.EnoughTokenMsg)
	}
	return true, nil
}

func floatToEtherBigInt(value float64) *big.Int {
	decimalMultiplier := new(big.Int)
	decimalMultiplier.Exp(big.NewInt(10), big.NewInt(18), nil)

	valueAsBigFloat := new(big.Float).SetFloat64(value)
	valueAsBigFloat.Mul(valueAsBigFloat, new(big.Float).SetInt(decimalMultiplier))

	etherBigInt := new(big.Int)
	valueAsBigFloat.Int(etherBigInt)

	return etherBigInt
}
