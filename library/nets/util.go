package nets

import (
	"net"
	"strings"
)

type NetInterfaceInfoList []*NetInterfaceInfo
type NetInterfaceInfo struct {
	Name string `json:"name"`
	Mac  string `json:"mac"`
	IPv4 string `json:"ipv4"`
	IPv6 string `json:"ipv6"`
}

func (n *NetInterfaceInfoList) GetByName(name string) *NetInterfaceInfo {
	for _, v := range *n {
		if v.Name == name {
			return v
		}
	}
	return nil
}

// GetLocalNetInterfaceAddress 获取本地接口地址信息
func GetLocalNetInterfaceAddress() (NetInterfaceInfoList, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	netList := NetInterfaceInfoList{}
	for _, inter := range interfaces {
		netInfo := &NetInterfaceInfo{
			Name: inter.Name,
			Mac:  inter.HardwareAddr.String(),
			IPv4: "",
			IPv6: "",
		}
		addrs, err2 := inter.Addrs()
		if err2 != nil {
			continue
		}
		for _, address := range addrs {
			if ipnet, ok := address.(*net.IPNet); ok {
				if ipnet.IP.To4() != nil { // ipv4
					if ipnet.IP.String() != "" {
						netInfo.IPv4 = ipnet.IP.String()
					}
				} else if ipnet.IP.To16() != nil && !ipnet.IP.IsLoopback() { // ipv6
					if ipnet.IP.String() != "" && !strings.HasPrefix(ipnet.IP.String(), "fe80") { // 去掉本地IPv6地址
						netInfo.IPv6 = ipnet.IP.String()
					}
				}
			}
		}
		netList = append(netList, netInfo)
	}
	return netList, nil
}
