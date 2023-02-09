package main

import (
	"fmt"
	"log"

	"github.com/miekg/dns"
)

// ~~stolen~~ copied from <https://gist.github.com/walm/0d67b4fb2d5daf3edd4fad3e13b162cb>.

var records = map[string]string{
	"example.com.": "192.168.3.9",
}

func parseQuery(m *dns.Msg) {
	for _, q := range m.Question {
		log.Printf("query for %s", q.Name)
		switch q.Qtype {
		case dns.TypeA:
			ip := records[q.Name]
			if ip != "" {
				rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ip))
				if err == nil {
					m.Answer = append(m.Answer, rr)
				}
			}
		}
	}
}

func handleDnsRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)

	switch r.Opcode {
	case dns.OpcodeQuery:
		parseQuery(m)
	}

	w.WriteMsg(m)
}

const port = 53530

func main() {
	dns.HandleFunc(".", handleDnsRequest)
	server := &dns.Server{Addr: fmt.Sprintf(":%d", port), Net: "udp"}
	log.Printf("listening on :%d", port)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("serve failed: %s\n ", err)
	}
}
