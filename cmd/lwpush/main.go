package main

import (
	"bytes"
	"cbor"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/dustin/go-coap"
)

var host string
var port int
var endpoint string

func main() {

	flag.StringVar(&host, "s", "qa.airvantage.io", "Server host")
	flag.IntVar(&port, "p", 5685, "Server port")
	flag.StringVar(&endpoint, "e", "lwpush", "LWM2M endpoint")

	flag.Usage = func() {
		fmt.Printf("Usage:\n")
		fmt.Printf("  lwpush [-options] jsonpayload\n")
		fmt.Printf("Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	jsonpayload := flag.Args()[0]

	rand.Seed(time.Now().Unix())
	msgID := rand.Intn(10000)

	// binding to a UDP socket
	laddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		log.Fatal(err)
	}
	c, err := net.ListenUDP("udp", laddr)
	if err != nil {
		log.Fatal(err)
	}

	// server address
	uaddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		log.Fatal(err)
	}

	// Send register request
	registerMsg := coap.Message{
		Type:      coap.Confirmable,
		Code:      coap.POST,
		MessageID: uint16(msgID),
		Payload:   []byte("</3/0>"),
	}
	registerMsg.SetPathString("/rd")
	registerMsg.AddOption(coap.URIQuery, fmt.Sprintf("ep=%s", endpoint))

	err = coap.Transmit(c, uaddr, registerMsg)
	if err != nil {
		log.Fatalf("Error while sending registration request: %v", err)
	}
	buf := make([]byte, 1500)
	rv, err := coap.Receive(c, buf)
	if err != nil {
		log.Fatalf("Error while receiving registration response: %v", err)
	}
	if &rv != nil {
		//log.Printf("Ack received: %v", &rv)
		log.Printf("Registered with id: %s", rv.Options(coap.LocationPath)[1])
	}

	// send push
	b := []byte(jsonpayload)
	var f interface{}
	err = json.Unmarshal(b, &f)
	if err != nil {
		log.Fatalf("Invalid json payload: %v", err)
	}

	var buffTest bytes.Buffer
	encoder := cbor.NewEncoder(&buffTest)
	ok, err := encoder.Marshal(f)
	if !ok {
		log.Fatalf("CBOR encoding failure: %v", err)
	}

	log.Printf("Pushing payload: %s", hex.EncodeToString(buffTest.Bytes()))
	push := coap.Message{
		Type:      coap.Confirmable,
		Code:      coap.POST,
		MessageID: uint16(msgID + 1),
		Payload:   buffTest.Bytes(),
	}
	push.SetPathString("/push")
	push.AddOption(coap.URIQuery, fmt.Sprintf("ep=%s", endpoint))
	push.AddOption(coap.ContentFormat, 60)

	err = coap.Transmit(c, uaddr, push)
	if err != nil {
		log.Fatalf("Error while sending push request: %v", err)
	}
	buf = make([]byte, 1500)
	rv, err = coap.Receive(c, buf)
	if err != nil {
		log.Fatalf("Error while receiving push response: %v", err)
	}

	if &rv != nil {
		log.Printf("Push response: %v", &rv)
	}
}
