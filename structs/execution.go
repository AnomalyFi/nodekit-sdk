package structs

import (
	context "context"

	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type DoBlockRequest struct {
	PrevStateRoot []byte   `protobuf:"bytes,1,opt,name=prev_state_root,json=prevStateRoot,proto3" json:"prev_state_root,omitempty"`
	Transactions  [][]byte `protobuf:"bytes,2,rep,name=transactions,proto3" json:"transactions,omitempty"`
	Timestamp     int64
}

// ExecutionServiceServer is the server API for ExecutionService service.
// All implementations must embed UnimplementedExecutionServiceServer
// for forward compatibility
type ExecutionServiceServer interface {
	InitState() ([]byte, error)
	DoBlock(context.Context, *DoBlockRequest) error
	FinalizeBlock(context.Context, []byte) error
}

type UnimplementedExecutionServiceServer struct {
}

func (UnimplementedExecutionServiceServer) InitState() ([]byte, error) {
	return nil, status.Errorf(codes.Unimplemented, "method InitState not implemented")
}
func (UnimplementedExecutionServiceServer) DoBlock(context.Context, *DoBlockRequest) error {
	return status.Errorf(codes.Unimplemented, "method DoBlock not implemented")
}
func (UnimplementedExecutionServiceServer) FinalizeBlock(context.Context, []byte) error {
	return status.Errorf(codes.Unimplemented, "method FinalizeBlock not implemented")
}
