// Package main demonstrates the low-latency UDP preset client.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/arcgolabs/clientx/preset"
	clientudp "github.com/arcgolabs/clientx/udp"
)

type udpDialCloser interface {
	Dial(ctx context.Context) (net.Conn, error)
	Close() error
}

func main() {
	server := listenUDP()
	defer closePacketConnWithLog("UDP server", server)

	serverErr := startUDPAckServer(server)

	client := newLowLatencyUDPClient(server.LocalAddr().String())
	defer closeWithLog("UDP client", client)

	reply := sendUDPPing(client)
	printUDPReply(reply)

	if err := <-serverErr; err != nil {
		panic(err)
	}
}

func listenUDP() net.PacketConn {
	var listenConfig net.ListenConfig

	server, err := listenConfig.ListenPacket(context.Background(), "udp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	return server
}

func startUDPAckServer(server net.PacketConn) <-chan error {
	serverErr := make(chan error, 1)

	go func() {
		buf := make([]byte, 128)
		n, addr, err := server.ReadFrom(buf)
		if err != nil {
			serverErr <- err
			return
		}
		_, err = server.WriteTo([]byte("ack:"+string(buf[:n])), addr)
		serverErr <- err
	}()

	return serverErr
}

func newLowLatencyUDPClient(address string) udpDialCloser {
	client, err := preset.NewLowLatencyUDP(
		clientudp.Config{Address: address},
		preset.WithLowLatencyUDPReadTimeout(500*time.Millisecond),
		preset.WithLowLatencyUDPWriteTimeout(500*time.Millisecond),
		preset.WithLowLatencyUDPTimeoutGuard(700*time.Millisecond),
	)
	if err != nil {
		panic(err)
	}

	return client
}

func sendUDPPing(client udpDialCloser) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := client.Dial(ctx)
	if err != nil {
		panic(err)
	}
	defer closeWithLog("UDP connection", conn)

	if _, err = conn.Write([]byte("ping")); err != nil {
		panic(err)
	}

	buf := make([]byte, 128)
	n, err := conn.Read(buf)
	if err != nil {
		panic(err)
	}

	return string(buf[:n])
}

func printUDPReply(reply string) {
	if _, err := fmt.Printf("udp reply=%q\n", strings.TrimSpace(reply)); err != nil {
		log.Printf("print UDP reply: %v", err)
	}
}

func closePacketConnWithLog(name string, conn net.PacketConn) {
	if err := conn.Close(); err != nil {
		log.Printf("close %s: %v", name, err)
	}
}

func closeWithLog(name string, closer interface{ Close() error }) {
	if err := closer.Close(); err != nil {
		log.Printf("close %s: %v", name, err)
	}
}
