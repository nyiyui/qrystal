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

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func init() {
	gob.Register(new(ConfigureDeviceQ))
	gob.Register(new(RemoveDeviceQ))
	gob.Register(new(ForwardingQ))
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

	handler := guard(rs)

	lis, addr, err := listen()
	if err != nil {
		log.Fatalf("バインド: %s", err)
	}
	fmt.Printf("addr:%s\n", addr)
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
	Token    []byte // put token here for simplicity
	Name     string
	Config   *wgtypes.Config
	Address  []net.IPNet
	PostUp   string
	PostDown string
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
	} else if err != nil {
		*r = fmt.Sprintf("wg dev: %s", err)
		return nil
	} else {
		err = devRemove(q.Name)
		if err != nil {
			*r = fmt.Sprintf("devRemove2: %s", err)
			return nil
		}
		log.Printf("既存デバイス：%s\n%s", q.Name, wgConfigToString(q.Config))
	}
	if q.Config.ListenPort == nil {
		*r = "nil ListenPort"
		return nil
	}
	err = devAdd(q.Name, devConfig{
		Address:    q.Address,
		PrivateKey: q.Config.PrivateKey,
		Peers:      q.Config.Peers,
		ListenPort: uint(*q.Config.ListenPort),
	})
	if err != nil {
		*r = fmt.Sprintf("devAdd: %s", err)
		return nil
	}
	err = sm.client.ConfigureDevice(q.Name, *q.Config)
	if err != nil {
		*r = fmt.Sprintf("wg config: %s", err)
		return nil
	}
	*r = ""
	return nil
}

type ForwardingQ struct {
	Token  []byte // put token here for simplicity
	Type   ForwardingType
	Enable bool
}

type ForwardingType uint8

const (
	ForwardingTypeInvalid ForwardingType = iota
	ForwardingTypeIPv4
)
