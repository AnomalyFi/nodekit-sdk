package tx

//TODO this is the code I need to modify the most for the websocket implementation to work
// It will not be a server anymore but will instead just be the methods I call to start the node
import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/AnomalyFi/hypersdk/examples/tokenvm/actions"
	trpc "github.com/AnomalyFi/hypersdk/examples/tokenvm/rpc"
	"github.com/AnomalyFi/hypersdk/rpc"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/log"
)

// executionServiceServer is the implementation of the ExecutionServiceServer interface.
type ExecutionServiceServer struct {
	consensus      *catalyst.ConsensusAPI
	eth            *eth.Ethereum
	bc             *core.BlockChain
	executionState []byte
}

type DoBlockRequest struct {
	PrevStateRoot []byte   `protobuf:"bytes,1,opt,name=prev_state_root,json=prevStateRoot,proto3" json:"prev_state_root,omitempty"`
	Transactions  [][]byte `protobuf:"bytes,2,rep,name=transactions,proto3" json:"transactions,omitempty"`
	Timestamp     int64
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
					if bytes.Equal(action.ChainId, tempchainId) {
						txs = append(txs, action.Data)
					}
				}
			}
		}
		n := len(txs)
		if n > 0 {
			//TODO need to look at Block object structure in hypersdk
			s.DoBlock(context.TODO(), &DoBlockRequest{
				PrevStateRoot: s.executionState,
				Transactions:  txs,
				Timestamp:     blk.Tmstmp,
			})
		}

	}

	return nil

}

func (s *ExecutionServiceServer) DoBlock(ctx context.Context, req *DoBlockRequest) error {
	log.Info("DoBlock called request", "request", req)
	prevHeadHash := common.BytesToHash(req.PrevStateRoot)

	// The Engine API has been modified to use transactions from this mempool and abide by it's ordering.
	s.eth.TxPool().SetNodeKitOrdered(req.Transactions)

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

	// super janky but this is what the payload builder requires :/ (miner.worker.buildPayload())
	// we should probably just execute + store the block directly instead of using the engine api.
	time.Sleep(time.Second)
	payloadResp, err := s.consensus.GetPayloadV1(*fcStartResp.PayloadID)
	if err != nil {
		log.Error("failed to call GetPayloadV1", "err", err)
		return err
	}

	// call blockchain.InsertChain to actually execute and write the blocks to state
	block, err := engine.ExecutableDataToBlock(*payloadResp)
	if err != nil {
		return err
	}
	blocks := types.Blocks{
		block,
	}
	n, err := s.bc.InsertChain(blocks)
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
	newForkChoice := &engine.ForkchoiceStateV1{
		HeadBlockHash:      block.Hash(),
		SafeBlockHash:      block.Hash(),
		FinalizedBlockHash: finalizedBlock.Hash(),
	}
	fcEndResp, err := s.consensus.ForkchoiceUpdatedV1(*newForkChoice, nil)
	if err != nil {
		log.Error("failed to call ForkchoiceUpdatedV1", "err", err)
		return err
	}

	s.executionState = fcEndResp.PayloadStatus.LatestValidHash.Bytes()
	// res := &executionv1.DoBlockResponse{
	// 	BlockHash: fcEndResp.PayloadStatus.LatestValidHash.Bytes(),
	// }
	err = s.FinalizeBlock(ctx, fcEndResp.PayloadStatus.LatestValidHash.Bytes())
	if err != nil {
		log.Error("failed to Finalize Block", "err", err)
		return err
	}

	return nil
}

//TODO might be able to combine Finalize Block into DoBlock because once the block comes through WS it is completely finalized
func (s *ExecutionServiceServer) FinalizeBlock(ctx context.Context, BlockHash []byte) error {
	header := s.bc.GetHeaderByHash(common.BytesToHash(BlockHash))
	if header == nil {
		return fmt.Errorf("failed to get header for block hash 0x%x", BlockHash)
	}

	s.bc.SetFinalized(header)
	return nil
}

func (s *ExecutionServiceServer) InitState() ([]byte, error) {
	currHead := s.eth.BlockChain().CurrentHeader()

	return currHead.Hash().Bytes(), nil
}
