package main

import (
	"fmt"
	"net"
	"./proto"
	 "./util"
	"./client"
	"strconv"
	"github.com/dedis/crypto/nist"
	"github.com/dedis/crypto/random"
	"bufio"
	"os"
	"strings"
	"log"
	"time"

)

// pointer to client itself
var dissentClient  *client.DissentClient

/**
  * register itself to controller
  */
func register() {
	// set the parameters to register
	bytePublicKey, _ := dissentClient.PublicKey.MarshalBinary()
	params := map[string]interface{}{
		"public_key": bytePublicKey,
	}
	event := &proto.Event{proto.CLIENT_REGISTER_CONTROLLERSIDE,params}

	util.SendToCoodinator(dissentClient.Socket,util.Encode(event))
}

/**
  * start listener to handle event
  */
func startClientListener() {
	fmt.Println("[debug] Client Listener started...");
	buf := make([]byte, 4096)
	for {
		n,addr,err := dissentClient.Socket.ReadFromUDP(buf)
		if err != nil {
			log.Fatal(err)
		}
		client.Handle(buf,addr,dissentClient,n) // a goroutine handles conn so that the loop can accept other connections
	}
}

/**
  * send message text to server
  */
func sendMsg(text string) {
	sendSigRequest(text,proto.MESSAGE)
}

/**
  * send general request to server
  * the request is encrypted by signature
  */
func sendSigRequest(text string, eventType int) {
	// generate signature
	rand := dissentClient.Suite.Cipher([]byte("example"))
	sig := util.ElGamalSign(dissentClient.Suite,rand,[]byte(text),dissentClient.PrivateKey,dissentClient.G)
	// serialize Point data structure
	byteNym, _ := dissentClient.OnetimePseudoNym.MarshalBinary()
	// wrap params
	params := map[string]interface{}{
		"text": text,
		"nym":byteNym,
		"signature":sig,
	}
	event := &proto.Event{eventType,params}
	// send to coordinator
	util.SendToCoodinator(dissentClient.Socket,util.Encode(event))
}

/**
  * send vote to server
  */
func sendVote(msgID, vote int) {
	// vote can be only 1 or -1
	if vote > 0 {
		vote = 1;
	}else {
		vote = -1;
	}
	v := strconv.Itoa(vote)
	m := strconv.Itoa(msgID)
	text :=  m + ";" + v
	sendSigRequest(text,proto.VOTE)
}


/**
  * initialize anonClient and encrypted parameters
  */
func initServer() {
	// load controller ip and port
	config := util.ReadConfig()
	ServerAddr,err := net.ResolveUDPAddr("udp",config["coordinator_ip"]+":"+ config["coordinator_port"])
	util.CheckErr(err)
	// initialize suite
	suite := nist.NewAES128SHA256QR512()
	a := suite.Secret().Pick(random.Stream)
	A := suite.Point().Mul(nil, a)
	dissentClient = &client.DissentClient{ServerAddr,nil,client.CONFIGURATION,suite,a,A,suite.Point(),nil}
}


func main() {
	// initialize parameters and server configurations
	initServer()
	fmt.Println("[debug] Client started...");
	// make tcp connection to controller
	conn, err := net.DialUDP("udp", nil, dissentClient.CoordinatorAddr)
	util.CheckErr(err)
	// set socket
	dissentClient.Socket = conn
	// start Listener
	go startClientListener()
	fmt.Println("[debug] My public key is: ")
	fmt.Println(dissentClient.PublicKey)
	// register itself to controller
	register()

	// wait until register successful
	for ; dissentClient.Status != client.MESSAGE ; {
		time.Sleep(500 * time.Millisecond)
	}

	// read command and process
	reader := bufio.NewReader(os.Stdin)
	Loop:
	for {
		data, _, _ := reader.ReadLine()
		command := string(data)
		commands := strings.Split(command," ")
		switch commands[0] {
		case "msg":
			sendMsg(commands[1]);
			break;
		case "vote":
			msgID,_ := strconv.Atoi(commands[1])
			vote, _ := strconv.Atoi(commands[2])
			sendVote(msgID,vote)
			break;
		case "exit":
			break Loop
		}
	}
	// close connection
	conn.Close()
	fmt.Println("[debug] Exit system...");
}