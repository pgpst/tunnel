package dns

import (
	"net"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/miekg/dns"
)

const ttl = 5 * 60

type DNS struct {
	Config *Config
	Redis  redis.Conn
}

func (d *DNS) handler(w dns.ResponseWriter, request *dns.Msg) {
	r := &dns.Msg{}
	r.SetReply(request)
	r.Authoritative = true

	for _, msg := range request.Question {
		answers := d.answer(msg)
		if len(answers) > 0 {
			r.Answer = append(r.Answer, answers...)
		} else {
			r.Ns = append(r.Ns, d.soa())
		}
	}
	w.WriteMsg(r)
}

func (d *DNS) answer(msg dns.Question) []dns.RR {
	answers := []dns.RR{}

	if msg.Qtype == dns.TypeNS {
		if msg.Name == d.Config.Domain {
			answers = append(answers, &dns.NS{
				Hdr: dns.RR_Header{Name: msg.Name, Rrtype: dns.TypeNS, Class: dns.ClassINET, Ttl: 300},
				Ns:  d.Config.Hostname,
			})
		}
		return answers
	}

	if msg.Qtype == dns.TypeSOA {
		if msg.Name == d.Config.Domain {
			answers = append(answers, d.soa())
		}
		return answers
	}

	if msg.Qtype == dns.TypeCNAME {
		result, err := redis.String(d.Redis.Do("GET", "cname:"+msg.Name))
		if err != nil {
			return answers
		}
		return append(answers, &dns.CNAME{
			Hdr:    dns.RR_Header{Name: msg.Name, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: ttl},
			Target: result,
		})
	} else if msg.Qtype == dns.TypeA {
		result, err := redis.String(d.Redis.Do("GET", "a:"+msg.Name))
		if err != nil {
			return answers
		}
		return append(answers, &dns.A{
			Hdr: dns.RR_Header{Name: msg.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl},
			A:   net.ParseIP(result),
		})
	}

	return answers
}

func (d *DNS) soa() dns.RR {
	return &dns.SOA{
		Hdr:     dns.RR_Header{Name: d.Config.Domain, Rrtype: dns.TypeSOA, Class: dns.ClassINET, Ttl: 60},
		Ns:      d.Config.Hostname,
		Mbox:    d.Config.Email,
		Serial:  uint32(time.Now().Unix()),
		Refresh: 86400,
		Retry:   7200,
		Expire:  86400,
		Minttl:  60,
	}
}

func (d *DNS) ListenAndServe() error {
	udpErr := make(chan error)
	tcpErr := make(chan error)
	go func() {
		udpErr <- dns.ListenAndServe(d.Config.Bind, "udp", dns.HandlerFunc(d.handler))
	}()
	go func() {
		tcpErr <- dns.ListenAndServe(d.Config.Bind, "tcp", dns.HandlerFunc(d.handler))
	}()

	select {
	case err := <-udpErr:
		return err
	case err := <-tcpErr:
		return err
	}

	return nil
}
