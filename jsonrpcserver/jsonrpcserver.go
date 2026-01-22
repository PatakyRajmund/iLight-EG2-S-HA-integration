package jsonrpcserver

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"host/lamp/wssconnection"

	"fmt"

	"sync"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"golang.org/x/net/websocket"
)

/*Global variables, for synchronization of multiple goroutines*/
var done = make(chan bool)
var brightness = make(chan int)
var wg *sync.WaitGroup
var turnedOff = &SharedFlag{}

type SharedFlag struct {
	flag  bool
	mutex sync.Mutex
}

// The struct of the used JSON-RPC implementation
type LampServerHandler struct {
	channelLevels   []int
	ha_host         string
	webhook_id      string
	num_of_channels int
	username        string
	password        string
	eg2sHost        string
	connection      *websocket.Conn
}

/*
Setup method, this is called by HomeAssistant in the start. The params should be configured in configuration.yaml
*/
func (h *LampServerHandler) Setup(username string, password string, eg2sHost string, ha_host string, webhook_id string, num_of_channels int, reset_time int) (string, error) {
	_, err := h.Connect(eg2sHost, username, password)
	if err != nil {
		return "", &jsonrpc.JSONRPCError{
			Code:    -32000,
			Message: "Unable to connect to EG2-S",
		}
	}
	h.num_of_channels = num_of_channels
	h.channelLevels = make([]int, h.num_of_channels)
	h.ha_host = ha_host
	h.webhook_id = webhook_id
	h.username = username
	h.password = password
	h.eg2sHost = eg2sHost

	if reset_time != -1 {
		c := cron.New()
		c.AddFunc(fmt.Sprintf("0 %d * * *", reset_time), func() { refreshConnection(h) })
		c.Start()
	}
	log.Println("INFO: Connection set up.")
	return "ok", nil
}

/*
Setting scene on the EG2S instance, this is called by HomeAssistant when using the set_scene service

params:

	scene: the number of scene needed (turn off is usually 0)
	transitionTime: time to transition to the set scene, I haven't figured out how it's scaling. It being set to 70 is around 3s
*/
func (h *LampServerHandler) SetScene(scene int, transitionTime int) (string, error) {
	msg := TransmitMessage{
		JsonRpc: "2.0",
		Id:      uuid.New().String(),
		Method:  "Transmit",
		Params: TransmitParams{
			Id:   1001,
			Data: []int{0, 1, 1, scene, transitionTime},
		},
	}
	if err := websocket.JSON.Send(h.connection, msg); err != nil {
		log.Printf("ERROR: Error when sending SetScene request %v\n", err)
		return "", &jsonrpc.JSONRPCError{
			Code:    -32001,
			Message: "Error when sending setScene request through Connection",
		}
	}

	log.Printf("INFO: %d. scene set with transition time of: %d", scene, transitionTime)
	if scene == 0 {
		turnedOff.mutex.Lock()
		turnedOff.flag = true
		turnedOff.mutex.Unlock()
	}

	return "ok", nil
}

/*
Sets a Channel's level to a given number

params:

	channel: number of the channel you want to set
	dim: dim percent wise (0-100),
*/
func (h *LampServerHandler) SetChannelLevel(channel int, dim int) (string, error) {
	msg := TransmitMessage{
		JsonRpc: "2.0",
		Id:      uuid.New().String(),
		Method:  "Transmit",
		Params: TransmitParams{
			Id:   1000,
			Data: []int{0, 1, 0, channel, 1, dim, 0},
		},
	}

	if err := websocket.JSON.Send(h.connection, msg); err != nil {
		log.Printf("ERROR: Error when sending SetChannelLevel request %v\n", err)
		return "", &jsonrpc.JSONRPCError{
			Code:    -32002,
			Message: "Error when sending setChannelLevel request",
		}

	} else {
		log.Printf("INFO: Channel %d's level successfully set to: %d\n", channel, dim)

	}
	return "ok", nil
}

/*
Method for manually disconnecting from the EG2s gadget, is used by refreshConnection as well
*/
func (h *LampServerHandler) Disconnect() (string, error) {
	err := wssconnection.Disconnect(h.connection)
	if err != nil {
		return "", &jsonrpc.JSONRPCError{
			Code:    -32003,
			Message: "Error when trying to disconnect from EG2-S",
		}
	}
	done <- true
	log.Println("WARN: WSS Connection Disconnected")
	return "ok", nil
}

/*
Function to manually Connect (through JSON RPC), used by Setup as well
*/
func (h *LampServerHandler) Connect(eg2sHost string, username string, password string) (string, error) {
	var err error = nil
	h.connection, err = wssconnection.Connect(eg2sHost, username, password)
	if err != nil {
		return "", &jsonrpc.JSONRPCError{
			Code:    -32000,
			Message: "Unable to Connect to EG2-S",
		}
	}
	done = make(chan bool)
	brightness = make(chan int)
	go ListenThroughWebSocket(done, brightness, h)
	return "ok", nil
}

/*
Function for connecting after Disconnect, can not be called manually (for security purposes, and validation => only works after Setup)
*/
func (h *LampServerHandler) connect() error {
	var err error = nil
	h.connection, err = wssconnection.Connect(h.eg2sHost, h.username, h.password)
	if err != nil {
		return err
	}
	done = make(chan bool)
	brightness = make(chan int)
	go ListenThroughWebSocket(done, brightness, h)
	return nil
}

