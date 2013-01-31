// The MIT License (MIT)
// Copyright © 2013 Nils Maier <https://tn123.org>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the “Software”), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
// 
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package gosocksv5d

import "bytes"
import "encoding/binary"
import "errors"
import "fmt"
import "io"
import "net"
import "time"

const (
	bufSize     = 1 << 16
	timeoutDiff = 10 * time.Minute
)

var (
	ErrorHandshake  = errors.New("Handshake failed!")
	ErrorCommand    = errors.New("Invalid command!")
	ErrorAddress    = errors.New("Not addressable!")
	ErrorNotAllowed = errors.New("Destination not allowed")
)

const (
	protoVersion = 0x5

	atypeIPV4   = 0x1
	atypeIPV6   = 0x4
	atypeDomain = 0x3

	cmdConnect = 0x1
	cmdBind    = 0x2
	cmdAssoc   = 0x3

	repSuccess         = 0x0
	repFailure         = 0x1
	repNotAllowed      = 0x2
	repNetUnreachable  = 0x3
	repHostUnreachable = 0x4
	repRefused         = 0x5
	repTTL             = 0x6
	repNotSupported    = 0x7
	repNotAddressable  = 0x8
)

func timeout() time.Time {
	return time.Now().Add(timeoutDiff)
}

type sockConn struct {
	conn *net.TCPConn
	DNSResolver
	*prefixLogger
	Ruler
}

func newSockConn(conn *net.TCPConn, resolver DNSResolver, logger Logger, ruler Ruler) *sockConn {
	plog := &prefixLogger{fmt.Sprintf("[%v -> %v]", conn.LocalAddr(), conn.RemoteAddr()), logger}
	return &sockConn{conn, resolver, plog, ruler}
}

func (sock *sockConn) Read(b []byte) (int, error) {
	sock.conn.SetReadDeadline(timeout())
	return sock.conn.Read(b)
}

func (sock *sockConn) Write(b []byte) (int, error) {
	sock.conn.SetWriteDeadline(timeout())
	return sock.conn.Write(b)
}

func (sock *sockConn) String() string {
	return fmt.Sprintf("Sock: %v", sock.conn.RemoteAddr())
}

func (sock *sockConn) readAll(count uint32) []byte {
	rv := make([]byte, count)
	_, err := io.ReadFull(sock, rv)
	if err != nil && err != io.EOF {
		panic(err)
	}
	return rv
}

func (sock *sockConn) writeAll(bytes []byte) {
	n, err := sock.Write(bytes)
	if err != nil {
		panic(err)
	}
	if n != len(bytes) {
		panic(io.EOF)
	}
}

func (sock *sockConn) writeError(rsp byte, err error) {
	sock.writeAll([]byte{protoVersion, rsp, 0x0, atypeIPV4, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0})
	panic(err)
}

func (sock *sockConn) copyFrom(dst *sockConn, quit chan int) {
	defer func() {
		if err := recover(); err != nil && err != io.EOF {
			sock.Printf("Panic while copying streams, %v", err)
		}
		sock.Print("Closed one direction")
		sock.conn.CloseRead()
		dst.conn.CloseWrite()
		quit <- 1
	}()

	buf := make([]byte, bufSize)
	for {
		nr, err := sock.Read(buf)
		wbuf := buf
		for nr > 0 {
			nw, werr := dst.Write(wbuf[0:nr])
			nr -= nw
			wbuf = wbuf[nr:]
			if werr != nil {
				if ne, ok := werr.(net.Error); ok && (ne.Timeout() || ne.Temporary()) {
					continue
				}
				panic(werr)
			}
		}
		if err != nil {
			if ne, ok := err.(net.Error); ok && (ne.Timeout() || ne.Temporary()) {
				continue
			}
			panic(err)
		}
	}
}

func (sock *sockConn) handshake() {
	handshake := sock.readAll(2)
	if handshake[0] != protoVersion {
		panic(ErrorHandshake)
	}
	methods := sock.readAll(uint32(handshake[1]))
	switch {
	case bytes.IndexByte(methods, 0x0) >= 0:
		// No auth
		sock.writeAll([]byte{0x5, 0x0})
		sock.Printf("No auth OK")

	default:
		sock.writeAll([]byte{0x5, 0xff})
		panic(ErrorHandshake)
	}
}

func (sock *sockConn) IP() net.IP {
	raddr := sock.conn.RemoteAddr()
	switch addr := raddr.(type) {
	case *net.IPAddr:
		return addr.IP
	}
	return nil
}

func (sock *sockConn) connect(lip net.IP) *sockConn {
	command := sock.readAll(4)
	if command[0] != protoVersion {
		panic(ErrorHandshake)
	}
	switch command[1] {
	case cmdConnect:
		break

	default:
		sock.writeError(repNotSupported, ErrorCommand)
	}

	var rips []net.IP
	switch command[3] {
	case atypeIPV4:
		rawip := sock.readAll(4)
		rips = []net.IP{net.IPv4(rawip[0], rawip[1], rawip[2], rawip[3])}

	case atypeIPV6:
		rips = []net.IP{sock.readAll(net.IPv6len)}

	case atypeDomain:
		domain := string(sock.readAll(uint32(sock.readAll(1)[0])))
		var err error
		rips, err = sock.LookupIP(domain)
		if err != nil {
			sock.writeError(repNotAddressable, err)
		}

	default:
		sock.writeError(repNotAddressable, ErrorAddress)
	}

	port := int(binary.BigEndian.Uint16(sock.readAll(2)))
	rconn, err := func() (rconn *net.TCPConn, err error) {
		for _, rip := range rips {
			switch sock.ConnectionAllowed(sock.IP(), rip) {
			case AllowConnection:
				sock.Printf("Connecting: %v", rip)
			default:
				sock.Printf("Not allowed: %v", rip)
				sock.writeError(repNotAllowed, ErrorNotAllowed)
			}
			proto := "tcp"
			if rip.To4() == nil {
				proto = "tcp6"
			}
			laddr := &net.TCPAddr{lip, 0}
			raddr := &net.TCPAddr{rip, port}
			rconn, err = net.DialTCP(proto, laddr, raddr)
			if err == nil {
				return
			}
		}
		return
	}()

	if err != nil {
		switch err.(type) {
		case net.InvalidAddrError:
			sock.writeError(repNotAddressable, err)
		default:
			sock.writeError(repFailure, err)
		}
	}
	rsock := newSockConn(rconn, sock, sock.prefixLogger.Logger, sock)

	sock.writeAll([]byte{protoVersion, repSuccess, 0x0})
	if lip.To4() != nil {
		sock.writeAll([]byte{atypeIPV4})
		sock.writeAll(lip.To4())
	} else {
		sock.writeAll([]byte{atypeIPV6})
		sock.writeAll(lip.To16())
	}
	bport := []byte{0x0, 0x0}
	binary.BigEndian.PutUint16(bport, uint16(port))
	sock.writeAll(bport)

	return rsock
}

func (sock *sockConn) handle(lip net.IP) {
	defer func() {
		sock.conn.Close()
		if err := recover(); err != nil {
			sock.Printf("Panic while serving, %v", err)
			return
		}
		sock.Print("Done serving")
	}()
	sock.conn.SetNoDelay(true)

	sock.handshake()
	sock.Print("Handshake OK")

	rsock := sock.connect(lip)
	defer rsock.conn.Close()
	rsock.Print("Connected")

	quit := make(chan int)
	go sock.copyFrom(rsock, quit)
	go rsock.copyFrom(sock, quit)
	for i := 0; i < 2; i++ {
		<-quit
	}
}

// vim: set noet ts=2 sw=2:
