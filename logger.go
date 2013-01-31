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

import "fmt"
import "log"
import "os"

var (
	// Logger using os.StdErr output.
	StdErrorLogger = log.New(os.Stderr, "", log.LstdFlags)
	// Default Logger, using StdErrLogger with a socksv5d message prefix.
	DefaultLogger = NewPrefixLogger("socksv5d", StdErrorLogger)
	// Blackhole Logger, not logging anything (silent)
	NullLogger = &nullLogger{}
)

// Logger implements a subset of log.Logger.
// Different Loggers may specify different output destinations or formats.
type Logger interface {
	Output(calldepth int, s string) error
	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

type prefixLogger struct {
	prefix string
	Logger
}

// Wraps another Logger, such as StdErrorLogger, adding a prefix to logged
// messages.
//
// Don't confuse this with log.Logger.Prefix(): Message prefixes
// are independent from that.
func NewPrefixLogger(prefix string, logger Logger) Logger {
	return &prefixLogger{prefix, logger}
}

func (self *prefixLogger) Output(calldepth int, s string) error {
	return self.Logger.Output(calldepth, fmt.Sprintf("%s - %s", self.prefix, s))
}
func (self *prefixLogger) Print(v ...interface{}) {
	self.Output(2, fmt.Sprint(v...))
}
func (self *prefixLogger) Printf(format string, v ...interface{}) {
	self.Output(2, fmt.Sprintf(format, v...))
}
func (self *prefixLogger) Println(v ...interface{}) {
	self.Output(2, fmt.Sprintln(v...))
}

type nullLogger struct{}

func (self *nullLogger) Output(calldepth int, s string) error {
	return nil
}
func (self *nullLogger) Print(v ...interface{})                 {}
func (self *nullLogger) Printf(format string, v ...interface{}) {}
func (self *nullLogger) Println(v ...interface{})               {}

// vim: set noet ts=2 sw=2:
