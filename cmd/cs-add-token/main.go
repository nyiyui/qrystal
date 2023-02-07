package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/nyiyui/qrystal/api"
	"github.com/nyiyui/qrystal/util"
)

type AddTokenQ struct {
	Overwrite bool           `json:"overwrite"`
	Name      string         `json:"name"`
	Hash      util.TokenHash `json:"hash"`
	CanPull   *struct {
		Networks map[string]string `json:"networks"`
	} `json:"canPull"`
	CanPush *struct {
		Networks map[string]Peer `json:"networks"`
	} `json:"canPush"`
}

type Peer struct {
	Name          string   `json:"name"`
	CanSeeElement []string `json:"canSeeElement"`
}

func newTLSTransport(certPath string) *http.Transport {
	pool := x509.NewCertPool()
	cert, err := os.ReadFile(certPath)
	if err != nil {
		log.Fatalf("read cert: %s", err)
	}
	ok := pool.AppendCertsFromPEM(cert)
	if !ok {
		log.Fatal("append certs failed")
	}
	return &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: pool,
		},
	}
}

func convQ(q AddTokenQ) api.HAddTokenQ {
	q2 := api.HAddTokenQ{
		Overwrite: q.Overwrite,
		Hash:      q.Hash,
		Name:      q.Name,
	}
	if q.CanPull != nil {
		q2.CanPull = q.CanPull.Networks
	}
	if q.CanPush != nil {
		q2.CanPush = map[string]api.CanNetwork{}
		for cnn, nc := range q.CanPush.Networks {
			q2.CanPush[cnn] = api.CanNetwork{
				PeerName:        nc.Name,
				CanSeeElementOf: nc.CanSeeElement,
			}
		}
	}
	return q2
}

func main() {
	serverAddr := flag.String("server", "", "server address")
	ctRaw := flag.String("token", "", "central token")
	certPath := flag.String("cert", "", "path to server cert")
	flag.Parse()

	ct, err := util.ParseToken(*ctRaw)
	if err != nil {
		log.Fatalf("parse token: %s", err)
	}

	var q AddTokenQ
	err = json.NewDecoder(os.Stdin).Decode(&q)
	if err != nil {
		log.Fatalf("unmarshal config: %s", err)
	}

	buf := new(bytes.Buffer)
	err = json.NewEncoder(buf).Encode(convQ(q))
	if err != nil {
		panic(fmt.Sprintf("json marshal failed: %s", err))
	}
	u := url.URL{
		Scheme: "https",
		Host:   *serverAddr,
		Path:   "/add-token",
	}
	hq, err := http.NewRequest("POST", u.String(), buf)
	if err != nil {
		panic(fmt.Sprintf("NewRequest: %s", err))
	}
	hq.Header.Set("X-Qrystal-CentralToken", ct.String())
	cl := &http.Client{
		Transport: newTLSTransport(*certPath),
		Timeout:   5 * time.Second,
	}
	hs, err := cl.Do(hq)
	if err != nil {
		log.Fatalf("request: %s", err)
	}
	if hs.StatusCode != 200 {
		log.Fatalf("response status: %s", hs.Status)
	}
	body, err := io.ReadAll(hs.Body)
	if err != nil {
		log.Fatalf("response body read: %s", err)
	}
	log.Printf("response: %s", body)
}
