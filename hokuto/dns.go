package hokuto

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/miekg/dns"
	"github.com/nyiyui/qrystal/hokuto/simple"
	"github.com/nyiyui/qrystal/util"
)

// ~~stolen~~ copied from <https://gist.github.com/walm/0d67b4fb2d5daf3edd4fad3e13b162cb>.

var extraParents []ExtraParent

var sc *simple.Config
var scLock sync.RWMutex

var mask32 = net.CIDRMask(32, 32)

var suffix string

func returnPeer(m *dns.Msg, q dns.Question, ipNets []net.IPNet) {
	for _, ipNet := range ipNets {
		if !bytes.Equal(ipNet.Mask, mask32) {
			// non-/32s seem very *fun* to deal with...
			continue
		}
		rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ipNet.IP.String()))
		if err == nil {
			m.Answer = append(m.Answer, rr)
		}
	}
}

func handleQuery(m *dns.Msg) (rcode int) {
	for _, q := range m.Question {
		util.S.Debugf("handleQuery: %s", q.Name)

		if strings.HasSuffix(q.Name, suffix) {
			switch q.Qtype {
			case dns.TypeA:
				rcode2 := handleInternal(m, q, suffix, "")
				if rcode2 != dns.RcodeSuccess {
					rcode = rcode2
					return
				}
			case dns.TypeSRV:
				rcode2 := handleInternalSRV(m, q, suffix, "")
				if rcode2 != dns.RcodeSuccess {
					rcode = rcode2
					return
				}
			}
		} else {
			// TODO: SRV support for extraParents
			for _, extra := range extraParents {
				if strings.HasSuffix(q.Name, extra.Domain) {
					switch q.Qtype {
					case dns.TypeA:
						rcode2 := handleInternal(m, q, extra.Domain, extra.Network)
						if rcode2 != dns.RcodeSuccess {
							rcode = rcode2
							return
						}
					case dns.TypeSRV:
						rcode2 := handleInternalSRV(m, q, extra.Domain, extra.Network)
						if rcode2 != dns.RcodeSuccess {
							rcode = rcode2
							return
						}
					}
				}
			}
		}
	}
	return
}

func handleInternal(m *dns.Msg, q dns.Question, suffix, cnn string) (rcode int) {
	scLock.RLock()
	defer scLock.RUnlock()
	if sc == nil {
		util.S.Errorf("cc nil (not updated?)")
		return dns.RcodeServerFailure
	}
	parts := strings.Split(strings.TrimSuffix(q.Name, suffix), ".")
	if len(parts) == 0 {
		util.S.Debugf("handleInternal nx no parts")
		return dns.RcodeNameError
	}
	reverse(parts)
	if cnn == "" {
		cnn = parts[0]
		parts = parts[1:]
	}
	cn, ok := sc.Networks[cnn]
	if !ok {
		util.S.Debugf("handleInternal nx net %s", cnn)
		return dns.RcodeNameError
	}
	switch len(parts) {
	case 0:
		util.S.Debugf("handleInternal net %s", cnn)
		for _, peerIP := range cn.PeerIPs {
			returnPeer(m, q, peerIP)
		}
	case 1:
		pn := parts[0]
		peerIP, ok := cn.PeerIPs[pn]
		if !ok {
			util.S.Debugf("handleInternal nx net %s peer %s", cnn, pn)
			return dns.RcodeNameError
		}
		returnPeer(m, q, peerIP)
	}
	return dns.RcodeSuccess
}

func handleInternalSRV(m *dns.Msg, q dns.Question, suffix, presetCNN string) (rcode int) {
	scLock.RLock()
	defer scLock.RUnlock()
	if sc == nil {
		util.S.Errorf("cc nil (not updated?)")
		return dns.RcodeServerFailure
	}
	parts := strings.Split(strings.TrimSuffix(q.Name, suffix), ".")
	reverse(parts)
	var cnn, pn, protocol, service string
	if presetCNN != "" {
		cnn = presetCNN
		// Domains:
		// - _service._protocol.PARENT
		// - _service._protocol.peer.PARENT
		switch len(parts) {
		case 2:
			protocol = parts[0]
			service = parts[1]
		case 3:
			pn = parts[0]
			protocol = parts[1]
			service = parts[2]
		default:
			util.S.Debugf("handleInternalSRV nx no parts")
			return dns.RcodeNameError
		}
	} else {
		// Domains:
		// - _service._protocol.network.PARENT
		// - _service._protocol.peer.network.PARENT
		switch len(parts) {
		case 3:
			cnn = parts[0]
			protocol = parts[1]
			service = parts[2]
		case 4:
			cnn = parts[0]
			pn = parts[1]
			protocol = parts[2]
			service = parts[3]
		default:
			util.S.Debugf("handleInternalSRV nx no parts")
			return dns.RcodeNameError
		}
	}

	cn, ok := sc.Networks[cnn]
	if !ok {
		util.S.Debugf("handleInternalSRV nx net %s", cnn)
		return dns.RcodeNameError
	}
	util.S.Debugf("handleInternalSRV: parts: %#v", parts)
	for _, record := range cn.SRVs[simple.ServiceProtocol{service, protocol}] {
		if pn != "" && record.PeerName != pn {
			continue
		}
		rr, err := dns.NewRR(fmt.Sprintf(
			"%s SRV %d %d %d %s.%s%s",
			q.Name,
			record.Priority,
			record.Weight,
			record.Port,
			record.PeerName,
			cnn,
			suffix,
		))
		if err == nil {
			m.Answer = append(m.Answer, rr)
		} else {
			util.S.Debugf("handleInternalSRV: parts: %#v error: %s", parts, err)
		}
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
