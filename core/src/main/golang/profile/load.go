package profile

import (
	"fmt"
	"io/ioutil"
	"net"

	"github.com/Dreamacro/clash/component/fakeip"
	"github.com/Dreamacro/clash/config"
	"github.com/Dreamacro/clash/constant"
	"github.com/Dreamacro/clash/dns"
	"github.com/Dreamacro/clash/hub/executor"
	"github.com/Dreamacro/clash/log"
	"github.com/kr328/cfa/tun"
)

const tunAddress = "172.31.255.253/30"

const defaultConfig = `
log: debug
mode: Direct
Proxy:
- name: "broadcast"
  type: socks5
  server: 255.255.255.255
  port: 1080

Proxy Group:
- name: "select"
  type: select
  proxies: [DIRECT]

Rule:
- 'MATCH,DIRECT'
`

// LoadDefault - load default configure
func LoadDefault() {
	defaultC, err := parseConfig([]byte(defaultConfig), constant.Path.HomeDir())
	if err != nil {
		log.Warnln("Load Default Failure " + err.Error())
		return
	}

	executor.ApplyConfig(defaultC, true)

	tun.ResetDnsRedirect()
}

// LoadFromFile - load file
func LoadFromFile(path, baseDir string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	cfg, err := parseConfig(data, baseDir)
	if err != nil {
		return err
	}

	executor.ApplyConfig(cfg, true)

	if dns.DefaultResolver == nil && cfg.DNS.Enable {
		c := cfg.DNS

		r := dns.New(dns.Config{
			Main:         c.NameServer,
			Fallback:     c.Fallback,
			IPv6:         c.IPv6,
			EnhancedMode: c.EnhancedMode,
			Pool:         c.FakeIPRange,
			FallbackFilter: dns.FallbackFilter{
				GeoIP:  c.FallbackFilter.GeoIP,
				IPCIDR: c.FallbackFilter.IPCIDR,
			},
		})

		dns.DefaultResolver = r
	}

	if dns.DefaultResolver == nil {
		_, ipnet, _ := net.ParseCIDR("198.18.0.1/16")
		pool, _ := fakeip.New(ipnet, 1000, nil)

		var defaultDNSResolver = dns.New(dns.Config{
			Main: []dns.NameServer{
				dns.NameServer{Net: "tcp", Addr: "1.1.1.1:53"},
				dns.NameServer{Net: "tcp", Addr: "208.67.222.222:53"},
				dns.NameServer{Net: "", Addr: "119.29.29.29:53"},
				dns.NameServer{Net: "", Addr: "223.5.5.5:53"},
			},
			Fallback:     make([]dns.NameServer, 0),
			IPv6:         false,
			EnhancedMode: dns.FAKEIP,
			Pool:         pool,
			FallbackFilter: dns.FallbackFilter{
				GeoIP:  false,
				IPCIDR: make([]*net.IPNet, 0),
			},
		})

		dns.DefaultResolver = defaultDNSResolver
	}

	tun.ResetDnsRedirect()

	log.Infoln("Profile " + path + " loaded")

	return nil
}

func parseConfig(data []byte, baseDir string) (*config.Config, error) {
	raw, err := config.UnmarshalRawConfig(data)
	if err != nil {
		return nil, err
	}

	raw.ExternalUI = ""
	raw.ExternalController = ""
	raw.Rule = append([]string{fmt.Sprintf("IP-CIDR,%s,REJECT", tunAddress)}, raw.Rule...)

	return config.ParseRawConfig(raw, baseDir)
}
