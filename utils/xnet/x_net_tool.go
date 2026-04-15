package xnet

import (
	"encoding/binary"
	"net"

	innernet "github.com/xbaseio/xbase/core/net"
)

// ExtractIP 提取主机地址
func ExtractIP(addr net.Addr) (string, error) {
	return innernet.ExtractIP(addr)
}

// ExtractPort 提取主机端口
func ExtractPort(addr net.Addr) (int, error) {
	return innernet.ExtractPort(addr)
}

// PublicIP 获取公网IP
func PublicIP() (string, error) {
	return innernet.PublicIP()
}

// PrivateIP 获取私网IP
func PrivateIP() (string, error) {
	return innernet.PrivateIP()
}

// FulfillAddr 补全地址
func FulfillAddr(addr string) string {
	return innernet.FulfillAddr(addr)
}

// AssignRandPort 分配一个随机端口
func AssignRandPort(ip ...string) (int, error) {
	return innernet.AssignRandPort(ip...)
}

// IP2Long IP地址转换为长整型
func IP2Long(ip string) uint32 {
	v := net.ParseIP(ip).To4()

	if len(v) == 0 {
		return 0
	}

	return binary.BigEndian.Uint32(v)
}

// Long2IP 长整型转换为字符串地址
func Long2IP(v uint32) string {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, v)
	return ip.String()
}
