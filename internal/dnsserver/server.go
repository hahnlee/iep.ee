package dnsserver

import (
	"fmt"
	"github.com/billopark/iep.ee/config"
	"github.com/billopark/iep.ee/internal/record"
	"github.com/miekg/dns"
	"log"
	"net"
)

func handleNS(m *dns.Msg) {
	for ns, ip := range config.Get().Nss {
		m.Answer = append(m.Answer, record.BuildNS(ns))
		r, _ := record.BuildA(ns, net.ParseIP(ip)) // TODO(billopark): error handle
		m.Extra = append(m.Extra, r)
	}
}

func handler(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	switch r.Question[0].Qtype {
	case dns.TypeTXT:
		m.Answer = append(m.Answer, record.BuildTXT())

	case dns.TypeNS:
		handleNS(m)
		m.Extra = append(m.Extra, record.BuildTXT())

	default:
		fallthrough
	case dns.TypeAAAA, dns.TypeA:
		rr, err := record.BuildA(r.Question[0].Name, nil)
		if err != nil {
			return
		}
		m.Answer = append(m.Answer, rr)

	case dns.TypeSOA:
		soa, err := record.BuildSOA()
		if err != nil {
			return
		}
		m.Answer = append(m.Answer, soa)
		m.Extra = append(m.Extra, record.BuildTXT())
	}

	log.Println(m)
	err := w.WriteMsg(m)
	if err != nil {
		log.Println(err)
	}
}

func Start(halt chan bool) {
	dns.HandleFunc(config.Get().Domain, handler)
	server := &dns.Server{
		Addr: fmt.Sprintf("[::]:%d", config.Get().DnsPort),
		Net:  "udp",
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			fmt.Printf("Failed to setup the udp dnsserver: %s\n", err.Error())
		}
	}()

	go func() {
		<-halt
		if err := server.Shutdown(); err != nil {
			panic(err)
		}
	}()

	return
}