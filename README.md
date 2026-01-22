# iLight-EG2-S-HA-integration
**The project includes a microservice that handles the communication between Home Assistant (within Home Assistant a custom integration) and an iLight EG2-S.**

## Abstract Visualization, Functionality
<img width="701" height="171" alt="EG2-S" src="https://github.com/user-attachments/assets/8cc522da-5c81-4b62-b0d1-a7ee1d828cc4" />

- The microservice gives an API that a HA Custom Integration can use for manipulating the lights' managed by an EG2-S controller
- It builds a WSS connection with the EG2-S Instance which can optionally be set to refresh (disconnect + connect) at a certain time of every day **(be careful with containers that do not share the same time as the host computer)**
- It also gives a JSON-RPC method to query the current state of the lights'
- With webhooks it can be a foundation for a PUSH HA integration (triggers the webhook if the lights change states, it needs a webhook - as far as I know - for forcing out state updates)
- EG2-S operates light channels (not single light sources), the microservice can handle any number of such channels, and you can change the dimming of each one seperately.
- iLight software and so EG2-S offers scenes, which you can also set through the API of the microservice with a transition time of your liking.
- It listens to every request sent by the EG2-S device and filters out ones when the state has actually changed
- Has a workaround regarding the last functionality, as EG2-S can sometimes get into inconsistent states, when a user changes a single channel's dimming and then turns the lights off.
- The always listening makes it capable of detecting state changes that were not made through the microservice itself   

## How to use it
As of right now there is no official Docker Image for the microservice, but the project includes a Dockerfile with which you can build your own (takes about 1-2 minutes). It does not have any data stored persistently so when running the container you do not need to attach volumes. The only thing it needs is a port (originally 8080 but you can forward any port of the host machine you'd like). Unfortunately there is no official Home Assistant Integration either, but it's a work in progress. 

## How to use it on its own (JSON-RPC methods, examples)

### Setup 
```json
{
    "jsonrpc": "2.0",
    "method": "LampServerHandler.Setup",
    "params": [
        "username",
        "password",
        "eg2sHost (ONLY HOST NAME (ip, or DNS))",
        "ha_host, also ONLY host, needs a bit of changing if not HTTPS",
        "webhook_id",
        {{number_of_channels_int}},
        {{reset_time_int}}
    ],
    "id": "uuid"
}
```
This needs to be called first, it allows the microservice to build the WSS connection, and store every required information for its operation later on. {{reset_time_int}} is the time of day when you want the connection to be reset, set to -1 if never.

### SetScene
```json
{
    "jsonrpc": "2.0",
    "method": "LampServerHandler.SetScene",
    "params": [
        {{scene_int}},
        {{transition_time_int}}
    ],
    "id": "uuid"
}
```
{{scene_int}} represents the number identifier of the scene to be set (usually 0 is off ...). {{transition_time_int}} **is directly sent to the EG2-S** which I could not find out how it matches numbers to time periods. **It seems like setting it to 70 makes the transition take 3 seconds**.

### SetChannelLevel
```json
{
    "jsonrpc": "2.0",
    "method": "LampServerHandler.SetChannelLevel",
    "params": [
        {{channel_num_int}},
        {{dim_level_int}}
    ],
    "id": "uuid"
}
```
### QueryChannelLevels
```json
{
    "jsonrpc": "2.0",
    "method": "LampServerHandler.QueryChannelLevels",
    "id": "uuid"
}
```
It returns an Integer array where arr[i] = i-1. channel's dimming level

## Contribution
Feel free to add issues if you notice something not working properly, also feel free to fork the project, if you'd like.

## Disclaimer

**The making of this project was purely for educational reasons.**

