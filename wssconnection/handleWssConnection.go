package wssconnection

import (
	"crypto/tls"

	"log"

	"fmt"

	"github.com/google/uuid"
	"golang.org/x/net/websocket"
)

/*
Connecting to EG2s

params:

	eg2sHost: Host of eg2s (only host name, or IP)
	username: username
	password: password
*/
func Connect(eg2sHost string, username string, password string) (*websocket.Conn, error) {
	// Because of Eg2s's self signed certificate this is needed for direct connection
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	config, err := websocket.NewConfig(fmt.Sprintf("wss://%s/eg", eg2sHost), "http://localhost")
	if err != nil {
		log.Printf("ERROR: Error when setting NewConfig in Connect %v\n", err)
		return nil, err
	}
	config.TlsConfig = tlsConfig

	conn, err := websocket.DialConfig(config)
	if err != nil {
		log.Printf("ERROR: Error when DialConfig in Connect %v\n", err)
		return nil, err
	} else {
		log.Printf("INFO: Connected to EG2s")
	}
	sendLoginMessage(conn, username, password)
	SendMonitorMessage(conn)
	return conn, nil
}

/*
Sending Login Message, Eg2s requires this before making any usable methods visible (it will answer invalid method for existing methods as well, before getting login)

params:

	connection: The WSS connection
	username: username
	password: password
*/
func sendLoginMessage(connection *websocket.Conn, username string, password string) {
	msg := LoginMessage{
		JsonRpc: "2.0",
		Id:      uuid.New().String(),
		Method:  "LogIn",
		Params: LoginParams{
			Username: username,
			Password: password,
		},
	}
	err := websocket.JSON.Send(connection, msg)
	if err != nil {
		fmt.Printf("ERROR: Error when Sending Login Message: %v\n", err)
	}
}

/*
Send Monitor Message to be notified of changes

params:

	connection: The WSS connection
*/
func SendMonitorMessage(connection *websocket.Conn) {
	msg := MonitorMessage{
		JsonRpc: "2.0",
		Id:      uuid.New().String(),
		Method:  "Monitor",
		Params: MonitorParams{
			Predicate: PredicateType{
				Type: "True",
			},
		},
	}
	if err := websocket.JSON.Send(connection, msg); err != nil {
		log.Printf("ERROR: Error when sending monitor request %v\n", err)
	}
}

/*
Disconnect from EG2s
*/
func Disconnect(connection *websocket.Conn) error {
	if err := connection.Close(); err != nil {
		return err
	}
	return nil
}
