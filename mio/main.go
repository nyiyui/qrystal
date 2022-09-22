// Package mio provides a local RPC server to configure WireGuard. Its purpose
// is to try to reduce the number of programs running with high priviledges.
package mio

import (
	"bytes"
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strings"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func init() {
	gob.Register(new(ConfigureDeviceQ))
}

type Server struct {
	client  *wgctrl.Client
	handler http.Handler
}

func genToken() ([]byte, string, error) {
	b := make([]byte, 4)
	_, err := rand.Read(b)
	if err != nil {
		return nil, "", err
	}
	return b, base64.StdEncoding.EncodeToString(b), nil
}

func Main() error {
	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("WGクライアント生成：%w", err)
	}
	defer client.Close()

	_, err = client.Devices()
	if err != nil {
		return fmt.Errorf("WGクライントテスト: %w", err)
	}

	token, tokenBase64, err := genToken()
	if err != nil {
		log.Fatalf("トークン生成: %s", err)
	}
	_ = tokenBase64

	rs := rpc.NewServer()
	err = rs.Register(&Mio{client: client, token: token})
	if err != nil {
		return err
	}

	handler := blockNonLocal(rs)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("バインド: %s", err)
	}
	fmt.Printf("port:%d\n", lis.Addr().(*net.TCPAddr).Port)
	fmt.Printf("token:%s\n", tokenBase64)
	err = os.Stdout.Close()
	if err != nil {
		log.Fatalf("close stdout: %s", err)
	}
	log.Printf("聞きます。")
	return http.Serve(lis, handler)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

type Mio struct {
	client *wgctrl.Client
	token  []byte
}

type ConfigureDeviceQ struct {
	Token   []byte // put token here for simplicity
	Name    string
	Config  *wgtypes.Config
	Address []net.IPNet
}

func toString(config *wgtypes.Config) string {
	b := new(strings.Builder)
	fmt.Fprint(b, "[Interface]\n")
	fmt.Fprintf(b, "PrivateKey = %s\n", config.PrivateKey)
	fmt.Fprintf(b, "ListenPort = %d\n", config.ListenPort)
	fmt.Fprintf(b, "FirewallMark = %d\n", config.FirewallMark)
	fmt.Fprintf(b, "ReplacePeers = %t\n", config.ReplacePeers)
	for i, peer := range config.Peers {
		fmt.Fprintf(b, "\n[Peer %d]\n", i)
		fmt.Fprintf(b, "PublicKey = %s\n", peer.PublicKey)
		fmt.Fprintf(b, "Remove = %t\n", peer.Remove)
		fmt.Fprintf(b, "UpdateOnly = %t\n", peer.UpdateOnly)
		fmt.Fprintf(b, "PresharedKey = %s\n", peer.PresharedKey)
		fmt.Fprintf(b, "Endpoint = %s\n", peer.Endpoint)
		fmt.Fprintf(b, "PersistentKeepalive = %s\n", peer.PersistentKeepaliveInterval)
		fmt.Fprintf(b, "ReplaceAllowedIPs = %t\n", peer.ReplaceAllowedIPs)
		fmt.Fprintf(b, "AllowedIPs = %s\n", peer.AllowedIPs)
	}
	return b.String()
}

// TODO: allow removing devices

func (sm *Mio) ConfigureDevice(q ConfigureDeviceQ, r *string) error {
	if !bytes.Equal(sm.token, q.Token) {
		// Better to die as *something* is broken or someone is trying to do something bad.
		panic("token mismatch")
	}

	// These errors shouldn't happen but this way, these are easier to debug.
	// TODO: consider returning an error (instead of setting *r) for these types of errors.
	if q.Name == "" {
		*r = "q.Name blank"
		return nil
	}
	if q.Config == nil {
		*r = "q.Config nil"
		return nil
	}
	if len(q.Address) == 0 {
		*r = "q.Address blank"
		return nil
	}
	if len(q.Name) > 15 {
		*r = "q.Name too long"
		// NOTE: this behaviour was found when testing
		return nil
	}

	_, err := sm.client.Device(q.Name)
	if errors.Is(err, os.ErrNotExist) {
		log.Printf("※新たなデバイス：%s\n%s", q.Name, toString(q.Config))
		err = devAdd(q.Name, devConfig{
			Address:    q.Address,
			PrivateKey: q.Config.PrivateKey,
		})
		if err != nil {
			*r = fmt.Sprintf("devAdd: %s", err)
			return nil
		}
	} else if err != nil {
		*r = fmt.Sprintf("wg dev: %s", err)
		return nil
	} else {
		log.Printf("既存デバイス：%s\n%s", q.Name, toString(q.Config))
	}
	err = sm.client.ConfigureDevice(q.Name, *q.Config)
	if err != nil {
		*r = fmt.Sprintf("wg config: %s", err)
		return nil
	}
	*r = ""
	return nil
}

// blockNonLocal aborts connections from hosts that are not localhost.
//
// This is intended to be a last resort; it should not be relied on for
// security.
//
// Note: Server should be listening on 127.0.0.1, not 0.0.0.0.
func blockNonLocal(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.RemoteAddr, "127.0.0.1:") {
			log.Printf("blocked request from %s", r.RemoteAddr)
			http.Error(w, "", 403)
			return
		}
		handler.ServeHTTP(w, r)
	})
}
