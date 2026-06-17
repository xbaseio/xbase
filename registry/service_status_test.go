package registry

import "testing"

func TestMaxVersionForGameByServiceStatus(t *testing.T) {
	services := []*ServiceInstance{
		{Kind: "node", GameID: 1, Version: "1.0.0", Metadata: map[string]string{MetadataServiceStatusKey: string(ServiceStatusNormal)}},
		{Kind: "node", GameID: 1, Version: "1.2.0", Metadata: map[string]string{MetadataServiceStatusKey: string(ServiceStatusNormal)}},
		{Kind: "node", GameID: 1, Version: "2.0.0", Metadata: map[string]string{MetadataServiceStatusKey: string(ServiceStatusTest)}},
		{Kind: "node", GameID: 2, Version: "3.0.0", Metadata: map[string]string{MetadataServiceStatusKey: string(ServiceStatusTest)}},
	}

	maxVersions := MaxVersionForGameByServiceStatus(services)

	if got := maxVersions[1][ServiceStatusNormal]; got != "1.2.0" {
		t.Fatalf("unexpected normal version: %s", got)
	}

	if got := maxVersions[1][ServiceStatusTest]; got != "2.0.0" {
		t.Fatalf("unexpected test version: %s", got)
	}

	if got := maxVersions[2][ServiceStatusTest]; got != "3.0.0" {
		t.Fatalf("unexpected game2 test version: %s", got)
	}
}
