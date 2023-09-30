package main

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
func BuildAndSendTransaction(jsonRpcEndpoint string, ChainID string, tx []byte) error {
	nkchainID, err := ids.FromString("2bLP6aabd9Hju4SNnn1dsE4Q8FNrAg3N1zeWmzYFky1yDzoFVr")

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
	fmt.Println("No errors")

	_, terr := BuildAndSignTx(nkchainID, acc.rsender, tx, []byte(ChainID), acc.factory, cli, tcli)
	if terr != nil {
		return err
	}
	fmt.Println("TX FINISHED ON HYPERSDK")

	return nil
}

func CreateAccount(chainID ids.ID, cli *rpc.JSONRPCClient, tcli *trpc.JSONRPCClient) (*account, error) {
	tpriv, err := crypto.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}
	fmt.Println("No errors")

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
	fmt.Println("PARSER WORKED")
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
	fmt.Println("Transaction WORKED")
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

func main() {
	byteArray := []byte{2, 248, 117, 130, 5, 57, 8, 132, 89, 104, 47, 0, 133, 4, 90, 118, 125, 113, 130, 82, 8, 148, 228, 159, 63, 227, 110, 159, 13, 235, 100, 138, 112, 128, 80, 21, 170, 137, 148, 2, 255, 248, 136, 13, 224, 182, 179, 167, 100, 0, 0, 128, 192, 1, 160, 187, 97, 123, 192, 59, 150, 57, 70, 19, 129, 28, 78, 191, 80, 67, 53, 195, 22, 137, 141, 48, 197, 124, 151, 245, 227, 197, 130, 253, 2, 109, 6, 160, 21, 40, 87, 45, 214, 180, 231, 224, 205, 62, 27, 172, 72, 18, 62, 31, 237, 121, 68, 25, 4, 231, 13, 193, 104, 89, 15, 107, 13, 35, 172, 216}
	err := BuildAndSendTransaction("http://127.0.0.1:9650/ext/bc/2bLP6aabd9Hju4SNnn1dsE4Q8FNrAg3N1zeWmzYFky1yDzoFVr", "test", byteArray)
	if err != nil {
		panic(err)
	}

}
