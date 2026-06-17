package node

import (
	"context"
	"maps"
	"runtime"
	"time"

	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/crypto"
	"github.com/xbaseio/xbase/encoding"
	"github.com/xbaseio/xbase/etc"
	"github.com/xbaseio/xbase/locate"
	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/registry"
	"github.com/xbaseio/xbase/transport"
	"github.com/xbaseio/xbase/utils/xuuid"
)

const (
	defaultName        = "node"
	defaultAddr        = ":0"
	defaultCodec       = "proto"
	defaultTimeout     = 3 * time.Second
	defaultWeight      = 1
	defaultVersion     = "1"
	defaultRetireDelay = 10 * time.Minute
)

const (
	defaultIDKey             = "etc.cluster.node.id"
	defaultNameKey           = "etc.cluster.node.name"
	defaultAddrKey           = "etc.cluster.node.addr"
	defaultExposeKey         = "etc.cluster.node.expose"
	defaultCodecKey          = "etc.cluster.node.codec"
	defaultWeightKey         = "etc.cluster.node.weight"
	defaultTimeoutKey        = "etc.cluster.node.timeout"
	defaultMetadataKey       = "etc.cluster.node.metadata"
	defaultGameIDKey         = "etc.cluster.node.gameID"
	defaultVersionKey        = "etc.cluster.node.version"
	defaultRetireDelayKey    = "etc.cluster.node.retireDelay"
	defaultRequestWorkersKey = "etc.cluster.node.requestWorkers"
	defaultEventWorkersKey   = "etc.cluster.node.eventWorkers"
	defaultDeliverTimeoutKey = "etc.cluster.node.deliverTimeout"
	defaultMailboxTimeoutKey = "etc.cluster.node.mailboxTimeout"
	defaultServiceStatusKey  = "etc.cluster.node.serviceStatus"
)

type SchedulingModel string
type Option func(o *options)

type options struct {
	ctx            context.Context
	id             string
	name           string
	addr           string
	expose         bool
	codec          encoding.Codec
	weight         int
	timeout        time.Duration
	locator        locate.Locator
	registry       registry.Registry
	encryptor      crypto.Encryptor
	transporter    transport.Transporter
	metadata       map[string]string
	nodeKind       cluster.NodeKind
	gameID         int32
	version        string
	serviceStatus  registry.ServiceStatus
	retireDelay    time.Duration
	requestWorkers int
	eventWorkers   int
	deliverTimeout time.Duration
	mailboxTimeout time.Duration
}

func defaultOptions() *options {
	opts := &options{
		ctx:            context.Background(),
		name:           defaultName,
		addr:           defaultAddr,
		codec:          encoding.Invoke(defaultCodec),
		weight:         defaultWeight,
		timeout:        defaultTimeout,
		metadata:       make(map[string]string),
		expose:         etc.Get(defaultExposeKey).Bool(),
		nodeKind:       cluster.Node_Normal,
		gameID:         0,
		version:        defaultVersion,
		serviceStatus:  registry.ServiceStatusNormal,
		retireDelay:    defaultRetireDelay,
		requestWorkers: max(runtime.NumCPU(), 1),
		eventWorkers:   2,
		deliverTimeout: defaultTimeout,
		mailboxTimeout: defaultTimeout,
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

	if codec := etc.Get(defaultCodecKey).String(); codec != "" {
		opts.codec = encoding.Invoke(codec)
	}

	if timeout := etc.Get(defaultTimeoutKey).Duration(); timeout > 0 {
		opts.timeout = timeout
	}

	if weight := etc.Get(defaultWeightKey).Int(); weight > 0 {
		opts.weight = weight
	}

	if err := etc.Get(defaultMetadataKey).Scan(&opts.metadata); err != nil {
		log.Warnf("scan node metadata failed: %v", err)
	}

	if etc.Has(defaultGameIDKey) {
		opts.gameID = int32(etc.Get(defaultGameIDKey).Int())
	}

	if version := etc.Get(defaultVersionKey).String(); version != "" {
		opts.version = version
	}

	if serviceStatus := etc.Get(defaultServiceStatusKey).String(); serviceStatus != "" {
		opts.serviceStatus = registry.ParseServiceStatus(serviceStatus)
	}

	if delay := etc.Get(defaultRetireDelayKey).Duration(); delay > 0 {
		opts.retireDelay = delay
	}

	if n := etc.Get(defaultRequestWorkersKey).Int(); n > 0 {
		opts.requestWorkers = n
	}

	if n := etc.Get(defaultEventWorkersKey).Int(); n > 0 {
		opts.eventWorkers = n
	}

	if d := etc.Get(defaultDeliverTimeoutKey).Duration(); d > 0 {
		opts.deliverTimeout = d
	}

	if d := etc.Get(defaultMailboxTimeoutKey).Duration(); d > 0 {
		opts.mailboxTimeout = d
	}

	return opts
}

func WithID(id string) Option                   { return func(o *options) { o.id = id } }
func WithName(name string) Option               { return func(o *options) { o.name = name } }
func WithAddr(addr string) Option               { return func(o *options) { o.addr = addr } }
func WithExpose(expose bool) Option             { return func(o *options) { o.expose = expose } }
func WithCodec(codec encoding.Codec) Option     { return func(o *options) { o.codec = codec } }
func WithContext(ctx context.Context) Option    { return func(o *options) { o.ctx = ctx } }
func WithTimeout(timeout time.Duration) Option  { return func(o *options) { o.timeout = timeout } }
func WithLocator(locator locate.Locator) Option { return func(o *options) { o.locator = locator } }
func WithRegistry(r registry.Registry) Option   { return func(o *options) { o.registry = r } }
func WithEncryptor(encryptor crypto.Encryptor) Option {
	return func(o *options) { o.encryptor = encryptor }
}
func WithTransporter(transporter transport.Transporter) Option {
	return func(o *options) { o.transporter = transporter }
}
func WithWeight(weight int) Option { return func(o *options) { o.weight = weight } }

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

func WithNodeKind(nodeKind cluster.NodeKind) Option {
	return func(o *options) { o.nodeKind = nodeKind }
}

func WithGameID(gameID int32) Option {
	return func(o *options) { o.gameID = gameID }
}

func WithVersion(version string) Option {
	return func(o *options) { o.version = version }
}

func WithServiceStatus(status registry.ServiceStatus) Option {
	return func(o *options) { o.serviceStatus = registry.ParseServiceStatus(string(status)) }
}

func WithRetireDelay(delay time.Duration) Option {
	return func(o *options) { o.retireDelay = delay }
}

func WithRequestWorkers(n int) Option {
	return func(o *options) { o.requestWorkers = n }
}

func WithEventWorkers(n int) Option {
	return func(o *options) { o.eventWorkers = n }
}

func WithDeliverTimeout(timeout time.Duration) Option {
	return func(o *options) { o.deliverTimeout = timeout }
}

func WithMailboxTimeout(timeout time.Duration) Option {
	return func(o *options) { o.mailboxTimeout = timeout }
}
