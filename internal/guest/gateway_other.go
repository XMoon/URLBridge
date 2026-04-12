//go:build !windows && !linux

package guest

func defaultGatewayURLs() []string {
	return nil
}
