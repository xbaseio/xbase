package dispatcher_test

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/core/endpoint"
	"github.com/xbaseio/xbase/internal/dispatcher"
	"github.com/xbaseio/xbase/registry"
	"github.com/xbaseio/xbase/xerrors"
)

func TestDispatcher_ReplaceServices(t *testing.T) {
	var (
		instance1 = &registry.ServiceInstance{
			ID:       "xc",
			Name:     "gate-3",
			Kind:     cluster.Node.String(),
			Alias:    "gate-3",
			State:    cluster.Work.String(),
			Endpoint: endpoint.NewEndpoint("grpc", "127.0.0.1:8003", false).String(),
			GameID:   2,
		}
		instance2 = &registry.ServiceInstance{
			ID:       "xa",
			Name:     "gate-1",
			Kind:     cluster.Node.String(),
			Alias:    "gate-1",
			State:    cluster.Work.String(),
			Endpoint: endpoint.NewEndpoint("grpc", "127.0.0.1:8001", false).String(),
			GameID:   1,
		}
		instance3 = &registry.ServiceInstance{
			ID:       "xb",
			Name:     "gate-2",
			Kind:     cluster.Node.String(),
			Alias:    "gate-2",
			State:    cluster.Hang.String(),
			Endpoint: endpoint.NewEndpoint("grpc", "127.0.0.1:8002", false).String(),
			Events:   []int{int(cluster.Disconnect)},
			GameID:   1,
		}
	)

	d := dispatcher.NewDispatcher(cluster.WeightRoundRobin)

	d.ReplaceServices(instance1, instance2, instance3)

	route, err := d.FindRoute(1)
	if err != nil {
		t.Errorf("find route failed: %v", err)
	} else {
		t.Log(route.FindEndpoint())
	}
}

func TestVisitWorkingNodesContinuesAfterFailures(t *testing.T) {
	const gameID = cluster.AdminGameID
	instance := func(id string, game int32, state cluster.State, port string) *registry.ServiceInstance {
		return &registry.ServiceInstance{
			ID: id, Kind: cluster.Node.String(), GameID: game, State: state.String(),
			Endpoint: endpoint.NewEndpoint("grpc", "127.0.0.1:"+port, false).String(),
		}
	}
	a := instance("admin-a", gameID, cluster.Work, "8001")
	b := instance("admin-b", gameID, cluster.Work, "8002")
	c := instance("admin-c", gameID, cluster.Work, "8003")
	a.Version, b.Version = "1", "2"
	d := dispatcher.NewDispatcher(cluster.Random)
	d.ReplaceServices(c, b, a, a, instance("self", gameID, cluster.Work, "8004"),
		instance("busy", gameID, cluster.Busy, "8005"), instance("lobby", cluster.LobbyGameID, cluster.Work, "8006"))
	route, err := d.FindGameRoute(gameID)
	if err != nil {
		t.Fatal(err)
	}
	var visited []string
	errA, errB := errors.New("first failure"), errors.New("second failure")
	err = route.VisitWorkingNodes("self", func(addr string) error {
		visited = append(visited, addr)
		switch addr {
		case "127.0.0.1:8001":
			return errA
		case "127.0.0.1:8002":
			return errB
		}
		return nil
	})
	if !slices.Equal(visited, []string{"127.0.0.1:8001", "127.0.0.1:8002", "127.0.0.1:8003"}) {
		t.Fatalf("visited %v", visited)
	}
	if !errors.Is(err, errA) || !errors.Is(err, errB) || !strings.Contains(err.Error(), "admin-a") || !strings.Contains(err.Error(), "admin-b") {
		t.Fatalf("unexpected fan-out error: %v", err)
	}
	d.ReplaceServices(a)
	route, err = d.FindGameRoute(gameID)
	if err != nil {
		t.Fatal(err)
	}
	if err := route.VisitWorkingNodes("admin-a", func(string) error { t.Fatal("visited self"); return nil }); !errors.Is(err, xerrors.ErrNotFoundEndpoint) {
		t.Fatalf("no remote targets: %v", err)
	}
}

func TestDispatcher_SelectGameNode(t *testing.T) {
	d := dispatcher.NewDispatcher(cluster.Random)
	d.ReplaceServices(&registry.ServiceInstance{
		ID:       "lobby-1",
		Name:     cluster.Node.String(),
		Kind:     cluster.Node.String(),
		Alias:    "lobby",
		State:    cluster.Work.String(),
		Endpoint: endpoint.NewEndpoint("grpc", "127.0.0.1:8001", false).String(),
		GameID:   cluster.LobbyGameID,
	})

	group, nid, err := d.SelectGameNode(cluster.LobbyGameID, registry.ServiceStatusNormal)
	if err != nil {
		t.Fatalf("select lobby node failed: %v", err)
	}
	if group != "lobby" || nid != "lobby-1" {
		t.Fatalf("selected lobby = (%q, %q), want (%q, %q)", group, nid, "lobby", "lobby-1")
	}
}

