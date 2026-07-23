package gate

import (
	"context"
	"hash/fnv"
	"maps"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/xbaseio/xbase/cluster"
	gatepolicy "github.com/xbaseio/xbase/component/eventbus"
	"github.com/xbaseio/xbase/etc"
	"github.com/xbaseio/xbase/locate"
	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/network"
	"github.com/xbaseio/xbase/packet"
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
	defaultLobbyGameID    = cluster.LobbyGameID
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
	defaultGrayWhitelistKey    = "etc.cluster.gate.grayWhitelist"
	defaultGrayTrafficPctKey   = "etc.cluster.gate.grayTrafficPercent"
	defaultGrayTrafficSaltKey  = "etc.cluster.gate.grayTrafficSalt"
	defaultTestWhitelistKey    = "etc.cluster.gate.testWhitelist"
)

type Option func(o *options)

// MessageDispatcher handles Gate control messages (GameID = GateGameID)
// that are not consumed by a built-in Gate handler.
type MessageDispatcher func(ctx context.Context, conn network.Conn, message *packet.Message)

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
	messageDispatcher MessageDispatcher
	loginMessageID   int32
	lobbyGameID      int32
	jwt              *xjwt.JWT
	jwtSecretKey     string
	jwtIdentityKey   string
	jwtSignAlgorithm xjwt.SignAlgorithm
	metadata         map[string]string
	policyMu         sync.RWMutex
	grayWhitelist    map[int64]struct{}
	grayUserChecker  func(uid int64) bool
	grayTrafficPct   int
	grayTrafficSalt  string
	testWhitelist    map[int64]struct{}
	testUserChecker  func(uid int64) bool
}

type serviceStatusDecision struct {
	targetStatus      registry.ServiceStatus
	reason            string
	grayWhitelistHit  bool
	testWhitelistHit  bool
	grayTrafficHit    bool
	grayTrafficBucket int
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
		grayWhitelist:    make(map[int64]struct{}),
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

	for _, uid := range etc.Get(defaultGrayWhitelistKey).Int64s() {
		if uid > 0 {
			opts.grayWhitelist[uid] = struct{}{}
		}
	}

	opts.grayTrafficPct = normalizeTrafficPercent(etc.Get(defaultGrayTrafficPctKey).Int())
	if salt := etc.Get(defaultGrayTrafficSaltKey).String(); salt != "" {
		opts.grayTrafficSalt = salt
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
func WithMessageDispatcher(dispatcher MessageDispatcher) Option {
	return func(o *options) { o.messageDispatcher = dispatcher }
}
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

func WithGrayWhitelist(uids ...int64) Option {
	return func(o *options) {
		if o.grayWhitelist == nil {
			o.grayWhitelist = make(map[int64]struct{}, len(uids))
		}
		for _, uid := range uids {
			if uid > 0 {
				o.grayWhitelist[uid] = struct{}{}
			}
		}
	}
}

func WithGrayUserChecker(checker func(uid int64) bool) Option {
	return func(o *options) { o.grayUserChecker = checker }
}

func WithGrayTrafficPercent(percent int) Option {
	return func(o *options) { o.grayTrafficPct = normalizeTrafficPercent(percent) }
}

func WithGrayTrafficSalt(salt string) Option {
	return func(o *options) { o.grayTrafficSalt = salt }
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

func (o *options) resolveServiceStatus(uid int64) registry.ServiceStatus {
	return o.resolveServiceStatusDecision(uid).targetStatus
}

func (o *options) resolveServiceStatusDecision(uid int64) serviceStatusDecision {
	o.policyMu.RLock()
	defer o.policyMu.RUnlock()

	decision := serviceStatusDecision{
		targetStatus: registry.ServiceStatusNormal,
		reason:       "normal_default",
	}

	if uid <= 0 {
		decision.reason = "invalid_uid"
		return decision
	}

	if o.testUserChecker != nil && o.testUserChecker(uid) {
		decision.targetStatus = registry.ServiceStatusTest
		decision.reason = "test_checker"
		return decision
	}

	if _, ok := o.testWhitelist[uid]; ok {
		decision.targetStatus = registry.ServiceStatusTest
		decision.reason = "test_whitelist"
		decision.testWhitelistHit = true
		return decision
	}

	if o.grayUserChecker != nil && o.grayUserChecker(uid) {
		decision.targetStatus = registry.ServiceStatusGray
		decision.reason = "gray_checker"
		return decision
	}

	if _, ok := o.grayWhitelist[uid]; ok {
		decision.targetStatus = registry.ServiceStatusGray
		decision.reason = "gray_whitelist"
		decision.grayWhitelistHit = true
		return decision
	}

	if o.grayTrafficPct > 0 {
		bucket := trafficBucket(uid, o.grayTrafficSalt)
		decision.grayTrafficBucket = bucket
		if bucket < o.grayTrafficPct {
			decision.targetStatus = registry.ServiceStatusGray
			decision.reason = "gray_percent"
			decision.grayTrafficHit = true
			return decision
		}
	}

	return decision
}

func (o *options) serviceStatusPolicy() gatepolicy.ServiceStatusPolicy {
	o.policyMu.RLock()
	defer o.policyMu.RUnlock()

	return gatepolicy.ServiceStatusPolicy{
		GrayWhitelist:      mapKeys(o.grayWhitelist),
		GrayTrafficPercent: o.grayTrafficPct,
		GrayTrafficSalt:    o.grayTrafficSalt,
		TestWhitelist:      mapKeys(o.testWhitelist),
	}
}

func (o *options) updateServiceStatusPolicy(policy gatepolicy.ServiceStatusPolicy, isMathchGate bool) gatepolicy.ServiceStatusPolicy {
	o.policyMu.Lock()
	defer o.policyMu.Unlock()

	o.grayWhitelist = sliceToSet(policy.GrayWhitelist)
	o.grayTrafficPct = normalizeTrafficPercent(policy.GrayTrafficPercent)

	if isMathchGate {
		o.grayTrafficSalt = policy.GrayTrafficSalt
		o.testWhitelist = sliceToSet(policy.TestWhitelist)
	}

	return gatepolicy.ServiceStatusPolicy{
		GrayWhitelist:      mapKeys(o.grayWhitelist),
		GrayTrafficPercent: o.grayTrafficPct,
		GrayTrafficSalt:    o.grayTrafficSalt,
		TestWhitelist:      mapKeys(o.testWhitelist),
	}
}

func normalizeTrafficPercent(percent int) int {
	switch {
	case percent < 0:
		return 0
	case percent > 100:
		return 100
	default:
		return percent
	}
}

func trafficBucket(uid int64, salt string) int {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(salt))
	_, _ = hasher.Write([]byte(":"))
	_, _ = hasher.Write([]byte(strconv.FormatInt(uid, 10)))

	return int(hasher.Sum32() % 100)
}

func mapKeys(set map[int64]struct{}) []int64 {
	if len(set) == 0 {
		return nil
	}

	keys := make([]int64, 0, len(set))
	for uid := range set {
		keys = append(keys, uid)
	}

	return keys
}

func sliceToSet(uids []int64) map[int64]struct{} {
	set := make(map[int64]struct{}, len(uids))
	for _, uid := range uids {
		if uid > 0 {
			set[uid] = struct{}{}
		}
	}

	return set
}
