package internal

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
)

func sendTraEthToken(c *Client, toAddr string, erc20Addr string, amount int64) (string, error) {
	c.ethLock.Lock()
	defer c.ethLock.Unlock()
	//使用合约地址
	contract, err := NewERC20(common.HexToAddress(erc20Addr), c.ethClient)
	if err != nil {
		c.logger.Fatalf("conn contract: %v \n", err)
		return "", err
	}
	////余额查询
	//accountBalance, err := contract.BalanceOf(nil, common.HexToAddress("0xFDc7b0d2C02c91cB2916494076a87255051F558d"))
	//if err != nil {
	//	c.logger.Fatalf("get Balances err: %v \n", err)
	//	return "", err
	//}
	//c.logger.Infof("tx sent: %s \n", tx.Hash().Hex())

	//转账
	tx, err := contract.Transfer(c.ethAuth, common.HexToAddress(toAddr), big.NewInt(amount*math.BigPow(10, 18).Int64()))
	if err != nil {
		c.logger.Errorf("Eth TransferFrom err: %v \n", err)
		return "", err
	}
	c.logger.Infof("erc20 tx sent: %s \n", tx.Hash().Hex())
	return tx.Hash().Hex(), nil
}
