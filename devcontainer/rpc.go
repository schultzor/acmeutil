package main

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
)

type Control int

// send a new command to the container
func (c *Control) Exec(cmd string, response *string) error {
	log.Println("exec", cmd)
	return nil
}

func sendRpc(sock, cmd string) {
	client, err := rpc.DialHTTP("unix", sock)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	var res string
	if err := client.Call("Control.Exec", cmd, &res); err != nil {
		log.Fatal("call error:", err)
	}
	log.Println("sent", cmd, "to", sock)
}

func rpcListener(sock string, c *Control) {
	log.Println("listening on", sock)
	rpc.Register(c)
	rpc.HandleHTTP()
	l, e := net.Listen("unix", sock)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}
