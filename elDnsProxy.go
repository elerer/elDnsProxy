package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/miekg/dns"
)

/* A Simple function to verify error */
func CheckError(err error) {
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(0)
	}
}

type proxyDns struct {
	conn         *net.UDPConn
	upAddr       *net.UDPAddr
	sem          chan int
	maxParrallel int
	bufs         [][]byte
}

func (pd *proxyDns) proxyDns(addr *net.UDPAddr, payLoad string) {
	//Get index of pre allocated buffer to use
	counter := <-pd.sem
	bufIndex := counter % pd.maxParrallel
	fmt.Println("-----Performing - ", len(pd.sem), " in parrallel, index is [", bufIndex, "]-\n")
	buf := pd.bufs[bufIndex]

	//buf := make([]byte, 1024)

	upStreamDns, err := net.DialUDP("udp", nil, pd.upAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error:%s", err.Error())
		os.Exit(1)
	}
	upStreamDns.SetDeadline(time.Now().Add(time.Millisecond * 1000))
	defer upStreamDns.Close()

	//fmt.Print("will resolve ", payLoad)
	_, err = upStreamDns.Write([]byte(payLoad))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR ", err.Error())
		return
	}
	n, err := upStreamDns.Read(buf[:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR ", err.Error())
		return
	}

	go printResponse(buf[:n])
	pd.conn.WriteToUDP(buf[0:n], addr)
	fmt.Println("------------PERFORMED ", counter, " RESOLUTIONS--------------\n")

}

func printResponse(m []byte) {
	var dm dns.Msg
	err := dm.Unpack(m[:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR :%s", err.Error())
		os.Exit(1)
	}
	fmt.Printf("resolved answer %s ", dm.String())
}

func main() {
	/* Lets prepare a address at any address at port 10001*/
	ServerAddr, err := net.ResolveUDPAddr("udp", ":53")
	CheckError(err)

	/* Now listen at selected port */
	ServerConn, err := net.ListenUDP("udp", ServerAddr)
	CheckError(err)
	defer ServerConn.Close()

	buf := make([]byte, 1024)

	var pd proxyDns
	pd.maxParrallel = 15
	pd.conn = ServerConn
	pd.sem = make(chan int, pd.maxParrallel)
	pd.upAddr, err = net.ResolveUDPAddr("udp4", "1.1.1.1:53")
	pd.bufs = make([][]byte, pd.maxParrallel)
	for i := range pd.bufs {
		pd.bufs[i] = make([]byte, 1024)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error:%s", err.Error())
		os.Exit(1)
	}
	var counter int = 0
	for {
		n, addr, err := ServerConn.ReadFromUDP(buf)
		payl := string(buf[0:n])
		fmt.Println("Received ", payl, " from ", addr)

		if err != nil {
			fmt.Println("Error: ", err)
		}
		pd.sem <- counter
		go pd.proxyDns(addr, payl)
		counter++
	}
}
