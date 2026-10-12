package whatsapp

import (
	"errors"
	"testing"

	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
)

type proxyRepo struct {
	domainChatStorage.IChatStorageRepository
	proxies map[string]string
}

func (p *proxyRepo) GetDeviceProxy(id string) (*string, error) {
	if v, ok := p.proxies[id]; ok {
		return &v, nil
	}
	return nil, nil
}

func (p *proxyRepo) SetDeviceProxy(id string, v *string) error {
	if v == nil {
		delete(p.proxies, id)
	} else {
		p.proxies[id] = *v
	}
	return nil
}

type plainRepo struct {
	domainChatStorage.IChatStorageRepository
}

// InitWaCLI builds the manager over the per-device wrapper: the proxy must reach the base.
func TestDeviceProxyPassesThroughTheDeviceWrapper(t *testing.T) {
	base := &proxyRepo{proxies: map[string]string{"dev-a": "http://u:p@proxy:3128"}}
	m := &DeviceManager{devices: map[string]*DeviceInstance{}, storage: newDeviceChatStorage("dev-global", base)}

	if got := m.deviceProxy("dev-a"); got != "http://u:p@proxy:3128" {
		t.Fatalf("deviceProxy(dev-a) = %q", got)
	}
	novo := "socks5://proxy2:1080"
	if err := m.SetDeviceProxy("dev-b", &novo); err != nil {
		t.Fatalf("SetDeviceProxy: %v", err)
	}
	if base.proxies["dev-b"] != novo {
		t.Fatalf("the wrapper did not store the proxy in the base: %v", base.proxies)
	}

	plain := &DeviceManager{devices: map[string]*DeviceInstance{}, storage: newDeviceChatStorage("dev-global", &plainRepo{})}
	if err := plain.SetDeviceProxy("dev-a", &novo); !errors.Is(err, ErrProxyNotSupported) {
		t.Fatalf("unsupported base: got %v", err)
	}
}
