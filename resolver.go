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

import "math/rand"
import "net"

var (
	// Default resolver, simply wrapping net.LookupIP().
	DefaultResolver = &defaultResolver{}
)

// Generic DNS Resolver
type DNSResolver interface {
	// Looks up the host, returning one or more IPv4 or IPv6 addresses on success.
	// See: net.LookupIP().
	LookupIP(host string) (addrs []net.IP, err error)
}

type defaultResolver struct{}

func (self defaultResolver) LookupIP(host string) (addrs []net.IP, err error) {
	return net.LookupIP(host)
}

type shuffleResolver struct {
	resolver DNSResolver
}

func (self shuffleResolver) LookupIP(host string) (addrs []net.IP, err error) {
	addrs, err = self.resolver.LookupIP(host)
	if err == nil {
		for n := len(addrs); n > 1; n-- {
			if r := rand.Intn(n + 1); r != n {
				addrs[r], addrs[n] = addrs[n], addrs[r]
			}
		}
	}
	return
}

// vim: set noet ts=2 sw=2:
