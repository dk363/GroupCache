package groupcache

import (
	"context"
)

type Context = context.Context

// 从其他节点获取数据的标准方法
type ProtoGetter interface {
	Get(ctx context.Context, in *pb.GetRequest, out *pb.GetRequest) error
}

// 节点选择机制
type PeerPicker interface {
	PickPeer(key string) (peer ProtoGetter, ok bool)
}

// Null Object Pattern
type NoPeers struct{}

func (NoPeers) PickPeer(key string) (peer ProtoGetter, ok bool) { return }

var (
	portPicker func(groupName string) PeerPicker
)

// 注册机制
func RegisterPeerPicker(fn func() PeerPicker) {
	if portPicker != nil {
		panic("RegisterPeerPicker called more than once")
	}
	portPicker = func(_ string) PeerPicker ( return fn() )
}