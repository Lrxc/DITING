package dns

import (
	"net"
	"unicode/utf8"

	"github.com/miekg/dns"
)

// MaxEDETextLength 限制 EDE extra text 长度，防止 UDP 报文溢出
const MaxEDETextLength = 200

// EDNS0Option 为泛型 EDNS0 选项约束接口
type EDNS0Option interface {
	*dns.EDNS0_SUBNET | *dns.EDNS0_EDE | *dns.EDNS0_LOCAL | *dns.EDNS0_NSID | *dns.EDNS0_COOKIE | *dns.EDNS0_UL
	Option() uint16
}

// SetEdns0Option 向 DNS 报文的 OPT RR Extra 区域追加或覆盖指定的 EDNS0 选项
func SetEdns0Option(msg *dns.Msg, opt dns.EDNS0) bool {
	if msg == nil || opt == nil {
		return false
	}

	optRecord := msg.IsEdns0()
	if optRecord == nil {
		optRecord = new(dns.OPT)
		optRecord.Hdr.Name = "."
		optRecord.Hdr.Rrtype = dns.TypeOPT
		optRecord.SetUDPSize(dns.DefaultMsgSize)
		msg.Extra = append(msg.Extra, optRecord)
	}

	newOpts := make([]dns.EDNS0, 0, len(optRecord.Option)+1)
	for _, o := range optRecord.Option {
		if o.Option() != opt.Option() {
			newOpts = append(newOpts, o)
		}
	}
	newOpts = append(newOpts, opt)
	optRecord.Option = newOpts
	return true
}

// GetEdns0Option 从 DNS 报文的 OPT RR 中提取指定类型的 EDNS0 选项
func GetEdns0Option[T EDNS0Option](msg *dns.Msg) T {
	if msg == nil {
		return nil
	}
	optRecord := msg.IsEdns0()
	if optRecord == nil {
		return nil
	}
	for _, o := range optRecord.Option {
		if res, ok := o.(T); ok {
			return res
		}
	}
	return nil
}

// SetEDE 向 DNS 报文附加 RFC 8914 Extended DNS Error (EDE) 选项
func SetEDE(msg *dns.Msg, infoCode uint16, extraText string) bool {
	if msg == nil {
		return false
	}
	ede := &dns.EDNS0_EDE{
		InfoCode:  infoCode,
		ExtraText: TruncateText(extraText, MaxEDETextLength),
	}
	return SetEdns0Option(msg, ede)
}

// TruncateText 将字符串截断至至多 maxLen 字节，并在 UTF-8 字符边界对齐截断
func TruncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	cut := maxLen
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

// RemoveEdns0Option 从 DNS 报文的 OPT RR 中移除指定 Option 类型的选项
func RemoveEdns0Option(msg *dns.Msg, optionCode uint16) bool {
	if msg == nil {
		return false
	}
	optRecord := msg.IsEdns0()
	if optRecord == nil {
		return false
	}
	newOpts := make([]dns.EDNS0, 0, len(optRecord.Option))
	removed := false
	for _, o := range optRecord.Option {
		if o.Option() == optionCode {
			removed = true
		} else {
			newOpts = append(newOpts, o)
		}
	}
	if removed {
		optRecord.Option = newOpts
	}
	return removed
}

// StripECS 从 DNS 请求报文中剥离 EDNS Client Subnet (RFC 7871) 选项，保护客户端隐私
func StripECS(msg *dns.Msg) bool {
	return RemoveEdns0Option(msg, dns.EDNS0SUBNET)
}

// SetECS 向 DNS 报文附加指定的 EDNS Client Subnet 选项
func SetECS(msg *dns.Msg, subnetCIDR string) error {
	if msg == nil || subnetCIDR == "" {
		return nil
	}
	_, ipNet, err := net.ParseCIDR(subnetCIDR)
	if err != nil {
		return err
	}
	maskLen, _ := ipNet.Mask.Size()
	family := uint16(1) // IPv4
	if ipNet.IP.To4() == nil {
		family = 2 // IPv6
	}
	ecs := &dns.EDNS0_SUBNET{
		Code:          dns.EDNS0SUBNET,
		Family:        family,
		SourceNetmask: uint8(maskLen),
		SourceScope:   0,
		Address:       ipNet.IP,
	}
	SetEdns0Option(msg, ecs)
	return nil
}
