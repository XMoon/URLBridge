//go:build !windows && !linux

package guest

import "net"

func enableUDPBroadcast(conn *net.UDPConn) error {
	return nil
}
