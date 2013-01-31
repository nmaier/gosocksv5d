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

/*
Package gosocksv5d implements a SOCKS v5 server.

The server supports a subset of RFC 1928:
 - Only "No Authentication" auth method
 - Only "Connect" command
 - All defined address types: IPv4, IPv6, domain name

Domain names will be resolved using the specified or default resolver
(net.LookupIP).

Examples:
	server := gosocksv5d.NewServer()
	server.SetDNSResolver(myResolver)
	server.ListenAndServe(net.IPv4zero, 12345) // Never returns
*/
package gosocksv5d

import "errors"
import "net"

var (
	ErrorAlreadyListening = errors.New("Already listening")
)

// Server implements a socks v5 server.
type Server interface {
	// Starts a new server. The server will bind to the provided IP and port.
	// Once running, the call will never return, so you better call this from a
	// goroutine.
	ListenAndServe(ip net.IP, port int) error

	// Set a new DNS resolver, in case you don't like the default one.
	// See: gosocksv5d.DefaultResolver
	// Attempting to set this after calling ListenAndServer will panic()
	SetDNSResolver(resolver DNSResolver)

	// Set a new Logger.
	// See: gosocksv5d.DefaultLogger.
	// Attempting to set this after calling ListenAndServer will panic()
	SetLogger(logger Logger)

	// Set a new Ruler.
	// See: gosocksv5d.DefaultRuler.
	// Attempting to set this after calling ListenAndServer will panic()
	SetRuler(ruler Ruler)

	// Stops the server again from accepting new connections.
	// Already accepted connection will still be served!
	Stop()

	// Allows the server to accept new connections (again).
	// You don't need to Continue() after ListenAndServe().
	Continue()
}

type connChan chan *net.TCPConn
type boolChan chan bool

type server struct {
	running   boolChan
	instances int
	DNSResolver
	Logger
	Ruler
}

// Creates a new server.
// Afterwards, set up the instance as desired in terms of logger, resolver, etc.
// Then call ListenAndServe()
func NewServer() Server {
	return &server{make(boolChan, 1), 0, DefaultResolver, DefaultLogger, DefaultRuler}
}

func (self *server) listen(c connChan, ip net.IP, port int) (l net.Listener, err error) {
	proto := "tcp"
	if ip.To4() == nil {
		proto = "tcp6"
	}
	l, err = net.ListenTCP(proto, &net.TCPAddr{ip, int(port)})
	if err == nil {
		go func() {
			for {
				conn, err := l.Accept()
				if err != nil {
					if ne, ok := err.(net.Error); ok && ne.Temporary() {
						self.Printf("Error while accepting: %v", err)
						continue
					}
				}
				tconn, ok := conn.(*net.TCPConn)
				if !ok {
					self.Print("Failed to accept; not tcp")
					conn.Close()
					continue
				}
				c <- tconn
			}
		}()
	}
	return
}

func (self *server) ListenAndServe(ip net.IP, port int) error {
	conns := make(connChan, 10)

	var l net.Listener
	var err error

	self.Printf("Starting sock server for %v:%d", ip, port)
	l, err = self.listen(conns, ip, port)
	if err != nil {
		return err
	}
	self.instances++

	for {
		select {
		case running := <-self.running:
			switch {
			case !running && l != nil:
				l.Close()
				l = nil
				self.instances--

			case running && l == nil:
				l, err = self.listen(conns, ip, port)
				if err != nil {
					return err
				}
				self.instances++
			}
		case conn := <-conns:
			sock := newSockConn(conn, self, self, self)
			go sock.handle(ip)
		}
	}
	panic("Not reached!")
}

func (self *server) panicIfListening() {
	if self.instances > 0 {
		panic(ErrorAlreadyListening)
	}
}

func (self *server) SetDNSResolver(resolver DNSResolver) {
	self.panicIfListening()
	self.DNSResolver = shuffleResolver{resolver}
}

func (self *server) SetLogger(logger Logger) {
	self.panicIfListening()
	self.Logger = logger
}

func (self *server) SetRuler(ruler Ruler) {
	self.panicIfListening()
	self.Ruler = ruler
}

func (self *server) Continue() {
	for i := 0; i < self.instances; i++ {
		self.running <- true
	}
}

func (self *server) Stop() {
	for i := 0; i < self.instances; i++ {
		self.running <- false
	}
}

// vim: set noet ts=2 sw=2:
