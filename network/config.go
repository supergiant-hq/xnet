package network

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/logging"
)

const (
	connectionIDLength         = 16
	ConnectionTimeout          = 5 * time.Second
	ConnectionHandshakeTimeout = 5 * time.Second
	RequestTimeout             = 10 * time.Second
)

// Config Interface
type Config interface {
	// Maximum number of concurrent streams open on connection
	MaxStreams() int16
	// Use QUIC Datagram
	// This model does not guarantee a packet will be delivered
	UseDatagram() bool
}

// Generate QUIC Config with defaults
func GenerateQuicConfig(c Config) *quic.Config {
	return &quic.Config{
		KeepAlive:             true,
		Versions:              []quic.VersionNumber{quic.Version1},
		ConnectionIDLength:    connectionIDLength,
		MaxIdleTimeout:        ConnectionTimeout,
		HandshakeIdleTimeout:  ConnectionHandshakeTimeout,
		MaxIncomingStreams:    int64(c.MaxStreams()),
		MaxIncomingUniStreams: int64(c.MaxStreams()),
		EnableDatagrams:       c.UseDatagram(),
		Tracer:                logging.NewMultiplexedTracer(),
	}
}

// Generate TLS Config with defaults
func GenerateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}

	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}

	return &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{tlsCert},
		NextProtos:         []string{"network-proto-v1"},
	}
}
