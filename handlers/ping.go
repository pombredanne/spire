package handlers

import "log"

func HandlePing(deviceName, topic string, payload []byte) error {
	log.Println("It's a ping!")
	return nil
}
