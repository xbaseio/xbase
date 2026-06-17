package gate

import (
	"context"
	"maps"
	"runtime"
	"time"

	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/etc"
	"github.com/xbaseio/xbase/locate"
	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/network"
	"github.com/xbaseio/xbase/registry"
	"github.com/xbaseio/xbase/utils/xjwt"
	"github.com/xbaseio/xbase/utils/xuuid"
)

const (
	defaultName           = "gate"
	defaultAddr           = ":0"
	defaultTimeout        = 3 * time.Second
	defaultDispatch       = cluster.Random
	defaultVersion        = "1"
	defaultRetireDelay    = 10 * time.Minute
	defaultReceiveQueue   = 8192
	defaultLoginMessageID = 1000
	defaultLobbyGameID    = 0
	defaultJWTIdentityKey = "uid"
)

const (
	defaultIDKey               = "etc.cluster.gate.id"
	defaultNameKey             = "etc.cluster.gate.name"
	defaultAddrKey             = "etc.cluster.gate.addr"
	defaultExposeKey           = "etc.cluster.gate.expose"
	defaultTimeoutKey          = "etc.cluster.gate.timeout"
	defaultDispatchKey         = "etc.cluster.gate.dispatch"
	defaultMetadataKey         = "etc.cluster.gate.metadata"
	defaultVersionKey          = "etc.cluster.gate.version"
	defaultRetireDelayKey      = "etc.cluster.gate.retireDelay"
	defaultReceiveQueueKey     = "etc.cluster.gate.receiveQueue"
	defaultDeliverWorkersKey   = "etc.cluster.gate.deliverWorkers"
	defaultLoginMessageIDKey   = "etc.cluster.gate.login.messageID"
	defaultLobbyGameIDKey      = "etc.cluster.gate.login.lobbyGameID"
	defaultJWTSecretKey        = "etc.cluster.gate.jwt.secretKey"
	defaultJWTIdentityKeyKey   = "etc.cluster.gate.jwt.identityKey"
	defaultJWTSignAlgorithmKey = "etc.cluster.gate.jwt.signAlgorithm"
	defaultTestWhitelistKey    = "etc.cluster.gate.testWhitelist"
)

type Option func(o *options)

type options struct {
	ctx              context.Context
	id               string
	name             string
	addr             string
	expose           bool
	timeout          time.Duration
	server           network.Server
	locator          locate.Locator
	registry         registry.Registry
	dispatch         cluster.Dispatch
	nodeKind         cluster.NodeKind
	gameID           int32
	version          string
	retireDelay      time.Duration
	receiveQueue     int
	deliverWorkers   int
	loginMessageID   int32
	lobbyGameID      int32
	jwt              *xjwt.JWT
	jwtSecretKey     string
	jwtIdentityKey   string
	jwtSignAlgorithm xjwt.SignAlgorithm
	metadata         map[string]string
	testWhitelist    map[int64]struct{}
	testUserChecker  func(uid int64) bool
}

func defaultOptions() *options {
	opts := &options{
		ctx:              context.Background(),
		name:             defaultName,
		addr:             defaultAddr,
		timeout:          defaultTimeout,
		dispatch:         defaultDispatch,
		metadata:         make(map[string]string),
		expose:           etc.Get(defaultExposeKey).Bool(),
		nodeKind:         cluster.Node_Normal,
		gameID:           -1,
		version:          defaultVersion,
		retireDelay:      defaultRetireDelay,
		receiveQueue:     defaultReceiveQueue,
		deliverWorkers:   max(runtime.NumCPU(), 1),
		loginMessageID:   defaultLoginMessageID,
		lobbyGameID:      defaultLobbyGameID,
		jwtIdentityKey:   defaultJWTIdentityKey,
		jwtSignAlgorithm: xjwt.HS256,
		testWhitelist:    make(map[int64]struct{}),
	}

	if id := etc.Get(defaultIDKey).String(); id != "" {
		opts.id = id
	} else {
		opts.id = xuuid.UUID()
	}

	if name := etc.Get(defaultNameKey).String(); name != "" {
		opts.name = name
	}

	if addr := etc.Get(defaultAddrKey).String(); addr != "" {
		opts.addr = addr
	}

	if timeout := etc.Get(defaultTimeoutKey).Duration(); timeout > 0 {
		opts.timeout = timeout
	}

	if strategy := etc.Get(defaultDispatchKey).String(); strategy != "" {
		opts.dispatch = cluster.Dispatch(strategy)
	}

	if err := etc.Get(defaultMetadataKey).Scan(&opts.metadata); err != nil {
		log.Warnf("scan gate metadata failed: %v", err)
	}

	if version := etc.Get(defaultVersionKey).String(); version != "" {
		opts.version = version
	}

	if delay := etc.Get(defaultRetireDelayKey).Duration(); delay > 0 {
		opts.retireDelay = delay
	}

	if n := etc.Get(defaultReceiveQueueKey).Int(); n > 0 {
		opts.receiveQueue = n
	}

	if n := etc.Get(defaultDeliverWorkersKey).Int(); n > 0 {
		opts.deliverWorkers = n
	}

	if etc.Has(defaultLoginMessageIDKey) {
		opts.loginMessageID = int32(etc.Get(defaultLoginMessageIDKey).Int())
	}

	if etc.Has(defaultLobbyGameIDKey) {
		opts.lobbyGameID = int32(etc.Get(defaultLobbyGameIDKey).Int())
	}

	opts.jwtSecretKey = etc.Get(defaultJWTSecretKey).String()

	if identityKey := etc.Get(defaultJWTIdentityKeyKey).String(); identityKey != "" {
		opts.jwtIdentityKey = identityKey
	}

	if signAlgorithm := etc.Get(defaultJWTSignAlgorithmKey).String(); signAlgorithm != "" {
		opts.jwtSignAlgorithm = xjwt.SignAlgorithm(signAlgorithm)
	}

	for _, uid := range etc.Get(defaultTestWhitelistKey).Int64s() {
		if uid > 0 {
			opts.testWhitelist[uid] = struct{}{}
		}
	}

	return opts
}

