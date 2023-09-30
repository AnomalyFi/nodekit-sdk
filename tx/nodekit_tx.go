package tx

import (
	"context"
	"fmt"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/crypto/ed25519"
	"github.com/AnomalyFi/hypersdk/pubsub"
	"github.com/AnomalyFi/hypersdk/rpc"
	"github.com/AnomalyFi/nodekit-seq/actions"
	"github.com/AnomalyFi/nodekit-seq/auth"
	trpc "github.com/AnomalyFi/nodekit-seq/rpc"
	"github.com/AnomalyFi/nodekit-seq/utils"
	"github.com/ava-labs/avalanchego/ids"
)

// const (
// 	//TODO fix this
// 	DefaultJSONRPCEndpoint = "http://127.0.0.1:9650/ext/bc/2bLP6aabd9Hju4SNnn1dsE4Q8FNrAg3N1zeWmzYFky1yDzoFVr"
// )

type account struct {
	priv    ed25519.PrivateKey
	factory *auth.ED25519Factory
	rsender ed25519.PublicKey
	sender  string
}

// BuildAndSendTransaction builds and sends a transaction to the NodeKit Subnet with the given chain ID and transaction bytes.
func BuildAndSendTransaction(jsonRpcEndpoint string, primaryChainId string, secondaryChainID string, tx []byte) error {
	ctx := context.Background()
	nkchainID, err := ids.FromString(primaryChainId)

	if err != nil {
		return err
	}

	// cli := rpc.NewJSONRPCClient(jsonRpcEndpoint)
	// tcli := trpc.NewJSONRPCClient(jsonRpcEndpoint, nkchainID)
	rcli := rpc.NewJSONRPCClient(jsonRpcEndpoint)
	networkID, _, _, err := rcli.Network(context.TODO())

	tcli := trpc.NewJSONRPCClient(jsonRpcEndpoint, networkID, nkchainID)

	scli, err := rpc.NewWebSocketClient(
		jsonRpcEndpoint,
		rpc.DefaultHandshakeTimeout,
		pubsub.MaxPendingMessages,
		pubsub.MaxReadMessageSize,
	)

	acc, err := CreateAccount(ctx, nkchainID, rcli, scli, tcli)

	if err != nil {
		return err
	}
	fmt.Println("Successfully Created Account")

	_, terr := BuildAndSignTx(nkchainID, acc.rsender, tx, []byte(secondaryChainID), acc.factory, rcli, scli, tcli)
	if terr != nil {
		return err
	}
	fmt.Println("Tx Finished on Hypersdk")

	return nil
}

func CreateAccount(ctx context.Context, chainID ids.ID, cli *rpc.JSONRPCClient, scli *rpc.WebSocketClient, tcli *trpc.JSONRPCClient) (*account, error) {
	tpriv, err := ed25519.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}

	trsender := tpriv.PublicKey()
	tsender := utils.Address(trsender)
	acc := &account{tpriv, auth.NewED25519Factory(tpriv), trsender, tsender}
	amount := 1
	var i64 uint64
	i64 = uint64(amount)

	priv, err := ed25519.HexToKey(
		"323b1d8f4eed5f0da9da93071b034f2dce9d2d22692c172f3cb252a64ddfafd01b057de320297c29ad0c1f589ea216869cf1938d88c9fbd70d6748323dbf2fa7", //nolint:lll
	)
	factory := auth.NewED25519Factory(priv)
	parser, err := tcli.Parser(context.TODO())

	if err != nil {
		return nil, err
	}

	_, tx, _, err := cli.GenerateTransaction(
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

	if err := scli.RegisterTx(tx); err != nil {
		return false, ids.Empty, err
	}
	var res *chain.Result
	for {
		txID, dErr, result, err := scli.ListenTx(ctx)
		if dErr != nil {
			return false, ids.Empty, dErr
		}
		if err != nil {
			return false, ids.Empty, err
		}
		if txID != tx.ID() {
			continue
		}
		res = result
		break
	}

	utils.Outf("%s {{yellow}}txID:{{/}} %s\n", res.Success, tx.ID())

	return acc, nil
}

func BuildAndSignTx(chainID ids.ID, to ed25519.PublicKey, data []byte, chainid []byte, factory chain.AuthFactory, cli *rpc.JSONRPCClient, scli *rpc.WebSocketClient, tcli *trpc.JSONRPCClient) (ids.ID, error) {
	ctx := context.Background()
	parser, err := tcli.Parser(context.TODO())
	if err != nil {
		fmt.Errorf("Parser failed", err)
		return ids.Empty, err
	}
	fmt.Println("txdata :\t %v \n", data)

	_, tx, _, err := cli.GenerateTransaction(
		ctx,
		parser,
		nil,
		&actions.SequencerMsg{
			ChainId:     chainid,
			Data:        data,
			FromAddress: to,
		},
		factory,
	)
	if err != nil {
		fmt.Errorf("It failed", err)
		return nil, err
	}

	if err := scli.RegisterTx(tx); err != nil {
		return false, ids.Empty, err
	}
	var res *chain.Result
	for {
		txID, dErr, result, err := scli.ListenTx(ctx)
		if dErr != nil {
			return false, ids.Empty, dErr
		}
		if err != nil {
			return false, ids.Empty, err
		}
		if txID != tx.ID() {
			continue
		}
		res = result
		break
	}

	utils.Outf("%s {{yellow}}txID:{{/}} %s\n", res.Success, tx.ID())

	return tx.ID(), err
}
