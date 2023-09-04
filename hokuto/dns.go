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
			switch q.Qtype {
			case dns.TypeA:
				rcode2 := handleInternal(m, q, suffix, "")
				if rcode2 != dns.RcodeSuccess {
					rcode = rcode2
					return
				}
			case dns.TypeSRV:
				rcode2 := handleInternalSRV(m, q, suffix)
				if rcode2 != dns.RcodeSuccess {
					rcode = rcode2
					return
				}
			}
		} else {
			// TODO: SRV support for extraParents
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
	cn, ok := cc.Networks[cnn]
	if !ok {
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

func handleInternalSRV(m *dns.Msg, q dns.Question, suffix string) (rcode int) {
	ccLock.Lock()
	defer ccLock.Unlock()
	if cc == nil {
		util.S.Errorf("cc nil (not updated?)")
		return dns.RcodeServerFailure
	}
	parts := strings.Split(strings.TrimSuffix(q.Name, suffix), ".")
	reverse(parts)
	var cnn, pn, protocol, service string
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
		util.S.Debugf("handleQuery nx no parts")
		return dns.RcodeNameError
	}
	cn, ok := cc.Networks[cnn]
	if !ok {
		util.S.Debugf("handleQuery nx net %s", cnn)
		return dns.RcodeNameError
	}
	type srvPeerPair struct {
		PeerName string
		SRV      central.SRV
	}
	var spps []srvPeerPair
	if pn == "" {
		spps = make([]srvPeerPair, 0)
		for pn, peer := range cn.Peers {
			for _, srv := range peer.SRVs {
				spps = append(spps, srvPeerPair{
					PeerName: pn,
					SRV:      srv,
				})
			}
		}
	} else {
		peer := cn.Peers[pn]
		if peer == nil {
			util.S.Debugf("handleQuery nx net %s peer %s", cnn, pn)
			return dns.RcodeNameError
		}
		spps = make([]srvPeerPair, 0)
		for _, srv := range peer.SRVs {
			spps = append(spps, srvPeerPair{
				PeerName: pn,
				SRV:      srv,
			})
		}
	}
	util.S.Infof("handleQuery debug: parts: %#v srvs: %#v", parts, spps)
	for _, spp := range spps {
		srv := spp.SRV
		if srv.Service != service || srv.Protocol != protocol {
			continue
		}
		rr, err := dns.NewRR(fmt.Sprintf(
			"%s SRV %d %d %d %s.%s%s",
			q.Name,
			srv.Priority,
			srv.Weight,
			srv.Port,
			spp.PeerName,
			cnn,
			suffix,
		))
		if err == nil {
			m.Answer = append(m.Answer, rr)
		} else {
			util.S.Errorf("handleQuery debug: parts: %#v srvs: %#v error: %s", parts, spps, err)
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
