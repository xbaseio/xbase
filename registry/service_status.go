package registry

const MetadataServiceStatusKey = "service_status"

type ServiceStatus string

const (
	ServiceStatusNormal ServiceStatus = "normal"
	ServiceStatusTest   ServiceStatus = "test"
)

func ParseServiceStatus(status string) ServiceStatus {
	switch ServiceStatus(status) {
	case ServiceStatusTest:
		return ServiceStatusTest
	default:
		return ServiceStatusNormal
	}
}

func ServiceStatusOf(ins *ServiceInstance) ServiceStatus {
	if ins == nil || ins.Metadata == nil {
		return ServiceStatusNormal
	}

	return ParseServiceStatus(ins.Metadata[MetadataServiceStatusKey])
}

func MaxVersionForGameByServiceStatus(services []*ServiceInstance) map[int32]map[ServiceStatus]string {
	maxVersions := make(map[int32]map[ServiceStatus]string)

	for _, ins := range services {
		if ins == nil || ins.Kind != "node" {
			continue
		}

		status := ServiceStatusOf(ins)
		group, ok := maxVersions[ins.GameID]
		if !ok {
			group = make(map[ServiceStatus]string)
			maxVersions[ins.GameID] = group
		}

		if cur, ok := group[status]; !ok || CompareVersion(ins.Version, cur) > 0 {
			group[status] = ins.Version
		}
	}

	return maxVersions
}
