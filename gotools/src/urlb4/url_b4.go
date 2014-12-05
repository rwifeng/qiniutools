package main

import (
	"encoding/base64"
	"flag"
	"fmt"
)

func main() {

	d := flag.Bool("d", false, "url safe base64 decode.")
	flag.Parse()

	args := flag.Args()

	if len(args) != 1 {
		fmt.Println("usage:\n ./urlb4 <data> \n ./urlb4 -d <data>")
		return
	}

	if *d {
		ret, err := base64.URLEncoding.DecodeString(args[0])
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(string(ret))
		}
	} else {
		ens := base64.URLEncoding.EncodeToString([]byte(args[0]))
		fmt.Println(ens)
	}

}
