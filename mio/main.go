// Package mio provides a local RPC server to configure WireGuard. Its purpose
// is to try to reduce the number of programs running with high priviledges.
package mio

import (
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

type RemoveDeviceQ struct {
	Token []byte // put token here for simplicity
	Name  string
}

func (sm *Mio) RemoveDevice(q RemoveDeviceQ, r *string) error {
	sm.ensureTokenOk(q.Token)

	if q.Name == "" {
		return errors.New("q.Name blank")
	}
	if len(q.Name) > 15 {
		return errors.New("q.Name too long")
	}
	_, err := sm.client.Device(q.Name)
	if errors.Is(err, os.ErrNotExist) {
		*r = "デバイスは無い"
		return nil
	} else if err != nil {
		*r = fmt.Sprintf("wg dev: %s", err)
		return nil
	}
	log.Printf("デバイスを削除：%s", q.Name)
	err = devRemove(q.Name)
	if err != nil {
		*r = fmt.Sprintf("devRemove: %s", err)
		return nil
	}
	*r = ""
	return nil
}

type ConfigureDeviceQ struct {
	Token   []byte // put token here for simplicity
	Name    string
	Config  *wgtypes.Config
	Address []net.IPNet
}

// TODO: allow removing devices

func (sm *Mio) ConfigureDevice(q ConfigureDeviceQ, r *string) error {
	sm.ensureTokenOk(q.Token)

	// These errors shouldn't happen but this way, but this is easier to debug.
	// TODO: consider returning an error (instead of setting *r) for these types of errors.
	if q.Name == "" {
		return errors.New("q.Name blank")
	}
	if q.Config == nil {
		return errors.New("q.Config nil")
	}
	if len(q.Address) == 0 {
		return errors.New("q.Address blank")
	}
	if len(q.Name) > 15 {
		return errors.New("q.Name too long")
	}

	_, err := sm.client.Device(q.Name)
	if errors.Is(err, os.ErrNotExist) {
		log.Printf("デバイスを追加：%s\n%s", q.Name, wgConfigToString(q.Config))
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
		log.Printf("既存デバイス：%s\n%s", q.Name, wgConfigToString(q.Config))
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