func TestDispatcher_WeightRoundRobin(t *testing.T) {
	var (
		instance1 = &registry.ServiceInstance{
			ID:       "xa",
			Name:     "node-1",
			Kind:     cluster.Node.String(),
			Alias:    "node-1",
			State:    cluster.Work.String(),
			Endpoint: endpoint.NewEndpoint("grpc", "127.0.0.1:8001", false).String(),
			Weight:   4,
			GameID:   1,
		}
		instance2 = &registry.ServiceInstance{
			ID:       "xb",
			Name:     "node-2",
			Kind:     cluster.Node.String(),
			Alias:    "node-2",
			State:    cluster.Work.String(),
			Endpoint: endpoint.NewEndpoint("grpc", "127.0.0.1:8002", false).String(),
			Weight:   2,
			GameID:   1,
		}
		instance3 = &registry.ServiceInstance{
			ID:       "xc",
			Name:     "node-3",
			Kind:     cluster.Node.String(),
			Alias:    "node-3",
			State:    cluster.Work.String(),
			Endpoint: endpoint.NewEndpoint("grpc", "127.0.0.1:8003", false).String(),
			Weight:   1,
			GameID:   1,
		}
	)

	d := dispatcher.NewDispatcher(cluster.WeightRoundRobin)
	d.ReplaceServices(instance1, instance2, instance3)

	counts := make(map[string]int)
	totalRounds := 200

	for range totalRounds {
		route, err := d.FindRoute(1)
		if err != nil {
			t.Errorf("find route failed: %v", err)
			return
		}

		ep, err := route.FindEndpoint()
		if err != nil {
			t.Errorf("find endpoint failed: %v", err)
			return
		}

		parsedEp, err := endpoint.ParseEndpoint(ep.String())
		if err != nil {
			t.Errorf("parse endpoint failed: %v", err)
			return
		}
		addr := parsedEp.Address()
		counts[addr]++
	}

	expectedRatios := map[string]float64{
		"127.0.0.1:8001": 4.0 / 7.0,
		"127.0.0.1:8002": 2.0 / 7.0,
		"127.0.0.1:8003": 1.0 / 7.0,
	}

	t.Log("Distribution results:")
	for addr, count := range counts {
		ratio := float64(count) / float64(totalRounds)
		expected := expectedRatios[addr]
		t.Logf("Server %s: selected %d times, ratio=%.3f, expected=%.3f",
			addr, count, ratio, expected)

		if delta := math.Abs(ratio - expected); delta > 0.05 {
			t.Errorf("distribution ratio for %s is %.3f, want %.3f (±0.05)",
				addr, ratio, expected)
		}
	}

	total := 0
	for _, count := range counts {
		total += count
	}
	if total != totalRounds {
		t.Errorf("total rounds = %d, want %d", total, totalRounds)
	}
	for addr := range expectedRatios {
		if counts[addr] == 0 {
			t.Errorf("server %s was never selected", addr)
		}
	}
}

