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

import "net"

var (
	// The DefaultRuler implements an access rule set that will only allow
	// non-local connections. Non-local also excludes the subnets of all
	// network interfaces.
	DefaultRuler Ruler = &defaultRuler{}
)

type RulerResult int

const (
	DenyConnection  RulerResult = iota // Ruler allows this connection
	AllowConnection                    // Ruler denies this connection
)

// Ruler implements access rule sets.
// Each connection attempt will check the Ruler whether this connection should be allowed or not.
type Ruler interface {
	// Requestee is allowed to connect to the request IP via a socksv5d server.
	ConnectionAllowed(requestee, requested net.IP) RulerResult
}

type defaultRuler struct{}

func (self *defaultRuler) ConnectionAllowed(requestee, requested net.IP) RulerResult {
	if !requested.IsGlobalUnicast() {
		return DenyConnection
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return DenyConnection
	}
	for _, addr := range addrs {
		switch ipa := addr.(type) {
		case *net.IPAddr:
			if ipa.IP.Equal(requested) {
				return DenyConnection
			}
		case *net.IPNet:
			if ipa.Contains(requested) {
				return DenyConnection
			}
		}
	}
	return AllowConnection
}