/*
With this function you can query the levels of all of your channels. Is used when updating state in HomeAssistant
*/
func (h *LampServerHandler) QueryChannelLevels() ([]int, error) {
	for channel := 1; channel <= h.num_of_channels; channel++ {
		msg := TransmitMessage{
			JsonRpc: "2.0",
			Id:      uuid.New().String(),
			Method:  "Transmit",
			Params: TransmitParams{
				Id:   1000,
				Data: []int{0, 1, 0, channel, 18},
			},
		}
		if err := websocket.JSON.Send(h.connection, msg); err != nil || h.connection.RemoteAddr() == nil {
			log.Printf("ERROR: Error when sending query requests %v\n", err)
			return nil, &jsonrpc.JSONRPCError{
				Code:    -32004,
				Message: "Error when querying channel levels",
			}
		}
		brightnessLevel := <-brightness
		h.channelLevels[channel-1] = brightnessLevel
	}
	if h.num_of_channels == 0 {
		return nil, &jsonrpc.JSONRPCError{
			Code:    -32005,
			Message: "There are no channels to query (num_of_channels is 0)",
		}
	}
	// Turn off Anomaly WORKAROUND
	turnedOff.mutex.Lock()
	if turnedOff.flag {
		turnedOff.flag = false
	}
	turnedOff.mutex.Unlock()

	log.Println("INFO: Channel Levels Successfully Queried")
	return h.channelLevels[:], nil
}

/*
Refreshing WSS connection with the EG2s Gadget
*/
func refreshConnection(rpcServer *LampServerHandler) {
	if _, err := rpcServer.Disconnect(); err != nil {
		return
	}
	if err := rpcServer.connect(); err != nil {
		return
	}
	log.Println("INFO: Connection refreshed")
}

/*
Triggering the webhook of HomeAssistant

param:

	url: Home Assistant's full URL (containing the webhook's path)
*/
func sendRequestToHA(url string) {
	response, err := http.Post(url, "text/plain", nil)
	if err != nil {
		log.Printf("ERROR: An error occured when sending to HA: %v\n", err)

	}
	response.Body.Close()
}

/*
Collects all messages received on websocket and reacts if needed

params:

	done: channel to send done signal to the goroutine
	brightness: channel on which the goroutine sends back the brightness of each channel
	serverHandler: RPC server instance
*/
func ListenThroughWebSocket(done chan bool, brightness chan int, serverHandler *LampServerHandler) {
	wg.Add(1)
	defer wg.Done()
	for {
		select {
		case <-done:
			return
		// Default loop to read all messages from EG2s
		default:
			var reply string
			if err := websocket.Message.Receive(serverHandler.connection, &reply); err != nil {
				log.Printf("ERROR: when reading response from EG2s %v\n", err)
				return
			}
			log.Printf("DEBUG: Sent from EG2s: %s", reply)

			// The method Eg2s uses to notify about change
			if strings.Contains(reply, `"method":"AttributeChangeAlert"`) {
				/*
					I have set a baked in transition time, when changing scenes (as it is right now this can't be overwritten with user input),
					so after 3s delay it reads the final state after setting the scene
				*/
				if strings.Contains(reply, "scene") {
					time.Sleep(3 * time.Second)
					sendRequestToHA(fmt.Sprintf("https://%s/api/webhook/%s", serverHandler.ha_host, serverHandler.webhook_id))
				} else {
					sendRequestToHA(fmt.Sprintf("https://%s/api/webhook/%s", serverHandler.ha_host, serverHandler.webhook_id))
				}

			} else {
				if strings.Contains(reply, `"method":"MessageAlert"`) {
					var queryResponse ChannelQueryResponse
					if err := json.Unmarshal([]byte(reply), &queryResponse); err != nil {
						log.Printf("ERROR: Could not Unmarshal queryResponse: %v\n", err)
						continue
					}

					if queryResponse.Params.Id == 994 {
						turnedOff.mutex.Lock()
						if turnedOff.flag {
							// Turn Off Anomaly WORKAROUND
							brightness <- 0
						} else {
							brightness <- queryResponse.Params.Data[5]
						}
						turnedOff.mutex.Unlock()

					}
					// Turn Off Anomaly WORKAROUND
					if queryResponse.Params.Id == 1001 {
						if queryResponse.Params.Data[3] == 0 {
							defer wg.Done()
							turnedOff.mutex.Lock()
							turnedOff.flag = true
							turnedOff.mutex.Unlock()
						}

					}
				}
			}
		}
	}
}

// The used jsonrpc implementation sometimes doesn't overwrite the Content-Type Header, so WORKAROUND
func setJSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

/*
Start the JSON RPC Server
*/
func Run(waitGroup *sync.WaitGroup) {
	wg = waitGroup
	rpcServer := jsonrpc.NewServer()

	// create a handler instance and register it
	serverHandler := &LampServerHandler{}
	rpcServer.Register("LampServerHandler", serverHandler)

	http.Handle("/", setJSONContentType(rpcServer))
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Could not start server: %v", err)
	} else {
		log.Println("INFO: JSON RPC Server Started")
	}

}
