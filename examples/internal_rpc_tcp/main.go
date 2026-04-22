// Package main demonstrates the internal RPC TCP preset client.
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/arcgolabs/clientx/preset"
	clienttcp "github.com/arcgolabs/clientx/tcp"
)

type tcpDialCloser interface {
	Dial(ctx context.Context) (net.Conn, error)
	Close() error
}

func main() {
	listener := listenTCP()
	defer closeWithLog("TCP listener", listener)

	serverErr := startTCPAckServer(listener)

	client := newInternalRPCClient(listener.Addr().String())
	defer closeWithLog("internal RPC client", client)

	reply := sendTCPPing(client)
	printTCPReply(reply)

	if err := <-serverErr; err != nil {
		panic(err)
	}
}

func listenTCP() net.Listener {
	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	return listener
}

func startTCPAckServer(listener net.Listener) <-chan error {
	serverErr := make(chan error, 1)

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer closeWithLog("accepted TCP connection", conn)

		reply, err := readTCPLine(conn)
		if err != nil {
			serverErr <- err
			return
		}
		serverErr <- writeTCPAck(conn, reply)
	}()

	return serverErr
}

func newInternalRPCClient(address string) tcpDialCloser {
	client, err := preset.NewInternalRPC(
		clienttcp.Config{Address: address},
		preset.WithInternalRPCDisableRetry(),
		preset.WithInternalRPCTimeoutGuard(2*time.Second),
	)
	if err != nil {
		panic(err)
	}

	return client
}

func sendTCPPing(client tcpDialCloser) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := client.Dial(ctx)
	if err != nil {
		panic(err)
	}
	defer closeWithLog("TCP client connection", conn)

	if _, err = conn.Write([]byte("ping\n")); err != nil {
		panic(err)
	}

	reply, err := readTCPLine(conn)
	if err != nil {
		panic(err)
	}

	return reply
}

func readTCPLine(conn net.Conn) (string, error) {
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read TCP line: %w", err)
	}

	return line, nil
}

func writeTCPAck(conn net.Conn, line string) error {
	if _, err := conn.Write([]byte("ack:" + strings.TrimSpace(line) + "\n")); err != nil {
		return fmt.Errorf("write TCP ack: %w", err)
	}

	return nil
}

func printTCPReply(reply string) {
	if _, err := fmt.Printf("tcp reply=%q\n", strings.TrimSpace(reply)); err != nil {
		log.Printf("print TCP reply: %v", err)
	}
}

func closeWithLog(name string, closer interface{ Close() error }) {
	if err := closer.Close(); err != nil {
		log.Printf("close %s: %v", name, err)
	}
}
