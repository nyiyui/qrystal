package hokuto

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strings"
	"sync"

	"github.com/miekg/dns"
	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/mio"
	"github.com/nyiyui/qrystal/util"
)

// ~~stolen~~ copied from <https://gist.github.com/walm/0d67b4fb2d5daf3edd4fad3e13b162cb>.

var cc *central.Config
var ccLock sync.Mutex

var mask32 = net.CIDRMask(32, 32)

var c Config
var token []byte
var suffix string

type Config struct {
	Parent string `json:"parent"`
}

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

func handleQuery(m *dns.Msg) (nxdomain bool) {
	for _, q := range m.Question {
		log.Printf("Query for %s\n", q.Name)

		switch q.Qtype {
		case dns.TypeA:
			if strings.HasSuffix(q.Name, suffix) {
				ccLock.Lock()
				defer ccLock.Unlock()
				parts := strings.Split(strings.TrimSuffix(q.Name, suffix), ".")
				if len(parts) == 0 {
					nxdomain = true
					return
				}
				cnn := parts[0]
				cn := cc.Networks[cnn]
				if cn == nil {
					nxdomain = true
					return
				}
				switch len(parts) {
				case 1:
					for _, peer := range cn.Peers {
						returnPeer(m, q, peer)
					}
				case 2:
					pn := parts[1]
					peer := cn.Peers[pn]
					if cn == nil {
						nxdomain = true
						return
					}
					returnPeer(m, q, peer)
				}
			} else {
				nxdomain = true
				return
			}
		}
	}
	return
}

func handle(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false
	switch r.Opcode {
	case dns.OpcodeQuery:
		nxdomain := handleQuery(m)
		if nxdomain {
			m.MsgHdr.Rcode = dns.RcodeNameError
		}
	}
	w.WriteMsg(m)
}

func main2() {
	configPath := flag.String("config", "", "config path")
	addr := flag.String("addr", "", "dns bind address")
	flag.Parse()

	buf, err := os.ReadFile(*configPath)
	if err != nil {
		util.S.Fatalf("read config: %s", err)
	}
	err = json.Unmarshal(buf, &c)
	if err != nil {
		util.S.Fatalf("parse config: %s", err)
	}
	suffix = c.Parent + "."
	util.S.Infof("config parsed from %s", *configPath)

	dns.HandleFunc(".", handle)

	server := &dns.Server{Addr: *addr, Net: "udp"}
	util.S.Infof("listening on %s", server.Addr)
	err = server.ListenAndServe()
	if err != nil {
		util.S.Fatalf("listen failed: %s\n ", err.Error())
	}
	defer server.Shutdown()
}

type Hokuto struct{}

type UpdateCCQ struct {
	Token []byte
	CC    *central.Config
}

func (_ Hokuto) UpdateCC(q *UpdateCCQ, _ *bool) error {
	if !bytes.Equal(token, q.Token) {
		return errors.New("token mismatch")
	}
	if q.CC == nil {
		return errors.New("cc is nil")
	}
	ccLock.Lock()
	defer ccLock.Unlock()
	cc = q.CC
	return nil
}

type Mio struct{}

func (_ Mio) Ping(q string, r *string) error {
	*r = "pong"
	return nil
}

func Main() error {
	var tokenBase64 string
	var err error
	token, tokenBase64, err = mio.GenToken()
	if err != nil {
		util.S.Fatalf("GenToken: %s", err)
	}
	lis, addr, err := mio.Listen()
	if err != nil {
		util.S.Fatalf("Listen: %s", err)
	}
	fmt.Printf("addr:%s\n", addr)
	fmt.Printf("token:%s\n", tokenBase64)
	err = os.Stdout.Close()
	if err != nil {
		util.S.Fatalf("close stdout: %s", err)
	}
	util.S.Info("聞きます。")
	rs := rpc.NewServer()
	rs.Register(Hokuto{})
	rs.Register(Mio{})
	handler := mio.Guard(rs)
	err = http.Serve(lis, handler)
	if err != nil {
		util.S.Fatalf("serve: %s", err)
	}
	return nil
}
