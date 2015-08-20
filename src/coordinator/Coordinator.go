package coordinator
import (
	"github.com/dedis/crypto/abstract"
	"net"
)

type Coordinator struct {
	// local address
	LocalAddr *net.UDPAddr
	// socket
	Socket *net.UDPConn
	// network topology for server cluster
	ServerList []*net.UDPAddr
	// initialize the controller status
	Status int


	// crypto things
	Suite abstract.Suite
	// private key
	PrivateKey abstract.Secret
	// public key
	PublicKey abstract.Point
	// generator g
	G abstract.Point

	// store client address
	Clients map[string]*net.UDPAddr
	// store reputation map
	ReputationKeyMap map[string]abstract.Point
	ReputationMap map[string][]byte
	// we only add new clients at the beginning of each round
	// store the new clients's one-time pseudo nym
	NewClientsBuffer []abstract.Point
	// msg sender's record nym
	MsgLog []abstract.Point

	DecryptedReputationMap map[string]int
	DecryptedKeysMap map[string]abstract.Point



}

// get last server in topology
func (c *Coordinator) GetLastServer() *net.UDPAddr {
	if len(c.ServerList) == 0 {
		return nil
	}
	return c.ServerList[len(c.ServerList)-1]
}

// get first server in topology
func (c *Coordinator) GetFirstServer() *net.UDPAddr {
	if len(c.ServerList) == 0 {
		return nil
	}
	return c.ServerList[0]
}

func (c *Coordinator) AddClient(key abstract.Point, val *net.UDPAddr) {
	// delete the client who has same ip address
	for k,v := range c.Clients {
		if v.String() == val.String() {
			delete(c.Clients,k)
			break
		}
	}
	c.Clients[key.String()] = val
}

// add server into topology
func (c *Coordinator) AddServer(addr *net.UDPAddr){
	c.ServerList = append(c.ServerList,addr)
}

// add msg log and return msg id
func (c *Coordinator) AddMsgLog(log abstract.Point) int{
	c.MsgLog = append(c.MsgLog,log)
	return len(c.MsgLog)
}

// get reputation
func (c *Coordinator) GetReputation(key abstract.Point) int{
	return c.DecryptedReputationMap[key.String()]
}

func (c *Coordinator) AddClientInBuffer(nym abstract.Point) {
	c.NewClientsBuffer = append(c.NewClientsBuffer, nym)
}

func (c *Coordinator) AddIntoDecryptedMap(key abstract.Point, val int) {
	keyStr := key.String()
	c.DecryptedKeysMap[keyStr] = key
	c.DecryptedReputationMap[keyStr] = val
}

func (c *Coordinator) AddIntoRepMap(key abstract.Point, val []byte) {
	keyStr := key.String()
	c.ReputationKeyMap[keyStr] = key
	c.ReputationMap[keyStr] = val
}

