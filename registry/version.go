package registry

import (
	"strconv"
	"strings"
)

// CompareVersion 比较版本号，返回值 >0 表示 a 更新，<0 表示 b 更新
func CompareVersion(a, b string) int {
	if a == b {
		return 0
	}

	if a == "" {
		a = "0"
	}

	if b == "" {
		b = "0"
	}

	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")
	n := len(partsA)
	if len(partsB) > n {
		n = len(partsB)
	}

	for i := 0; i < n; i++ {
		va := int64(0)
		vb := int64(0)

		if i < len(partsA) {
			va = parseVersionPart(partsA[i])
		}

		if i < len(partsB) {
			vb = parseVersionPart(partsB[i])
		}

		if va > vb {
			return 1
		}

		if va < vb {
			return -1
		}
	}

	return 0
}

// MaxVersion 返回最高版本号
func MaxVersion(versions ...string) string {
	if len(versions) == 0 {
		return ""
	}

	max := versions[0]
	for i := 1; i < len(versions); i++ {
		if CompareVersion(versions[i], max) > 0 {
			max = versions[i]
		}
	}

	return max
}

// MaxVersionForGame 按 GameID 分组取各组最高版本（Node）
func MaxVersionForGame(services []*ServiceInstance) map[int32]string {
	maxVersions := make(map[int32]string)

	for _, ins := range services {
		if ins == nil || ins.Kind != "node" {
			continue
		}

		if cur, ok := maxVersions[ins.GameID]; !ok || CompareVersion(ins.Version, cur) > 0 {
			maxVersions[ins.GameID] = ins.Version
		}
	}

	return maxVersions
}

// MaxVersionByKindAlias 按 Kind + Alias 分组取各组最高版本（Gate / Mesh）
func MaxVersionByKindAlias(services []*ServiceInstance, kind string) map[string]string {
	maxVersions := make(map[string]string)

	for _, ins := range services {
		if ins == nil || ins.Kind != kind {
			continue
		}

		if cur, ok := maxVersions[ins.Alias]; !ok || CompareVersion(ins.Version, cur) > 0 {
			maxVersions[ins.Alias] = ins.Version
		}
	}

	return maxVersions
}

// IsLatestVersion 判断实例是否为同组最高版本
func IsLatestVersion(ins *ServiceInstance, maxVersion string) bool {
	if ins == nil {
		return false
	}

	return CompareVersion(ins.Version, maxVersion) >= 0
}

func parseVersionPart(part string) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
	if err != nil {
		return 0
	}

	return v
}
