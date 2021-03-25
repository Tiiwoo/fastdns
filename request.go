package fastdns

import (
	"errors"
	"sync"
)

// Request represents an DNS request received by a server or to be sent by a client.
type Request struct {
	/*
		Header encapsulates the construct of the header part of the DNS
		query message.
		It follows the conventions stated at RFC1035 section 4.1.1.


		The header contains the following fields:

						0  1  2  3  4  5  6  7
		      0  1  2  3  4  5  6  7  8  9  A  B  C  D  E  F
		    +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
		    |                      ID                       |
		    +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
		    |QR|   Opcode  |AA|TC|RD|RA|   Z    |   RCODE   |
		    +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
		    |                    QDCOUNT                    |
		    +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
		    |                    ANCOUNT                    |
		    +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
		    |                    NSCOUNT                    |
		    +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
		    |                    ARCOUNT                    |
		    +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	*/
	Header struct {

		// ID is an arbitrary 16bit request identifier that is
		// forwarded back in the response so that we can match them up.
		ID uint16

		// QR is an 1bit flag specifying whether this message is a query (0)
		// of a response (1)
		// 1bit
		QR byte

		// Opcode is a 4bit field that specifies the query type.
		// Possible values are:
		// 0		- standard query		(QUERY)
		// 1		- inverse query			(IQUERY)
		// 2		- server status request		(STATUS)
		// 3 to 15	- reserved for future use
		Opcode Opcode

		// AA indicates whether this is an (A)nswer from an (A)uthoritative
		// server.
		// Valid in responses only.
		// 1bit.
		AA byte

		// TC indicates whether the message was (T)run(C)ated due to the length
		// being grater than the permitted on the transmission channel.
		// 1bit.
		TC byte

		// RD indicates whether (R)ecursion is (D)esired or not.
		// 1bit.
		RD byte

		// RA indidicates whether (R)ecursion is (A)vailable or not.
		// 1bit.
		RA byte

		// Z is reserved for future use
		Z byte

		// RCODE contains the (R)esponse (CODE) - it's a 4bit field that is
		// set as part of responses.
		RCODE Rcode

		// QDCOUNT specifies the number of entries in the question section
		QDCount uint16

		// ANCount specifies the number of resource records (RR) in the answer
		// section
		ANCount uint16

		// NSCount specifies the number of name server resource records in the
		// authority section
		NSCount uint16

		// ARCount specifies the number of resource records in the additional
		// records section
		ARCount uint16
	}

	/*
	     0  1  2  3  4  5  6  7  8  9  0  1  2  3  4  5
	   +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	   |                                               |
	   /                     QNAME                     /
	   /                                               /
	   +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	   |                     QTYPE                     |
	   +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	   |                     QCLASS                    |
	   +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	*/
	Question struct {
		// QName refers to the raw query name to be resolved in the query.
		Name []byte

		// QTYPE specifies the type of the query to perform.
		Type Type

		// QCLASS
		Class Class
	}
}

// GetDomainName returns the client's normalized domain name.
func (req *Request) GetDomainName() string {
	return string(decodeQName(make([]byte, 0, 256), req.Question.Name))
}

// AppendDomainName appends the normalized domain name to dst and returns the resulting dst.
func (req *Request) AppendDomainName(dst []byte) []byte {
	return decodeQName(dst, req.Question.Name)
}

var (
	// ErrInvalidHeader is returned when dns message does not have the expected header size.
	ErrInvalidHeader = errors.New("dns message does not have the expected header size")
	// ErrInvalidQuestion is returned when dns message does not have the expected question size.
	ErrInvalidQuestion = errors.New("dns message does not have the expected question size")
)

