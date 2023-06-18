package main

import (
	"context"
	"fmt"
	"time"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/crypto"
	"github.com/AnomalyFi/hypersdk/examples/tokenvm/actions"
	"github.com/AnomalyFi/hypersdk/examples/tokenvm/auth"
	trpc "github.com/AnomalyFi/hypersdk/examples/tokenvm/rpc"
	"github.com/AnomalyFi/hypersdk/examples/tokenvm/utils"
	"github.com/AnomalyFi/hypersdk/rpc"
	"github.com/ava-labs/avalanchego/ids"
)

const (
	//TODO fix this
	DefaultJSONRPCEndpoint = "http://192.168.0.230:64204/ext/bc/2GVP5faTRBGtYDJF6VWNHUgqfzP3PgYt1JZNd5JcBzVnyUiSta"
)

type account struct {
	priv    crypto.PrivateKey
	factory *auth.ED25519Factory
	rsender crypto.PublicKey
	sender  string
}

// BuildAndSendTransaction builds and sends a transaction to the NodeKit Subnet with the given chain ID and transaction bytes.
func BuildAndSendTransaction(jsonRpcEndpoint string, ChainID string, tx []byte) error {
	nkchainID, err := ids.FromString("2GVP5faTRBGtYDJF6VWNHUgqfzP3PgYt1JZNd5JcBzVnyUiSta")

	if err != nil {
		return err
	}
	fmt.Println("here")
	cli := rpc.NewJSONRPCClient(DefaultJSONRPCEndpoint)
	tcli := trpc.NewJSONRPCClient(DefaultJSONRPCEndpoint, nkchainID)

	acc, err := CreateAccount(nkchainID, cli, tcli)
	fmt.Printf("here3\n")

	if err != nil {
		return err
	}
	fmt.Println(acc)

	_, terr := BuildAndSignTx(nkchainID, acc.rsender, tx, []byte(ChainID), acc.factory, cli, tcli)
	if terr != nil {
		return err
	}

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
	submit, tx, fee, err := cli.GenerateTransaction(
		context.Background(),
		parser,
		nil,
		&actions.SequencerMsg{
			Data:        data,
			ChainId:     chainid,
			FromAddress: to,
		},
		factory,
	)
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
		fmt.Println("SUCCESS")
	}
	return tx.ID(), err
}

func main() {
	err := BuildAndSendTransaction("http://192.168.0.230:64204/ext/bc/2GVP5faTRBGtYDJF6VWNHUgqfzP3PgYt1JZNd5JcBzVnyUiSta", "test", []byte("2"))
	if err != nil {
		panic(err)
	}

}
