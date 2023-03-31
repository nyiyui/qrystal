package csadmin

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

var tokenHash *string

func init() {
	addTokenCmd := flag.NewFlagSet("token-add", flag.ExitOnError)
	commands["token-add"] = command{
		FS:      addTokenCmd,
		Handler: tokenAddMain,
	}
	rmTokenCmd := flag.NewFlagSet("token-rm", flag.ExitOnError)
	commands["token-rm"] = command{
		FS:      rmTokenCmd,
		Handler: tokenRemoveMain,
	}
	tokenHash = rmTokenCmd.String("token-hash", "", "hash of token to remove (starts with qrystalcth_)")
}

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
		Hash:      &q.Hash,
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

func tokenAddMain() {
	var q AddTokenQ
	err := json.NewDecoder(os.Stdin).Decode(&q)
	if err != nil {
		log.Fatalf("unmarshal config: %s", err)
	}

	buf := new(bytes.Buffer)
	err = json.NewEncoder(buf).Encode(convQ(q))
	if err != nil {
		panic(fmt.Sprintf("json marshal failed: %s", err))
	}
	log.Printf("payload: %s", buf)
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
		Timeout: 5 * time.Second,
	}
	if certPath != nil && *certPath != "" {
		cl.Transport = newTLSTransport(*certPath)
	}
	hs, err := cl.Do(hq)
	if err != nil {
		log.Fatalf("request: %s", err)
	}
	body, err := io.ReadAll(hs.Body)
	if err != nil {
		log.Fatalf("response body read: %s", err)
	}
	if hs.StatusCode != 200 {
		log.Fatalf("response status: %s %s", hs.Status, body)
	}
	log.Printf("response: %s", body)
}

func tokenRemoveMain() {
	log.Printf("%s", *tokenHash)
	tokenHash, err := util.ParseTokenHash(*tokenHash)
	if err != nil {
		log.Fatalf("parse token hash: %s", err)
	}
	buf, err := json.Marshal(api.HRemoveTokenQ{
		Hash: tokenHash,
	})
	if err != nil {
		panic(fmt.Sprintf("json marshal failed: %s", err))
	}
	log.Printf("payload: %s", buf)
	u := url.URL{
		Scheme: "https",
		Host:   *serverAddr,
		Path:   "/remove-token",
	}
	hq, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(buf))
	if err != nil {
		panic(fmt.Sprintf("NewRequest: %s", err))
	}
	hq.Header.Set("X-Qrystal-CentralToken", ct.String())
	cl := &http.Client{
		Timeout: 5 * time.Second,
	}
	if *certPath != "" {
		cl.Transport = newTLSTransport(*certPath)
	}
	hs, err := cl.Do(hq)
	if err != nil {
		log.Fatalf("request: %s", err)
	}
	body, err := io.ReadAll(hs.Body)
	if err != nil {
		log.Fatalf("response body read: %s", err)
	}
	if hs.StatusCode != 200 {
		log.Fatalf("response status: %s %s", hs.Status, body)
	}
	log.Printf("response: %s", body)
}
