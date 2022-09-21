package node

/*

type Server struct {
	r            *mux.Router
	c            Config
	cc           CentralConfig
	coordPrivKey ed25519.PrivateKey
}

func (s *Server) doReq(r *http.Request) (*http.Response, error) {
	r.Header.Set("User-Agent", "QANMSClient/dev")
	client := &http.Client{}
	return client.Do(r)
}

func (s *Server) getCentralConfig() (*CentralConfig, error) {
	return nil, errors.New("not impl")
}

func (s *Server) sync() error {
	cc, err := s.getCentralConfig()
	if err != nil {
		return fmt.Errorf("get contral cofnig: %w", err)
	}

	for _, n := range cc.Networks {
		err := s.syncNetwork(n)
		if err != nil {
			return fmt.Errorf("sync network %d: %w", n.Name, err)
		}
	}
	return nil
}

func (s *Server) syncNetwork(cn CentralNetwork) error {
	myWGPrivKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return fmt.Errorf("gen privkey: %w", err)
	}
	cn.myPrivKey = myWGPrivKey

	var myI = -1
	for i, p := range cn.Peers {
		if p.PublicKey.Equal(s.coordPrivKey.Public()) {
			myI = i
		}
	}
	pcs := make([]wgtypes.PeerConfig, 0, len(cn.Peers))
	for i, p := range cn.Peers {
		if i == myI {
			continue
		}
		if i > myI {
			continue
		}
		pc, err := s.syncPeer(cn, p)
		if err != nil {
			// TODO: tolerate sync failures of some
			return fmt.Errorf("sync peer %d: %w", p.Name, err)
		}
		pcs = append(pcs, *pc)
	}
	ic := wgtypes.Config{
		PrivateKey:   &cn.myPrivKey,
		ListenPort:   &cn.ListenPort,
		ReplacePeers: true,
		Peers:        pcs,
	}
	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("new client: %w", err)
	}
	defer func() {
		err2 := client.Close()
		if err2 != nil {
			err = err2
		}
	}()
	devName := fmt.Sprintf("qnams-%s", cn.Name)
	err = client.ConfigureDevice(devName, ic)
	if err != nil {
		return fmt.Errorf("configure device: %w", err)
	}
	return nil
}

func genKeys(privKey wgtypes.Key) (*wgtypes.Key, *wgtypes.Key, error) {
	pubKey := privKey.PublicKey()
	psk, err := wgtypes.GenerateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("gen psk: %w", err)
	}
	return &pubKey, &psk, nil
}

func (s *Server) syncPeer(cn CentralNetwork, lrp CentralPeer) (*wgtypes.PeerConfig, error) {
	// NOTE: I am the higher-rank (HR) peer in this function.
	challenge, err := readRand(32)
	if err != nil {
		return nil, fmt.Errorf("gen chall: %w", err)
	}
	hrPubKey, hrPSK, err := genKeys(cn.myPrivKey)
	if err != nil {
		return nil, fmt.Errorf("gen keys: %w", err)
	}
	reqBuf := new(bytes.Buffer)
	req := exchangeReq{
		HRPubKey:   s.coordPrivKey.Public().(ed25519.PublicKey),
		Challenge:  challenge,
		HRWGPSK:    *hrPSK,
		HRWGPubKey: *hrPubKey,
	}
	err = json.NewEncoder(reqBuf).Encode(req)
	if err != nil {
		return nil, fmt.Errorf("encode req: %w", err)
	}

	r, err := http.NewRequest("POST", "/v1/exchange", reqBuf)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	q, _ := url.ParseQuery(r.URL.RawQuery)
	q.Add("nid", cn.Name)
	r.URL.RawQuery = q.Encode()
	r.Header.Set("User-Agent", "QANMSClient/dev")
	resp, err := s.doReq(r)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("request: invalid status %d / %s", resp.StatusCode, resp.Status)
	}
	var er exchangeResp
	err = json.NewDecoder(resp.Body).Decode(&er)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("request: decode: %w", err)
	}
	hrPC := wgtypes.PeerConfig{
		PublicKey:                   er.LRWGPubKey,
		Remove:                      false,
		UpdateOnly:                  false, // might be a new peer
		PresharedKey:                &er.LRWGPSK,
		PersistentKeepaliveInterval: &cn.Keepalive,
		ReplaceAllowedIPs:           true,
		AllowedIPs:                  lrp.AllowedIPs,
	}
	return &hrPC, nil
}
*/

/*
type NetworkID = string

type Network struct {
	Name        string
	UsableRange *net.IPNet
	UsableAddr  net.IP
	Tokens      [][]byte

	myPrivKey      wgtypes.Key
	stateLock      sync.Mutex
	stateUsedIPs   []net.IP
	stateLargestIP net.IP
}

type Config struct {
	PrivateKey ed25519.PrivateKey
	Networks   map[NetworkID]Network
}
*/

