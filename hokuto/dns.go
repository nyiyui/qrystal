package hokuto

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/miekg/dns"
	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/util"
)

// ~~stolen~~ copied from <https://gist.github.com/walm/0d67b4fb2d5daf3edd4fad3e13b162cb>.

var extraParents []ExtraParent

var cc *central.Config
var ccLock sync.Mutex

var mask32 = net.CIDRMask(32, 32)

var suffix string

func returnPeer(m *dns.Msg, q dns.Question, peer *central.Peer) {
	for _, in := range peer.AllowedIPs {
		if !bytes.Equal(net.IPNet(in).Mask, mask32) {
			// non-/32s seem very *fun* to deal with...
			continue
		}
		rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, in.IP.String()))
		if err == nil {
			m.Answer = append(m.Answer, rr)
		}
	}
}

func handleQuery(m *dns.Msg) (rcode int) {
	for _, q := range m.Question {
		util.S.Debugf("handleQuery: %s", q.Name)

		if strings.HasSuffix(q.Name, suffix) {
			if q.Qtype == dns.TypeA {
				rcode2 := handleInternal(m, q, suffix, "")
				if rcode2 != dns.RcodeSuccess {
					rcode = rcode2
					return
				}
			}
		} else {
			for _, extra := range extraParents {
				if strings.HasSuffix(q.Name, extra.Domain) {
					rcode2 := handleInternal(m, q, extra.Domain, extra.Network)
					if rcode2 != dns.RcodeSuccess {
						rcode = rcode2
						return
					}
				}
			}
		}
	}
	return
}

func handleInternal(m *dns.Msg, q dns.Question, suffix, cnn string) (rcode int) {
	ccLock.Lock()
	defer ccLock.Unlock()
	if cc == nil {
		util.S.Errorf("cc nil (not updated?)")
		return dns.RcodeServerFailure
	}
	parts := strings.Split(strings.TrimSuffix(q.Name, suffix), ".")
	if len(parts) == 0 {
		util.S.Debugf("handleQuery nx no parts")
		return dns.RcodeNameError
	}
	reverse(parts)
	if cnn == "" {
		cnn = parts[0]
		parts = parts[1:]
	}
	cn := cc.Networks[cnn]
	if cn == nil {
		util.S.Debugf("handleQuery nx net %s", cnn)
		return dns.RcodeNameError
	}
	switch len(parts) {
	case 0:
		util.S.Debugf("handleQuery net %s", cnn)
		for _, peer := range cn.Peers {
			returnPeer(m, q, peer)
		}
	case 1:
		pn := parts[0]
		peer := cn.Peers[pn]
		if peer == nil {
			util.S.Debugf("handleQuery nx net %s peer %s", cnn, pn)
			return dns.RcodeNameError
		}
		returnPeer(m, q, peer)
	}
	return dns.RcodeSuccess
}

func handle(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false
	switch r.Opcode {
	case dns.OpcodeQuery:
		m.MsgHdr.Rcode = handleQuery(m)
	}
	w.WriteMsg(m)
}
