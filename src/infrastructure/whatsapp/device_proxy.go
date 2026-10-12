package whatsapp

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
	"github.com/sirupsen/logrus"
)

// ErrProxyNotSupported is returned when the chat storage cannot keep a proxy per device.
var ErrProxyNotSupported = fmt.Errorf("this storage does not keep a proxy per device")

// deviceProxy returns the outbound proxy for one device: its own when set, otherwise
// the instance-wide WHATSAPP_PROXY. One instance can then serve numbers that leave
// through different proxies (one per number, a sticky session each), instead of
// running a container per proxy.
func (m *DeviceManager) deviceProxy(deviceID string) string {
	if store, ok := m.storage.(domainChatStorage.IDeviceProxyStore); ok {
		proxyURL, err := store.GetDeviceProxy(deviceID)
		if err != nil {
			logrus.Warnf("[DEVICE_MANAGER] failed to read the proxy of %s, falling back to WHATSAPP_PROXY: %v", deviceID, err)
		} else if proxyURL != nil && *proxyURL != "" {
			return *proxyURL
		}
	}
	return config.WhatsappProxy
}

// ValidateProxyURL accepts http, https and socks5 proxies with a host.
func ValidateProxyURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return pkgError.ValidationError("proxy_url must be a URL like http://user:pass@host:port or socks5://host:port")
	}
	switch u.Scheme {
	case "http", "https", "socks5":
		return nil
	}
	return pkgError.ValidationError("proxy_url scheme must be http, https or socks5")
}

// SetDeviceProxy stores (or, with nil, clears) the proxy of one device and applies it
// to its live client: the socket is closed, the proxy swapped and, for a paired
// device, the connection reopened through the new route. Errors from the store keep
// their type (sql.ErrNoRows for an unknown device).
func (m *DeviceManager) SetDeviceProxy(deviceID string, proxyURL *string) error {
	if m == nil {
		return fmt.Errorf("device manager not initialized")
	}
	store, ok := m.storage.(domainChatStorage.IDeviceProxyStore)
	if !ok {
		return ErrProxyNotSupported
	}
	if proxyURL != nil && strings.TrimSpace(*proxyURL) == "" {
		proxyURL = nil
	}
	if proxyURL != nil {
		if err := ValidateProxyURL(*proxyURL); err != nil {
			return err
		}
	}
	before := m.deviceProxy(deviceID)
	if err := store.SetDeviceProxy(deviceID, proxyURL); err != nil {
		return err
	}
	// Same effective proxy: the client already dials through it, so a reconnect
	// would only drop a live session.
	if m.deviceProxy(deviceID) == before {
		return nil
	}

	inst, ok := m.GetDevice(deviceID)
	if !ok {
		return nil // applied when the client is created
	}
	client := inst.GetClient()
	if client == nil {
		return nil
	}
	client.Disconnect()
	if effective := m.deviceProxy(deviceID); effective != "" {
		if err := client.SetProxyAddress(effective); err != nil {
			return err
		}
	} else {
		client.SetProxy(nil)
	}
	if client.Store != nil && client.Store.ID != nil {
		return client.Connect()
	}
	return nil
}