/*

type Config struct {
}

func New(c Config) (*Server, error) {
	s := &Server{
		r: mux.NewRouter(),
		c: c,
	}
	r2 := s.r.Methods("POST").Path("/v1").
		Queries("nid", "{nid}").
		Subrouter()
	r2.Use(s.restrictClient)
	r2.Use(s.network)
	r2.Path("/exchange").HandlerFunc(s.handleExchange)
	s.r.Methods("POST").Path("/net/{nid}/conn").Handler(s.restrictClient(http.HandlerFunc(s.handleConn)))
	return s, nil
}

type ctxKey int

const (
	ctxKeyInvalid ctxKey = iota
	ctxKeyNID
)

func (s *Server) restrictClient(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.UserAgent(), "QANMSClient") {
			http.Error(w, "only usable by QANMSClient", 403)
			return
		}
		handler.ServeHTTP(w, r)
	})
}
func (s *Server) network(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var nx int
		var cn CentralNetwork
		{
			vars := mux.Vars(r)
			// nx = network index
			nx2, err := strconv.ParseInt(vars["nx"], 10, 32)
			if err != nil {
				http.Error(w, "bad nx", 422)
				return
			}
			// nn = network name
			givenNN := vars["nx"]

			cn = s.cc.Networks[nx2]
			if int(nx) >= len(s.cc.Networks) {
				http.Error(w, "unknown nx", 422)
				return
			}
			expectedNN := cn.Name
			if expectedNN != givenNN {
				http.Error(w, "unknown nx", 422)
				return
			}
			nx = int(nx2)
		}

		/*
			{
				tokenAttempt := []byte(r.Header.Get("X-QANMS-Token"))
				tokenOk := false
				for _, token := range net.Tokens {
					tokenOk = bytes.Equal(tokenAttempt, token) || tokenOk
				}
				if !tokenOk {
					http.Error(w, "bad token", 403)
					return
				}
			}
*/
/*

		ctx := context.WithValue(context.Background(), ctxKeyNID, nx)
		r = r.WithContext(ctx)
		handler.ServeHTTP(w, r)
	})
}

func (s *Server) handleExchange(w http.ResponseWriter, r *http.Request) {
	// NOTE: I am the lower-rank (LR) peer in this function.
	nid := r.Context().Value(ctxKeyNID).(NetworkID)
	net := s.cc.Networks[nid]
	var req exchangeReq
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "decode json failed", 422)
		return
	}

	// ==Generate Keys==
	lrPubKey, lrPSK, err := genKeys(net.myPrivKey)
	if err != nil {
		http.Error(w, "generating keys failed", 500)
		return
	}

	// ==Challenge Response==
	var appendee []byte
	var challResp []byte
	{
		appendee, err = readRand(32)
		if err != nil {
			http.Error(w, "generating challenge appendee failed", 500)
			return
		}

		signThis := make([]byte, 64)
		copy(signThis, req.Challenge)
		signThis = append(signThis, appendee...)
		challResp = ed25519.Sign(s.coordPrivKey, signThis)
	}

	resp := exchangeResp{
		ChallengeResp:     challResp,
		ChallengeAppendee: appendee,
		LRWGPubKey:        *lrPubKey,
		LRWGPSK:           *lrPSK,
	}
	buf := new(bytes.Buffer)
	err = json.NewEncoder(buf).Encode(resp)
	if err != nil {
		http.Error(w, "enode json failed", 500)
		return
	}
	w.WriteHeader(200)
	_, err = io.Copy(w, buf)
	if err != nil {
		_, _ = w.Write([]byte("oops, copying data failed"))
		return
	}
}
func (s *Server) handleConn(w http.ResponseWriter, r *http.Request) {
	nid := r.Context().Value(ctxKeyNID).(NetworkID)
	net := s.c.Networks[nid]
}

func (s *Server) handleAdjust(w http.ResponseWriter, r *http.Request) {
}

type ClientConfig struct {
	InterfaceAddress    string
	InterfaceExtra      [][2]string
	InterfaceListenPort uint16
	Peers               []ClientPeer
}
type ServerConfig struct {
	NID    NetworkID
	Client ClientID
}

type ClientPeer struct {
	PublicKey  string
	AllowedIPs []string
	Endpoint   string
	Extra      [][2]string
}

func (s *Server) genCC(privateKey string) (ClientConfig, error) {
	clientIP, err := s.getNewIP()
	if err != nil {
		return ClientConfig{}, err
	}
	var cc ClientConfig
	cc.InterfaceAddress = clientIP

	me := ClientPeer{}

	return cc, nil
}
func (s *Server) getNewIP(n *Network) (net.IP, error) {
	n.stateLock.Lock()
	defer n.stateLock.Unlock()
	newIP2 := n.stateLargestIP
	var newIP net.IP
	if newIP2 == nil {
		newIP = n.UsableRange.IP
	} else {
		newIP = newIP2
	}
	last := newIP[len(newIP)-1]
	if last == 0xff {
		// last, round
		for j := len(newIP) - 2; j >= 0; j-- {
			if newIP[j] == 0xff {
				continue
			}
			newIP[j]++
			goto Success
		}
		return nil, errors.New("ran out of addresses to assign")
	Success:
	}
	newIP[len(newIP)-1]++
	if !n.UsableRange.Contains(newIP) {
		panic(fmt.Sprintf("ip %s not in range %s", newIP, n.UsableRange))
	}
	return newIP, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.r.ServeHTTP(w, r)
}
*/