func WithID(id string) Option                   { return func(o *options) { o.id = id } }
func WithName(name string) Option               { return func(o *options) { o.name = name } }
func WithAddr(addr string) Option               { return func(o *options) { o.addr = addr } }
func WithExpose(expose bool) Option             { return func(o *options) { o.expose = expose } }
func WithContext(ctx context.Context) Option    { return func(o *options) { o.ctx = ctx } }
func WithServer(server network.Server) Option   { return func(o *options) { o.server = server } }
func WithTimeout(timeout time.Duration) Option  { return func(o *options) { o.timeout = timeout } }
func WithLocator(locator locate.Locator) Option { return func(o *options) { o.locator = locator } }
func WithRegistry(r registry.Registry) Option   { return func(o *options) { o.registry = r } }
func WithDispatch(dispatch cluster.Dispatch) Option {
	return func(o *options) { o.dispatch = dispatch }
}

func WithMetadata(metadata map[string]string) Option {
	return func(o *options) {
		if len(metadata) == 0 {
			return
		}
		if len(o.metadata) == 0 {
			o.metadata = make(map[string]string)
		}
		maps.Copy(o.metadata, metadata)
	}
}

func WithVersion(version string) Option          { return func(o *options) { o.version = version } }
func WithRetireDelay(delay time.Duration) Option { return func(o *options) { o.retireDelay = delay } }
func WithReceiveQueue(size int) Option           { return func(o *options) { o.receiveQueue = size } }
func WithDeliverWorkers(n int) Option            { return func(o *options) { o.deliverWorkers = n } }
func WithLoginMessageID(messageID int32) Option {
	return func(o *options) { o.loginMessageID = messageID }
}
func WithLobbyGameID(gameID int32) Option { return func(o *options) { o.lobbyGameID = gameID } }
func WithJWT(jwt *xjwt.JWT) Option        { return func(o *options) { o.jwt = jwt } }
func WithJWTSecretKey(secretKey string) Option {
	return func(o *options) { o.jwtSecretKey = secretKey }
}
func WithJWTIdentityKey(identityKey string) Option {
	return func(o *options) { o.jwtIdentityKey = identityKey }
}
func WithJWTSignAlgorithm(signAlgorithm xjwt.SignAlgorithm) Option {
	return func(o *options) { o.jwtSignAlgorithm = signAlgorithm }
}

func WithTestWhitelist(uids ...int64) Option {
	return func(o *options) {
		if o.testWhitelist == nil {
			o.testWhitelist = make(map[int64]struct{}, len(uids))
		}
		for _, uid := range uids {
			if uid > 0 {
				o.testWhitelist[uid] = struct{}{}
			}
		}
	}
}

func WithTestUserChecker(checker func(uid int64) bool) Option {
	return func(o *options) { o.testUserChecker = checker }
}

func (o *options) allowTestService(uid int64) bool {
	if uid <= 0 {
		return false
	}
	if o.testUserChecker != nil {
		return o.testUserChecker(uid)
	}
	_, ok := o.testWhitelist[uid]
	return ok
}
