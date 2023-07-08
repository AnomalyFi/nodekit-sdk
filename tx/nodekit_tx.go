package tx

import (
	"context"
	"fmt"
	"time"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/crypto"
	"github.com/AnomalyFi/hypersdk/rpc"
	"github.com/AnomalyFi/nodekit-seq/actions"
	"github.com/AnomalyFi/nodekit-seq/auth"
	trpc "github.com/AnomalyFi/nodekit-seq/rpc"
	"github.com/AnomalyFi/nodekit-seq/utils"
	"github.com/ava-labs/avalanchego/ids"
)

const (
	//TODO fix this
	DefaultJSONRPCEndpoint = "http://127.0.0.1:9650/ext/bc/2bLP6aabd9Hju4SNnn1dsE4Q8FNrAg3N1zeWmzYFky1yDzoFVr"
)

type account struct {
	priv    crypto.PrivateKey
	factory *auth.ED25519Factory
	rsender crypto.PublicKey
	sender  string
}

// BuildAndSendTransaction builds and sends a transaction to the NodeKit Subnet with the given chain ID and transaction bytes.
func BuildAndSendTransaction(jsonRpcEndpoint string, primaryChainId string, secondaryChainID string, tx []byte) error {
	nkchainID, err := ids.FromString(primaryChainId)

	if err != nil {
		return err
	}

	cli := rpc.NewJSONRPCClient(DefaultJSONRPCEndpoint)
	tcli := trpc.NewJSONRPCClient(DefaultJSONRPCEndpoint, nkchainID)

	acc, err := CreateAccount(nkchainID, cli, tcli)

	if err != nil {
		return err
	}
	fmt.Println("Successfully Created Account")

	_, terr := BuildAndSignTx(nkchainID, acc.rsender, tx, []byte(secondaryChainID), acc.factory, cli, tcli)
	if terr != nil {
		return err
	}
	fmt.Println("Tx Finished on Hypersdk")

	return nil
}

func CreateAccount(chainID ids.ID, cli *rpc.JSONRPCClient, tcli *trpc.JSONRPCClient) (*account, error) {
	tpriv, err := crypto.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}

	trsender := tpriv.PublicKey()
	tsender := utils.Address(trsender)
	acc := &account{tpriv, auth.NewED25519Factory(tpriv), trsender, tsender}
	amount := 1
	var i64 uint64
	i64 = uint64(amount)

	priv, err := crypto.HexToKey(
		"323b1d8f4eed5f0da9da93071b034f2dce9d2d22692c172f3cb252a64ddfafd01b057de320297c29ad0c1f589ea216869cf1938d88c9fbd70d6748323dbf2fa7", //nolint:lll
	)
	factory := auth.NewED25519Factory(priv)
	parser, err := tcli.Parser(context.TODO())

	if err != nil {
		return nil, err
	}

	//TODO to implement payable by another account I just need to create a
	//new type of Auth that has the payer as the account I create with hyperlane and then create a new factory of that new auth type to sign this transaction

	submit, tx, fee, err := cli.GenerateTransaction(
		context.Background(),
		parser,
		nil,
		&actions.Transfer{
			To:    acc.rsender,
			Asset: ids.Empty,
			Value: 1000 + i64, // ensure we don't produce same tx
		},
		factory,
	)
	if err != nil {
		fmt.Errorf("It failed", err)
		return nil, err
	}

	fmt.Println(submit)
	fmt.Println(tx)
	fmt.Println(fee)

	submit(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	success, err := tcli.WaitForTransaction(ctx, tx.ID())
	cancel()
	if success == true {
		fmt.Println("SUCCESS")
	}

	return acc, nil
}

func BuildAndSignTx(chainID ids.ID, to crypto.PublicKey, data []byte, chainid []byte, factory chain.AuthFactory, cli *rpc.JSONRPCClient, tcli *trpc.JSONRPCClient) (ids.ID, error) {
	parser, err := tcli.Parser(context.TODO())
	if err != nil {
		fmt.Errorf("Parser failed", err)
		return ids.Empty, err
	}
	fmt.Println("txdata :\t %v \n", data)
	submit, tx, fee, err := cli.GenerateTransaction(
		context.Background(),
		parser,
		nil,
		&actions.SequencerMsg{
			ChainId:     chainid,
			Data:        data,
			FromAddress: to,
		},
		factory,
	)
	fmt.Println("Transaction Generated")
	if err != nil {
		fmt.Errorf("It failed", err)
		return ids.Empty, err
	}

	fmt.Println(submit)
	fmt.Println(tx)
	fmt.Println(fee)

	submit(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	success, err := tcli.WaitForTransaction(ctx, tx.ID())
	cancel()
	if success == true {
		fmt.Println("SUCCESS of Transaction")
	}
	return tx.ID(), err
}
