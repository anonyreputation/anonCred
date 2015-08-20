package client
import (
	"github.com/dedis/crypto/abstract"
	"net"
)


// data structure to store all the necessary data in client

type DissentClient struct {
	// client-side config
	CoordinatorAddr *net.UDPAddr
	Socket *net.UDPConn
	Status int
	// crypto variables
	Suite abstract.Suite
	PrivateKey abstract.Secret
	PublicKey abstract.Point
	OnetimePseudoNym abstract.Point
	G abstract.Point
}