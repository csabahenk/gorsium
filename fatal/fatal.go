/* Copyright (c) 2011 Sonia Keys
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

// Cf. http://soniacodes.wordpress.com/2011/04/28/deferred-functions-and-an-exit-code/

package fatal

import(
	"fmt"
	"os"
)

type fatal struct {
	err interface{}
}

// Fail terminates the program on given error
// while properly invoking deferred actions
func Fail(err interface{}) {
	panic(fatal{err})
}

// Deferring to HandleFatal() should be the first line of your main() func
// in order to make Fail() work
func HandleFatal() {
	if err := recover(); err != nil {
		if f, ok := err.(fatal); ok {
			fmt.Println(f.err)
			os.Exit(1)
		}
		panic(err)
	}
}
