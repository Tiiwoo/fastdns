## Usage

```go
import (
	"log"
	"net"
	"os"

	"github.com/phuslu/fastdns"
)

type DNSHandler struct {
	Debug bool
}

func (h *DNSHandler) ServeDNS(rw fastdns.ResponseWriter, req *fastdns.Request) {
	qname := req.GetQName()
	if h.Debug {
		log.Printf("addr=%s qname=%s req=%+v\n", rw.RemoteAddr().String(), qname, req)
	}

	if req.Question.QType != fastdns.QTypeA {
		fastdns.Error(rw, req, fastdns.NXDOMAIN)
		return
	}

	fastdns.HostRecord(rw, req, []net.IP{net.IP{8, 8, 8, 8}}, 300)
}

func main() {
	server := &fastdns.Server{
		Handler: &DNSHandler{
			Debug: os.Getenv("DEBUG") != "",
		},
		Logger:       log.New(os.Stderr, "", 0),
		HTTPPortBase: 9000,
	}

	server.ListenAndServe(":53")
}
```
