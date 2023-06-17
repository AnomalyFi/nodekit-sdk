package tx

import (
	"context"
	"time"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/crypto"
	"github.com/AnomalyFi/hypersdk/examples/tokenvm/actions"
	"github.com/AnomalyFi/hypersdk/examples/tokenvm/auth"
	"github.com/AnomalyFi/hypersdk/examples/tokenvm/consts"
	"github.com/AnomalyFi/hypersdk/examples/tokenvm/utils"
	"github.com/AnomalyFi/hypersdk/rpc"
	"github.com/ava-labs/avalanchego/ids"
)

const (
	//TODO fix this
	nkchainID              = ids.Empty
	DefaultJSONRPCEndpoint = "127.0.0.1:9090"
)

type account struct {
	priv    crypto.PrivateKey
	factory *auth.ED25519Factory
	rsender crypto.PublicKey
	sender  string
}

// BuildAndSendTransaction builds and sends a transaction to the NodeKit Subnet with the given chain ID and transaction bytes.
func BuildAndSendTransaction(jsonRpcEndpoint string, ChainID string, tx []byte) error {
	cli := rpc.NewJSONRPCClient(DefaultJSONRPCEndpoint)
	account, err := CreateAccount(nkchainID, cli)
	if err != nil {
		return err
	}

	_, terr := BuildAndSignTx(nkchainID, account.rsender, tx, []byte(ChainID), account.factory, cli)
	if terr != nil {
		return err
	}

	return nil
}

func CreateAccount(chainID ids.ID, cli *rpc.JSONRPCClient) (*account, error) {
	//TODO need to fund these accounts
	tpriv, err := crypto.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}
	trsender := tpriv.PublicKey()
	tsender := utils.Address(trsender)
	acc := &account{tpriv, auth.NewED25519Factory(tpriv), trsender, tsender}
	amount := 1

	tx := chain.NewTx(
		&chain.Base{
			Timestamp: time.Now().Unix() + 100_000,
			ChainID:   chainID,
			UnitPrice: 1,
		},
		nil,
		&actions.Transfer{
			To:    acc.rsender,
			Value: amount,
		},
	)
	//TODO need to somehow use the root factory
	txSigned, err := tx.Sign(acc.factory, consts.ActionRegistry, consts.AuthRegistry)
	verify := txSigned.AuthAsyncVerify()
	if verify != nil {
		return nil, nil
	}
	_, err = cli.SubmitTx(context.TODO(), txSigned.Bytes())
	return acc, nil
}

func BuildAndSignTx(chainID ids.ID, to crypto.PublicKey, data []byte, chainid []byte, factory chain.AuthFactory, cli *rpc.JSONRPCClient) (ids.ID, error) {
	tx := chain.NewTx(
		&chain.Base{
			Timestamp: time.Now().Unix() + 100_000,
			ChainID:   chainID,
			UnitPrice: 1,
		},
		nil,
		&actions.SequencerMsg{
			Data:        data,
			ChainId:     chainid,
			FromAddress: to,
		},
	)
	tx, err := tx.Sign(factory, consts.ActionRegistry, consts.AuthRegistry)
	verify := tx.AuthAsyncVerify()
	if verify != nil {
		return nil, nil
	}
	_, err = cli.SubmitTx(context.TODO(), tx.Bytes())
	return tx.ID(), err
}
