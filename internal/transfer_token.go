package internal

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

func sendTraEthToken(c *Client, toAddr string, amount int64) (string, error) {
	c.ethLock.Lock()
	defer c.ethLock.Unlock()
	//使用合约地址
	contract, err := NewERC20(common.HexToAddress(c.Config.Ether.ContractAddress), c.ethClient)
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
	tx, err := contract.Transfer(&bind.TransactOpts{
		From:   c.ethAuth.From,
		Signer: c.ethAuth.Signer,
		Value:  nil,
	}, common.HexToAddress(toAddr), big.NewInt(amount))
	if err != nil {
		c.logger.Fatalf("TransferFrom err: %v \n", err)
		return "", err
	}
	c.logger.Infof("tx sent: %s \n", tx.Hash().Hex())
	return tx.Hash().Hex(), nil
}
