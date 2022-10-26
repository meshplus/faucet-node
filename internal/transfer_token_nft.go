package internal

import (
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
)

func mintNftToken(c *Client, toAddr string, erc20Addr string) (string, error) {
	c.ethLock.Lock()
	defer c.ethLock.Unlock()

	contract, err := NewPandaNft(common.HexToAddress(erc20Addr), c.ethClient)
	if err != nil {
		c.logger.Fatalf("conn contract: %v \n", err)
		return "", err
	}

	count, err := contract.GetCount(nil)
	countNum, err := strconv.Atoi(count.String())
	if err != nil {
		return "", err
	}

	//转账
	tx, err := contract.Mint(c.ethAuth, common.HexToAddress(toAddr), big.NewInt(int64(countNum+1)))
	if err != nil {
		c.logger.Errorf("nft TransferFrom err: %v \n", err)
		return "", err
	}
	c.logger.Infof("nft tx sent: %s \n", tx.Hash().Hex())

	return tx.Hash().Hex(), nil

}
