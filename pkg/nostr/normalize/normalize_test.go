package normalize

import (
	"fmt"
	"testing"
)

func TestURL(t *testing.T) {
	fmt.Println(URL(""))
	fmt.Println(URL("wss://x.com/y"))
	fmt.Println(URL("wss://x.com/y/"))
	fmt.Println(URL("http://x.com/y"))
	fmt.Println(URL(URL("http://x.com/y")))
	fmt.Println(URL("wss://x.com"))
	fmt.Println(URL("wss://x.com/"))
	fmt.Println(URL(URL(URL("wss://x.com/"))))
	fmt.Println(URL("x.com"))
	fmt.Println(URL("x.com/"))
	fmt.Println(URL("x.com////"))
	fmt.Println(URL("x.com/?x=23"))

	// Output:
	//
	// wss://x.com/y
	// wss://x.com/y
	// ws://x.com/y
	// ws://x.com/y
	// wss://x.com
	// wss://x.com
	// wss://x.com
	// wss://x.com
	// wss://x.com
	// wss://x.com
	// wss://x.com?x=23
}