func BenchmarkDispatcher_WeightRoundRobin(b *testing.B) {
	instances := []*registry.ServiceInstance{
		{
			ID:       "xa",
			Name:     "node-1",
			Kind:     cluster.Node.String(),
			Alias:    "node-1",
			State:    cluster.Work.String(),
			Weight:   4,
			Endpoint: endpoint.NewEndpoint("grpc", "127.0.0.1:8001", false).String(),
			GameID:   1,
		},
		{
			ID:       "xb",
			Name:     "node-2",
			Kind:     cluster.Node.String(),
			Alias:    "node-2",
			State:    cluster.Work.String(),
			Weight:   2,
			Endpoint: endpoint.NewEndpoint("grpc", "127.0.0.1:8002", false).String(),
			GameID:   1,
		},
		{
			ID:       "xc",
			Name:     "node-3",
			Kind:     cluster.Node.String(),
			Alias:    "node-3",
			State:    cluster.Work.String(),
			Weight:   1,
			Endpoint: endpoint.NewEndpoint("grpc", "127.0.0.1:8003", false).String(),
			GameID:   1,
		},
	}

	benchmarks := []struct {
		name          string
		concurrency   int
		instanceCount int
	}{
		{"Concurrency1_Instances3", 1, 3},
		{"Concurrency10_Instances3", 10, 3},
		{"Concurrency100_Instances3", 100, 3},
		{"Concurrency1_Instances10", 1, 10},
		{"Concurrency10_Instances10", 10, 10},
		{"Concurrency100_Instances10", 100, 10},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			testInstances := make([]*registry.ServiceInstance, bm.instanceCount)
			for i := 0; i < bm.instanceCount; i++ {
				if i < len(instances) {
					testInstances[i] = instances[i]
				} else {
					last := instances[len(instances)-1]
					testInstances[i] = &registry.ServiceInstance{
						ID:       fmt.Sprintf("x%d", i),
						Name:     fmt.Sprintf("gate-%d", i+1),
						Kind:     last.Kind,
						Alias:    fmt.Sprintf("gate-%d", i+1),
						State:    last.State,
						Weight:   1,
						Endpoint: endpoint.NewEndpoint("grpc", fmt.Sprintf("127.0.0.1:%d", 8000+i), false).String(),
						GameID:   last.GameID,
					}
				}
			}

			d := dispatcher.NewDispatcher(cluster.WeightRoundRobin)
			d.ReplaceServices(testInstances...)

			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					route, err := d.FindRoute(1)
					if err != nil {
						b.Fatal(err)
					}
					_, err = route.FindEndpoint()
					if err != nil {
						b.Fatal(err)
					}
				}
			})

			b.ReportAllocs()
		})
	}
}

func TestDispatcher_ServiceStatusRouting(t *testing.T) {
	services := []*registry.ServiceInstance{
		{
			ID:       "normal-v1",
			Name:     "node-normal-v1",
			Kind:     cluster.Node.String(),
			Alias:    "node",
			State:    cluster.Work.String(),
			Endpoint: endpoint.NewEndpoint("grpc", "127.0.0.1:8101", false).String(),
			GameID:   1,
			Version:  "1.0.0",
			Metadata: map[string]string{registry.MetadataServiceStatusKey: string(registry.ServiceStatusNormal)},
		},
		{
			ID:       "normal-v2",
			Name:     "node-normal-v2",
			Kind:     cluster.Node.String(),
			Alias:    "node",
			State:    cluster.Work.String(),
			Endpoint: endpoint.NewEndpoint("grpc", "127.0.0.1:8102", false).String(),
			GameID:   1,
			Version:  "2.0.0",
			Metadata: map[string]string{registry.MetadataServiceStatusKey: string(registry.ServiceStatusNormal)},
		},
		{
			ID:       "gray-v25",
			Name:     "node-gray-v25",
			Kind:     cluster.Node.String(),
			Alias:    "node",
			State:    cluster.Work.String(),
			Endpoint: endpoint.NewEndpoint("grpc", "127.0.0.1:81025", false).String(),
			GameID:   1,
			Version:  "2.5.0",
			Metadata: map[string]string{registry.MetadataServiceStatusKey: string(registry.ServiceStatusGray)},
		},
		{
			ID:       "test-v3",
			Name:     "node-test-v3",
			Kind:     cluster.Node.String(),
			Alias:    "node",
			State:    cluster.Work.String(),
			Endpoint: endpoint.NewEndpoint("grpc", "127.0.0.1:8103", false).String(),
			GameID:   1,
			Version:  "3.0.0",
			Metadata: map[string]string{registry.MetadataServiceStatusKey: string(registry.ServiceStatusTest)},
		},
	}

	d := dispatcher.NewDispatcher(cluster.Random)
	d.ReplaceServices(services...)

	route, err := d.FindRoute(1)
	if err != nil {
		t.Fatalf("find route failed: %v", err)
	}

	ep, err := route.FindEndpointForUser(false)
	if err != nil {
		t.Fatalf("find normal endpoint failed: %v", err)
	}
	if got := ep.Address(); got != "127.0.0.1:8102" {
		t.Fatalf("unexpected normal endpoint: %s", got)
	}

	ep, err = route.FindEndpointForServiceStatus(registry.ServiceStatusGray)
	if err != nil {
		t.Fatalf("find gray endpoint failed: %v", err)
	}
	if got := ep.Address(); got != "127.0.0.1:81025" {
		t.Fatalf("unexpected gray endpoint: %s", got)
	}

	ep, err = route.FindEndpointForServiceStatus(registry.ServiceStatusTest)
	if err != nil {
		t.Fatalf("find test endpoint failed: %v", err)
	}
	if got := ep.Address(); got != "127.0.0.1:8103" {
		t.Fatalf("unexpected test endpoint: %s", got)
	}
}
