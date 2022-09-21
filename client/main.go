package client

import "net/http"

func setUA(r *http.Request) {
	r.Header.Set("User-Agent", "QANMSClient/dev")
}