// ParseRequest parses dns request from payload into dst and returns the error.
func ParseRequest(dst *Request, payload []byte) error {
	if len(payload) < 12 {
		return ErrInvalidHeader
	}

	_ = payload[11]

	// ID
	dst.Header.ID = uint16(payload[1]) | uint16(payload[0])<<8

	// RD, TC, AA, Opcode, QR
	b := payload[2]
	dst.Header.RD = b & 0b00000001
	dst.Header.TC = (b >> 1) & 0b00000001
	dst.Header.AA = (b >> 2) & 0b00000001
	dst.Header.Opcode = Opcode((b >> 3) & 0b00001111)
	dst.Header.QR = (b >> 7) & 0b00000001

	// RA, Z, RCODE
	b = payload[3]
	dst.Header.RCODE = Rcode(b & 0b00001111)
	dst.Header.Z = (b >> 4) & 0b00000111
	dst.Header.RA = (b >> 7) & 0b00000001

	// QDCOUNT, ANCOUNT, NSCOUNT, ARCOUNT
	dst.Header.QDCount = uint16(payload[4])<<8 | uint16(payload[5])
	dst.Header.ANCount = uint16(payload[6])<<8 | uint16(payload[7])
	dst.Header.NSCount = uint16(payload[8])<<8 | uint16(payload[9])
	dst.Header.ARCount = uint16(payload[10])<<8 | uint16(payload[11])

	if dst.Header.QDCount != 1 {
		return ErrInvalidHeader
	}

	// QNAME
	payload = payload[12:]
	var i int
	for i, b = range payload {
		if b == 0 {
			break
		}
	}
	if i+5 > len(payload) {
		return ErrInvalidQuestion
	}
	dst.Question.Name = append(dst.Question.Name[:0], payload[:i+1]...)

	// QTYPE, QCLASS
	payload = payload[i:]
	dst.Question.Class = Class(uint16(payload[4]) | uint16(payload[3])<<8)
	dst.Question.Type = Type(uint16(payload[2]) | uint16(payload[1])<<8)

	return nil
}

// AppendRequest appends the dns request to dst and returns the resulting dst.
func AppendRequest(dst []byte, req *Request) []byte {
	var header [12]byte

	// ID
	header[0] = byte(req.Header.ID >> 8)
	header[1] = byte(req.Header.ID & 0xff)

	// QR :		0
	// Opcode:	1 2 3 4
	// AA:		5
	// TC:		6
	// RD:		7
	b := req.Header.QR << (7 - 0)
	b |= byte(req.Header.Opcode) << (7 - (1 + 3))
	b |= req.Header.AA << (7 - 5)
	b |= req.Header.TC << (7 - 6)
	b |= req.Header.RD
	header[2] = b

	// second 8bit part of the second row
	// RA:		0
	// Z:		1 2 3
	// RCODE:	4 5 6 7
	b = req.Header.RA << (7 - 0)
	b |= req.Header.Z << (7 - 1)
	b |= byte(req.Header.RCODE) << (7 - (4 + 3))
	header[3] = b

	// QDCOUNT
	header[4] = byte(req.Header.QDCount >> 8)
	header[5] = byte(req.Header.QDCount & 0xff)
	// ANCOUNT
	header[6] = byte(req.Header.ANCount >> 8)
	header[7] = byte(req.Header.ANCount & 0xff)
	// NSCOUNT
	header[8] = byte(req.Header.NSCount >> 8)
	header[9] = byte(req.Header.NSCount & 0xff)
	// ARCOUNT
	header[10] = byte(req.Header.ARCount >> 8)
	header[11] = byte(req.Header.ARCount & 0xff)

	dst = append(dst, header[:]...)

	// question
	if req.Header.QDCount != 0 {
		// QNAME
		dst = append(dst, req.Question.Name...)
		// QTYPE
		dst = append(dst, byte(req.Question.Type>>8), byte(req.Question.Type&0xff))
		// QCLASS
		dst = append(dst, byte(req.Question.Class>>8), byte(req.Question.Class&0xff))
	}

	return dst
}

var reqPool = sync.Pool{
	New: func() interface{} {
		req := new(Request)
		req.Question.Name = make([]byte, 0, 256)
		return req
	},
}

// AcquireRequest returns new dns request.
func AcquireRequest() *Request {
	return reqPool.Get().(*Request)
}

// ReleaseRequest returnes the dns request to the pool.
func ReleaseRequest(req *Request) {
	reqPool.Put(req)
}
