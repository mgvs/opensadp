package sadp

import (
	"encoding/xml"
	"errors"
	"io"
	"net"
	"strconv"
	"time"
)

type Message struct {
	XMLName xml.Name `xml:"Probe"`
	Uuid    string   `xml:"Uuid"`
	MAC     string   `xml:"MAC"`
	Types   string   `xml:"Types"`
}

type BasicDictObject map[string]string

func UnmarshalResponse(buf []byte) (BasicDictObject, error) {
	// Capture only immediate children of the root element as key/value pairs.
	type kv struct {
		XMLName xml.Name
		Value   string `xml:",chardata"`
	}
	type root struct {
		XMLName xml.Name
		Fields  []kv `xml:",any"`
	}
	var r root
	if err := xml.Unmarshal(buf, &r); err != nil {
		return nil, err
	}
	out := make(BasicDictObject, len(r.Fields))
	for _, f := range r.Fields {
		out[f.XMLName.Local] = f.Value
	}
	// Normalize common field name variants observed in different payloads
	if v, ok := out["Ipv4Address"]; ok {
		out["IPv4Address"] = v
	}
	if v, ok := out["Ipv4SubnetMask"]; ok {
		out["IPv4SubnetMask"] = v
	}
	if v, ok := out["Ipv4Gateway"]; ok {
		out["IPv4Gateway"] = v
	}
	return out, nil
}

// SADPClient is a simple UDP multicast client
type SADPClient struct {
	conn     *net.UDPConn
	addr     *net.UDPAddr
	bufSize  int
	deadline time.Duration
}

func NewClient(port int, timeout time.Duration) (*SADPClient, error) {
	if port == 0 {
		port = 37020
	}
	addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort("239.255.255.250", itoa(port)))
	if err != nil {
		return nil, err
	}
	c, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, err
	}
	_ = c.SetWriteBuffer(1 << 20)
	_ = c.SetReadBuffer(1 << 20)
	return &SADPClient{conn: c, addr: addr, bufSize: 2048, deadline: timeout}, nil
}

func (c *SADPClient) Close() error { return c.conn.Close() }

func (c *SADPClient) WriteMessage(m Message) (int, error) {
	data, err := xml.Marshal(m)
	if err != nil {
		return 0, err
	}
	payload := append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>"), data...)
	return c.conn.WriteToUDP(payload, c.addr)
}

func (c *SADPClient) ReceiveOnce() ([]byte, *net.UDPAddr, error) {
	if c.deadline > 0 {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.deadline))
	}
	buf := make([]byte, c.bufSize)
	n, addr, err := c.conn.ReadFromUDP(buf)
	if err != nil {
		return nil, nil, err
	}
	return buf[:n], addr, nil
}

// Helper: avoid bytes.NewReader allocation when we only need io.Reader
// and keep code compact without importing bytes directly in many places.
type noCopyReader struct {
	b []byte
	i int
}

func bytesNewReaderNoCopy(b []byte) *noCopyReader { return &noCopyReader{b: b} }

func (r *noCopyReader) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, ioEOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	if n == 0 {
		return 0, ioEOF
	}
	return n, nil
}

// small local aliases to keep imports tidy
var (
	ioEOF    = io.EOF
	errorsIs = errors.Is
)

func itoa(i int) string { return strconvItoa(i) }

// Import wrappers
// We alias to avoid exporting extra std imports at top; keeps file tidy.
var (
	strconvItoa = strconv.Itoa
)
