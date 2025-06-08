package wsclient

import (
	"encoding/base64"
	"encoding/json"
	"log"

	"github.com/CP-Payne/comfylite/internal/client/httpclient"
	"github.com/CP-Payne/comfylite/internal/models"
	"github.com/gorilla/websocket"
)

const WebSocketNode = "10"

// TODO: Pass in the client which will send the images with promptID to the webhook
func HandleIncomingMessage(httpClient *httpclient.Client) func(msgType int, msg []byte) {

	currentNode := ""
	outputImages := []string{}
	promptIDTracker := ""

	return func(msgType int, msg []byte) {
		switch msgType {
		case websocket.TextMessage:
			var messageMap map[string]any
			if err := json.Unmarshal(msg, &messageMap); err != nil {
				log.Println("Unmarshal error: ", err)
				return
			}

			messageStatus, _ := messageMap["type"].(string)

			dataMap, _ := messageMap["data"].(map[string]any)
			if dataMap == nil {
				log.Println("Invalid 'data' field in message")
				return
			}

			promptID, _ := dataMap["prompt_id"].(string)

			node, _ := dataMap["node"].(string)

			if messageStatus == "execution_start" {
				promptIDTracker = promptID
				currentNode = ""
				outputImages = []string{}
			} else if messageStatus == "executing" {
				if promptID == promptIDTracker && promptID != "" && node != "" {
					currentNode = node
				}
			} else if messageStatus == "execution_success" {
				log.Println("Execution complete, ready to send images")
				httpClient.NotifyWebhook(models.WebhookRequestData{
					PromptID: promptID,
					Images:   outputImages,
				})
				// TODO: Call client here which will execute the webhook
			} else {
				log.Println("Unhandled message status: ", messageStatus)
			} // add error check here

		case websocket.BinaryMessage:
			log.Println("FOUND BINARY===================")
			if currentNode == WebSocketNode {
				outputImages = append(outputImages, base64.StdEncoding.EncodeToString(msg))
			}
		default:
			log.Println("Unknown message type: ", msgType)

		}
	}

}
