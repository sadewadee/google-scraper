package proxygate

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"

	"github.com/txthinking/socks5"
	"golang.org/x/net/proxy"
)

// Define constants that might be missing in the version of socks5 library used
const (
	socks5Ver5       = 0x05
	socks5MethodNone = 0x00
	socks5CmdConnect = 0x01
)

type Server struct {
	addr string
	pool *Pool
}

func NewServer(addr string, pool *Pool) *Server {
	return &Server{addr: addr, pool: pool}
}

func (s *Server) Run(ctx context.Context) error {
	socks5.Debug = true
	// We use socks5.NewClassicServer just to validate config if needed,
	// but we don't actually use the server instance because we want custom handling.
	// srv, _ := socks5.NewClassicServer(s.addr, "", "", "", 0, 0)

	go func() {
		<-ctx.Done()
	}()

	log.Printf("[ProxyGate] SOCKS5 server listening on %s", s.addr)

	// We can't easily replace the Handle method in the struct provided by this library
	// without reimplementing ListenAndServe or using a different library.
	// However, we can use the library's built-in TCP/UDP handling which is robust.
	// The library doesn't easily support dynamic upstream selection per request *inside* its standard handler.

	// Let's implement a custom listener loop using the library's components but our own Accept loop
	// to allow using our custom handleConnection.

	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				log.Printf("[ProxyGate] Accept error: %v", err)
				continue
			}
		}

		go func(c net.Conn) {
			if err := s.handleConnection(c); err != nil {
				// verbose logging only
			}
		}(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) error {
	// Re-implement basic SOCKS5 handshake since we are bypassing the library's main loop
	// to inject our dynamic upstream logic.

	// 1. Negotiation
	// +----+----------+----------+
	// |VER | NMETHODS | METHODS  |
	// +----+----------+----------+
	// | 1  |    1     | 1 to 255 |
	// +----+----------+----------+
	buf := make([]byte, 257)
	if _, err := io.ReadAtLeast(conn, buf, 2); err != nil {
		return err
	}
	if buf[0] != socks5Ver5 {
		return fmt.Errorf("unsupported version: %d", buf[0])
	}
	nMethods := int(buf[1])
	if _, err := io.ReadAtLeast(conn, buf, nMethods); err != nil {
		return err
	}

	// We only support NoAuth (0x00) for now
	if _, err := conn.Write([]byte{socks5Ver5, socks5MethodNone}); err != nil {
		return err
	}

	// 2. Request
	// +----+-----+-------+------+----------+----------+
	// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+

	// Use the library's Request struct if possible, but reading it manually is safer if we don't have a Server context
	// socks5.NewRequest(conn) is not available in all versions or signatures might vary.
	// Let's implement basic request reading manually to be safe and dependency-agnostic for this part.

	// +----+-----+-------+------+----------+----------+
	// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+

	header := make([]byte, 4)
	if _, err := io.ReadAtLeast(conn, header, 4); err != nil {
		return err
	}

	if header[0] != socks5Ver5 {
		return fmt.Errorf("unsupported version: %d", header[0])
	}

	cmd := header[1]
	// rsv := header[2]
	atyp := header[3]

	var dstAddr string
	var dstPort []byte

	switch atyp {
	case socks5.ATYPIPv4:
		ip := make([]byte, 4)
		if _, err := io.ReadAtLeast(conn, ip, 4); err != nil {
			return err
		}
		dstAddr = net.IP(ip).String()
	case socks5.ATYPDomain:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadAtLeast(conn, lenBuf, 1); err != nil {
			return err
		}
		domainLen := int(lenBuf[0])
		domain := make([]byte, domainLen)
		if _, err := io.ReadAtLeast(conn, domain, domainLen); err != nil {
			return err
		}
		dstAddr = string(domain)
	case socks5.ATYPIPv6:
		ip := make([]byte, 16)
		if _, err := io.ReadAtLeast(conn, ip, 16); err != nil {
			return err
		}
		dstAddr = net.IP(ip).String()
	default:
		return fmt.Errorf("unsupported address type: %d", atyp)
	}

	dstPort = make([]byte, 2)
	if _, err := io.ReadAtLeast(conn, dstPort, 2); err != nil {
		return err
	}
	portVal := int(dstPort[0])<<8 | int(dstPort[1])
	address := fmt.Sprintf("%s:%d", dstAddr, portVal)

	if cmd != socks5CmdConnect {
		// return r.Reply(socks5.RepCommandNotSupported, nil)
		// Manual reply
		conn.Write([]byte{socks5Ver5, socks5.RepCommandNotSupported, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return nil
	}

	// Retry mechanism: Try up to 3 different proxies if dialing fails
	var targetConn net.Conn
	var dialErr error

	for i := 0; i < 3; i++ {
		upstreamStr, err := s.pool.GetNext()
		if err != nil {
			log.Printf("[ProxyGate] No proxies available: %v", err)
			conn.Write([]byte{socks5Ver5, socks5.RepServerFailure, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
			return nil
		}

		targetConn, dialErr = s.dialUpstream(upstreamStr, address)
		if dialErr == nil {
			break
		}
	}

	if dialErr != nil {
		log.Printf("[ProxyGate] Upstream connection failed after retries: %v", dialErr)
		conn.Write([]byte{socks5Ver5, socks5.RepHostUnreachable, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return nil
	}
	defer targetConn.Close()

	// Reply Success
	// BIND.ADDR and BIND.PORT should be the server's address, but 0.0.0.0:0 is often accepted
	conn.Write([]byte{socks5Ver5, socks5.RepSuccess, 0x00, 0x01, 0, 0, 0, 0, 0, 0})

	// Bi-directional copy
	go func() {
		io.Copy(targetConn, conn)
	}()
	io.Copy(conn, targetConn)

	return nil
}



func (s *Server) dialUpstream(proxyURL, targetAddr string) (net.Conn, error) {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	var auth *proxy.Auth
	if u.User != nil {
		auth = &proxy.Auth{
			User: u.User.Username(),
		}
		if p, ok := u.User.Password(); ok {
			auth.Password = p
		}
	}

	// "tcp" is the network type for the proxy connection itself
	dialer, err := proxy.SOCKS5("tcp", u.Host, auth, proxy.Direct)
	if err != nil {
		return nil, err
	}

	return dialer.Dial("tcp", targetAddr)
}
