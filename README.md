gosocksv5d
==========
A SOCKS v5 (RFC 1928) server implementation in Go.

# Quick Start
  * Install Go (obviously)
  * `go get -u "github.com/nmaier/gosocksv5d"`
  * Implement your server
    ```go
package main
import "github.com/nmaier/gosocksv5d"
func main() {
	server := gosocksv5d.NewServer()
	server.ListenAndServe(net.IPv4zero, 12345) // Never returns
}
 ```

# Links
## Go language
http://golang.org/
## SOCKSv5 RFC 1928
http://www.ietf.org/rfc/rfc1928.txt
## Full online docs
http://go.pkgdoc.org/github.com/nmaier/gosocksv5d

# LICENSE
MIT-License; see LICENSE file
