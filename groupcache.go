package groupcache

import (
	"context"
	"errors"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"

	cachepolicy "example.com/gcache/cache_policy"
	"example.com/gcache/singleflight"
)

type Getter interface {
	Get(ctx context.Context, key string, dest Sink) error
}

// 函数接口 实现一类函数
type GetterFunc func(ctx context.Context, key string, dest Sink) error

func (f GetterFunc) Get(ctx context.Context, key string, dest Sink) error {
	return f(ctx, key, dest)
}

var (
	mu 					sync.RWMutex
	groups = make(map[string]*Group)

	initPeerServerOnce 	sync.Once
	initPeerServer 		func()
)

func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

func NewGroup(name string, cacheBytes int64, getter Getter, peers PeerPicker) *Group {
	return newGroup(name, cacheBytes, getter, nil)
}

func newGroup(name string, cacheBytes int64, getter Getter, peers PeerPicker) *Group {
	if getter == nil {
		panic("nil Getter")
	}

	mu.Lock()
	defer mu.Unlock()

	// 初始化对等节点服务
	initPeerServerOnce.Do(callInitPeerServer)

	if _, dup := groups[name]; dup {
		panic("duplicate registration of group " + name)
	}

	g := &Group {
		name: name,
		getter: getter,
		peers: peers,
		cacheBytes: cacheBytes,
		loadGroup: &singleflight.Group{},
	}

	if fn := newGroupHook; fn != nil {
		fn(g)
	}

	groups[name] = g
	return g
}

// Hook Function 拓展功能
var newGroupHook func(*Group)

func RegisterNewGroupHook(fn func(*Group)) {
	if newGroupHook != nil {
		panic("RegisterNewGroupHook called more than once")
	}
	newGroupHook = fn
}

func RegisterServeStart(fn func()) {
	if initPeerServer != nil {
		panic("RegisterServerStart called more than once")
	}
	initPeerServer = fn
}

func callInitPeerServer() {
	if initPeerServer != nil {
		initPeerServer()
	}
}

type Group struct {
	name 		string
	getter 		Getter
	peersOnce 	sync.Once
	peers 		PeerPicker
	cacheBytes 	int64

	mainCache 	cachepolicy.LRUCache
	hotCache	cachepolicy.LRUCache

	loadGroup 	flightGroup

	_ int32

	Stats Stats
	rand *rand.Rand
}

type flightGroup interface {
	Do(key string, fn func() (interface{}, error)) (interface{}, error)
}

type Stats struct {
	Gets           AtomicInt // 总请求数
	CacheHits      AtomicInt // 缓存命中数（Main 或 Hot）
	PeerLoads      AtomicInt // 成功从远程节点获取的次数
	PeerErrors     AtomicInt // 远程获取失败次数
	Loads          AtomicInt // 需要加载（没命中）的总次数
	LoadsDeduped   AtomicInt // 经 singleflight 去重后实际执行的加载次数
	LocalLoads     AtomicInt // 本地回源（调用 Getter）次数
	LocalLoadErrs  AtomicInt // 本地回源失败次数
	ServerRequests AtomicInt // 收到来自其他节点的请求数
}

func (g *Group) Name() string {
	return g.name
}

func (g *Group) initPeers() {
	if g.peers == nil {
		g.peers = getPeers(g.name)
	}
}

func (g *Group) Get(ctx context.Context, key string, dest Sink) error {
	g.peersOnce.Do(g.initPeers)
	g.Stats.Gets.Add(1)

	if dest == nil {
		return errors.New("Groupcache: nil dest Sink")
	}

	value, cacheHit := g.lookupCache(key)

	if cacheHit {
		g.Stats.CacheHits.Add(1)
		return setSinkView(dest, value)
	}

	destPopulated := false
	value, destPopulated, err := g.load(ctx, key, dest)
	if err != nil {
		return err
	}
	if destPopulated {
		return nil
	}

	return setSinkView(dest, value)
}

