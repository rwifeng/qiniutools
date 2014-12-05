package main

import (
	"os"
	"fmt"
	"encoding/json"
	"encoding/base64"
	upapi "qbox.us/api/up"
)

func main() {

	if len(os.Args) != 2 {
		fmt.Println("usage: ./decode_uptoken <data>")
		return
	}

	auth := upapi.AuthPolicy{}

	upToken := os.Args[1]
	data, err := base64.URLEncoding.DecodeString(upToken)
	if err != nil {
		fmt.Println(err)
	}
	err = json.Unmarshal(data, &auth)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("ctx:\n", string(data))
	fmt.Println("authPolicy info:")
	fmt.Println("Scope: ", auth.Scope)
	fmt.Println("CallbackUrl: ", auth.CallbackUrl)
	fmt.Println("CallbackBodyType: ", auth.CallbackBodyType)
	fmt.Println("CallbackBody: ", auth.CallbackBody)
	fmt.Println("Customer: ", auth.Customer)
	fmt.Println("Deadline: ", auth.Deadline)
	fmt.Println("ReturnBody: ", auth.ReturnBody)
	fmt.Println("PersistentOps: ", auth.PersistentOps)
	fmt.Println("PersistentNotifyUrl: ", auth.PersistentNotifyUrl)
}
