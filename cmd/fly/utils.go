package main

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"net"

	"github.com/ngagnon/flybywire/internal/wire"
)

func connect(host string, disableTls bool) (net.Conn, error) {
	if disableTls {
		return net.Dial("tcp", host)
	} else {
		tlsConfig := tls.Config{
			InsecureSkipVerify: true,
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				return verifyPeerCertificate(host, rawCerts[0])
			},
		}

		return tls.Dial("tcp", host, &tlsConfig)
	}
}

func sendCommand(conn net.Conn, name string, args ...interface{}) wire.Value {
	values := make([]wire.Value, len(args)+1)
	values[0] = wire.NewString(name)

	for i, arg := range args {
		j := i + 1

		switch v := arg.(type) {
		case wire.Value:
			values[j] = v
		case string:
			values[j] = wire.NewString(v)
		default:
			log.Fatalf("Unsupported array value: %v", arg)
		}
	}

	cmd := wire.NewArray(values)
	err := cmd.WriteTo(conn)

	if err != nil {
		log.Fatalf("Failed to write to socket: %v\n", err)
	}

	r, err := wire.ReadValue(conn)

	if err != nil {
		log.Fatalf("Failed to read from socket: %v\n", err)
	}

	return r
}
