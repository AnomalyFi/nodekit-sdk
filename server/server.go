package server

//TODO this is the code I need to modify the most for the websocket implementation to work
// It will not be a server anymore but will instead just be the methods I call to start the node
import (
	"bytes"
	"context"
	"fmt"
	"time"

	executionv1 "github.com/AnomalyFi/nodekit-sdk/structs"

	"github.com/AnomalyFi/hypersdk/examples/tokenvm/actions"
	trpc "github.com/AnomalyFi/hypersdk/examples/tokenvm/rpc"
	"github.com/AnomalyFi/hypersdk/rpc"
	"github.com/ava-labs/avalanchego/ids"

	"github.com/AnomalyFi/go-ethereum/beacon/engine"
	"github.com/AnomalyFi/go-ethereum/common"
	"github.com/AnomalyFi/go-ethereum/core"
	"github.com/AnomalyFi/go-ethereum/core/types"
	"github.com/AnomalyFi/go-ethereum/eth"
	"github.com/AnomalyFi/go-ethereum/eth/catalyst"
	"github.com/AnomalyFi/go-ethereum/log"
)

// executionServiceServer is the implementation of the ExecutionServiceServer interface.
type ExecutionServiceServer struct {
	executionv1.UnimplementedExecutionServiceServer

	consensus      *catalyst.ConsensusAPI
	eth            *eth.Ethereum
	bc             *core.BlockChain
	executionState []byte
}

func NewExecutionServiceServer(eth *eth.Ethereum) *ExecutionServiceServer {
	consensus := catalyst.NewConsensusAPI(eth)

	bc := eth.BlockChain()

	currHead := eth.BlockChain().CurrentHeader()

	return &ExecutionServiceServer{
		eth:            eth,
		consensus:      consensus,
		bc:             bc,
		executionState: currHead.Hash().Bytes(),
	}
}

func (s *ExecutionServiceServer) WSBlock(JSONRPCEndpoint string, chainID ids.ID, ctx context.Context, websocketClient *rpc.WebSocketClient) error {
	executionState, err := s.InitState()
	s.executionState = executionState

	fmt.Println("Execution State Completed")

	cli := trpc.NewJSONRPCClient(JSONRPCEndpoint, chainID)
	if err := websocketClient.RegisterBlocks(); err != nil {
		return err
	}

	parser, err := cli.Parser(ctx)

	tempchainId := []byte("ethereum")

	if err != nil {
		return err
	}
	for ctx.Err() == nil {
		blk, results, err := websocketClient.ListenBlock(ctx, parser)
		if err != nil {
			return err
		}
		var txs [][]byte
		//TODO need to decode all the messages here instead and bundle the TXDatas into a list of Bytes
		for i, tx := range blk.Txs {
			result := results[i]
			if result.Success {
				switch action := tx.Action.(type) {
				case *actions.SequencerMsg:
					//TODO this should add the relevant transactions from a block and then call DoBlock to execute them.
					fmt.Println(action.ChainId)
					fmt.Println(tempchainId)
					if bytes.Equal(action.ChainId, tempchainId) {
						fmt.Println("FOUND TRANSACTIONS FROM HYPERSDK")
						txs = append(txs, action.Data)
					}
				}
			}
		}
		fmt.Println("GOT TRANSACTIONS FROM HYPERSDK")

		n := len(txs)
		if n > 0 {
			fmt.Println("Submitted TRANSACTIONS FROM HYPERSDK")
			//TODO need to look at Block object structure in hypersdk
			err = s.DoBlock(context.TODO(), &executionv1.DoBlockRequest{
				PrevStateRoot: s.executionState,
				Transactions:  txs,
				Timestamp:     blk.Tmstmp,
			})
			if err != nil {
				log.Error("failed to DoBlock", "err", err)
				return err
			}
		}

	}

	return nil

}

func (s *ExecutionServiceServer) DoBlock(ctx context.Context, req *executionv1.DoBlockRequest) error {
	log.Info("DoBlock called request", "request", req)
	prevHeadHash := common.BytesToHash(req.PrevStateRoot)

	// The Engine API has been modified to use transactions from this mempool and abide by it's ordering.
	s.eth.TxPool().SetNodeKitOrdered(req.Transactions)

	log.Info("DoBlock ordered Transactions", "transactions", req.Transactions)

	// Do the whole Engine API in a single loop
	startForkChoice := &engine.ForkchoiceStateV1{
		HeadBlockHash:      prevHeadHash,
		SafeBlockHash:      prevHeadHash,
		FinalizedBlockHash: prevHeadHash,
	}
	payloadAttributes := &engine.PayloadAttributes{
		Timestamp:             uint64(req.Timestamp),
		Random:                common.Hash{},
		SuggestedFeeRecipient: common.Address{},
	}
	fcStartResp, err := s.consensus.ForkchoiceUpdatedV1(*startForkChoice, payloadAttributes)
	if err != nil {
		return err
	}

	log.Info("DoBlock ForkChoice Updated", "request", req)

	// super janky but this is what the payload builder requires :/ (miner.worker.buildPayload())
	// we should probably just execute + store the block directly instead of using the engine api.
	time.Sleep(time.Second)
	payloadResp, err := s.consensus.GetPayloadV1(*fcStartResp.PayloadID)
	if err != nil {
		log.Error("failed to call GetPayloadV1", "err", err)
		return err
	}
	log.Info("DoBlock called GetPayloadV1", "request", req)

	// call blockchain.InsertChain to actually execute and write the blocks to state
	block, err := engine.ExecutableDataToBlock(*payloadResp)
	if err != nil {
		return err
	}
	log.Info("DoBlock called ExecutableDataToBlock", "request", req)

	blocks := types.Blocks{
		block,
	}
	n, err := s.bc.InsertChain(blocks)
	log.Info("DoBlock called InsertChain")
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("failed to insert block into blockchain (n=%d)", n)
	}

	// remove txs from original mempool
	for _, tx := range block.Transactions() {
		s.eth.TxPool().RemoveTx(tx.Hash())
	}

	finalizedBlock := s.bc.CurrentFinalBlock()

	log.Info("DoBlock called CurrentFinalBlock", "request", req)

	newForkChoice := &engine.ForkchoiceStateV1{
		HeadBlockHash:      block.Hash(),
		SafeBlockHash:      block.Hash(),
		FinalizedBlockHash: finalizedBlock.Hash(),
	}
	fcEndResp, err := s.consensus.ForkchoiceUpdatedV1(*newForkChoice, nil)

	log.Info("DoBlock called ForkchoiceUpdatedV1 again", "request", req)

	if err != nil {
		log.Error("failed to call ForkchoiceUpdatedV1", "err", err)
		return err
	}

	s.executionState = fcEndResp.PayloadStatus.LatestValidHash.Bytes()

	err = s.FinalizeBlock(ctx, fcEndResp.PayloadStatus.LatestValidHash.Bytes())
	if err != nil {
		log.Error("failed to Finalize Block", "err", err)
		return err
	}

	return nil
}

func (s *ExecutionServiceServer) FinalizeBlock(ctx context.Context, BlockHash []byte) error {
	log.Info("Got to Finalize Block")
	header := s.bc.GetHeaderByHash(common.BytesToHash(BlockHash))
	if header == nil {
		return fmt.Errorf("failed to get header for block hash 0x%x", BlockHash)
	}

	s.bc.SetFinalized(header)
	log.Info("Finalized Block")

	return nil
}

func (s *ExecutionServiceServer) InitState() ([]byte, error) {
	currHead := s.eth.BlockChain().CurrentHeader()

	return currHead.Hash().Bytes(), nil
}
