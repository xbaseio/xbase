package registry

import "testing"

func TestMaxVersionForGameByServiceStatus(t *testing.T) {
	services := []*ServiceInstance{
		{Kind: "node", GameID: 1, Version: "1.0.0", Metadata: map[string]string{MetadataServiceStatusKey: string(ServiceStatusNormal)}},
		{Kind: "node", GameID: 1, Version: "1.2.0", Metadata: map[string]string{MetadataServiceStatusKey: string(ServiceStatusNormal)}},
		{Kind: "node", GameID: 1, Version: "1.5.0", Metadata: map[string]string{MetadataServiceStatusKey: string(ServiceStatusGray)}},
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

	if got := maxVersions[1][ServiceStatusGray]; got != "1.5.0" {
		t.Fatalf("unexpected gray version: %s", got)
	}

	if got := maxVersions[2][ServiceStatusTest]; got != "3.0.0" {
		t.Fatalf("unexpected game2 test version: %s", got)
	}
}

func TestPreferredServiceStatuses(t *testing.T) {
	tests := []struct {
		status ServiceStatus
		want   []ServiceStatus
	}{
		{status: ServiceStatusNormal, want: []ServiceStatus{ServiceStatusNormal}},
		{status: ServiceStatusGray, want: []ServiceStatus{ServiceStatusGray, ServiceStatusNormal}},
		{status: ServiceStatusTest, want: []ServiceStatus{ServiceStatusTest, ServiceStatusGray, ServiceStatusNormal}},
	}

	for _, tt := range tests {
		got := PreferredServiceStatuses(tt.status)
		if len(got) != len(tt.want) {
			t.Fatalf("status %s length = %d, want %d", tt.status, len(got), len(tt.want))
		}

		for i := range got {
			if got[i] != tt.want[i] {
				t.Fatalf("status %s[%d] = %s, want %s", tt.status, i, got[i], tt.want[i])
			}
		}
	}
}
