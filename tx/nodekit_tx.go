// package tx

// import (
// 	"context"
// 	"time"

// 	"github.com/AnomalyFi/hypersdk/chain"
// 	"github.com/AnomalyFi/hypersdk/crypto"
// 	"github.com/AnomalyFi/hypersdk/examples/tokenvm/actions"
// 	"github.com/AnomalyFi/hypersdk/examples/tokenvm/auth"
// 	"github.com/AnomalyFi/hypersdk/examples/tokenvm/consts"
// 	"github.com/AnomalyFi/hypersdk/examples/tokenvm/utils"
// 	"github.com/AnomalyFi/hypersdk/rpc"
// 	"github.com/ava-labs/avalanchego/ids"
// )

// const (
// 	//TODO fix this
// 	nkchainID              = ids.FromString("qPKuaWFeaFytxhh6sxV4DtsAsWoJZrWsKWfWi5wWayvNSbfV2")
// 	DefaultJSONRPCEndpoint = "http://192.168.0.230:26869/ext/bc/qPKuaWFeaFytxhh6sxV4DtsAsWoJZrWsKWfWi5wWayvNSbfV2"
// )

// type account struct {
// 	priv    crypto.PrivateKey
// 	factory *auth.ED25519Factory
// 	rsender crypto.PublicKey
// 	sender  string
// }

// // BuildAndSendTransaction builds and sends a transaction to the NodeKit Subnet with the given chain ID and transaction bytes.
// func BuildAndSendTransaction(jsonRpcEndpoint string, ChainID string, tx []byte) error {
// 	cli := rpc.NewJSONRPCClient(DefaultJSONRPCEndpoint)
// 	account, err := CreateAccount(nkchainID, cli)
// 	if err != nil {
// 		return err
// 	}

// 	_, terr := BuildAndSignTx(nkchainID, account.rsender, tx, []byte(ChainID), account.factory, cli)
// 	if terr != nil {
// 		return err
// 	}

// 	return nil
// }

// func CreateAccount(chainID ids.ID, cli *rpc.JSONRPCClient) (*account, error) {
// 	//TODO need to fund these accounts
// 	tpriv, err := crypto.GeneratePrivateKey()
// 	if err != nil {
// 		return nil, err
// 	}
// 	trsender := tpriv.PublicKey()
// 	tsender := utils.Address(trsender)
// 	acc := &account{tpriv, auth.NewED25519Factory(tpriv), trsender, tsender}
// 	amount := 1
// 	var i64 uint64
// 	i64 = uint64(amount)

// 	tx := chain.NewTx(
// 		&chain.Base{
// 			Timestamp: time.Now().Unix() + 100_000,
// 			ChainID:   chainID,
// 			UnitPrice: 1,
// 		},
// 		nil,
// 		&actions.Transfer{
// 			To:    acc.rsender,
// 			Value: i64,
// 		},
// 	)
// 	priv, err := crypto.HexToKey(
// 		"323b1d8f4eed5f0da9da93071b034f2dce9d2d22692c172f3cb252a64ddfafd01b057de320297c29ad0c1f589ea216869cf1938d88c9fbd70d6748323dbf2fa7", //nolint:lll
// 	)
// 	factory := auth.NewED25519Factory(priv)
// 	txSigned, err := tx.Sign(factory, consts.ActionRegistry, consts.AuthRegistry)
// 	verify := txSigned.AuthAsyncVerify()
// 	if verify != nil {
// 		return nil, nil
// 	}
// 	_, err = cli.SubmitTx(context.TODO(), txSigned.Bytes())
// 	return acc, nil
// }

// func BuildAndSignTx(chainID ids.ID, to crypto.PublicKey, data []byte, chainid []byte, factory chain.AuthFactory, cli *rpc.JSONRPCClient) (ids.ID, error) {
// 	tx := chain.NewTx(
// 		&chain.Base{
// 			Timestamp: time.Now().Unix() + 100_000,
// 			ChainID:   chainID,
// 			UnitPrice: 1,
// 		},
// 		nil,
// 		&actions.SequencerMsg{
// 			Data:        data,
// 			ChainId:     chainid,
// 			FromAddress: to,
// 		},
// 	)
// 	tx, err := tx.Sign(factory, consts.ActionRegistry, consts.AuthRegistry)
// 	verify := tx.AuthAsyncVerify()
// 	if verify != nil {
// 		return nil, nil
// 	}
// 	_, err = cli.SubmitTx(context.TODO(), tx.Bytes())
// 	return tx.ID(), err
// }

// func main() {
// 	err := BuildAndSendTransaction("http://192.168.0.230:26869/ext/bc/qPKuaWFeaFytxhh6sxV4DtsAsWoJZrWsKWfWi5wWayvNSbfV2", "test", []byte("2"))
// 	if err != nil {
// 		panic(err)
// 	}

// }
