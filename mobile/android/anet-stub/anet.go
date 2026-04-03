package anet
import "net"
func Interfaces() ([]net.Interface, error) {
	return []net.Interface{
		{Index: 1, MTU: 1500, Name: "wlan0", Flags: net.FlagUp | net.FlagMulticast},
	}, nil
}
func InterfaceAddrs() ([]net.Addr, error) {
	return []net.Addr{
		&net.IPNet{IP: net.ParseIP("0.0.0.0"), Mask: net.CIDRMask(0, 32)},
	}, nil
}
func InterfaceByIndex(index int) (*net.Interface, error) {
	ifaces, _ := Interfaces()
	return &ifaces[0], nil
}
func InterfaceByName(name string) (*net.Interface, error) {
	ifaces, _ := Interfaces()
	return &ifaces[0], nil
}
func InterfaceAddrsByInterface(iface *net.Interface) ([]net.Addr, error) {
	return InterfaceAddrs()
}