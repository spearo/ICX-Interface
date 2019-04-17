package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var INDICATOR_IDLE = byte(0xF1)
var INDICATOR_DATA = byte(0xF2)
var RESERVE_BYTE = byte(0x00)
var END_BYTE = byte(0xFF)

var IDLE_MESSAGE = []byte{INDICATOR_IDLE, RESERVE_BYTE, END_BYTE}

var intercomDirectory = map[string]string{
	"F150": "PHONENUMBER",
	"F101": "PHONENUMBER",
}

type Task struct {
	SystemReceiver string
	SystemSender   string
	TaskCode       string
	TypeCode       string
	Params         []string
	Data           string
}

func main() {
	// Just a message for debugging
	log("Connecting...")

	// Make the connection to the server
	conn, err := net.Dial("tcp", "192.168.1.200:18000")

	// Print any errors if available
	if err != nil {
		panic(err)
	}

	// Just a message for debugging
	log("Connected.")

	// start go routine in another thread
	go waitForMessage(conn)

	for {
		// read in input from command line
		reader := bufio.NewReader(os.Stdin)
		// Read everything until you press enter
		text, _ := reader.ReadString('\n')
		sendMessage(text, conn)
	}

	//close connection (but we won't ever get here)
	conn.Close()
}

func sendMessage(text string, conn net.Conn) {
	//Cut off any newline character
	text = strings.Trim(text, "\r\n")

	//Turn string into bytes
	textBytes := []byte(text)

	//Start the array with indicator and reserve
	byteArray := []byte{INDICATOR_DATA, RESERVE_BYTE}

	//Then add the actual data
	byteArray = append(byteArray, textBytes...)

	//Then add the end byte
	byteArray = append(byteArray, END_BYTE)

	//Just a message for debugging
	log(fmt.Sprintf("About to send: % x", byteArray))

	//send bytes to server
	conn.Write(byteArray)
}

func waitForMessage(conn net.Conn) {
	message, _ := bufio.NewReader(conn).ReadBytes(END_BYTE)

	//var stringArray []string

	if bytes.Equal(message, IDLE_MESSAGE) {
		log("Received from server: IDLE")
		conn.Write(IDLE_MESSAGE)
		log("Sent to server: IDLE")

	} else {
		task := getTaskFromMessage(message)
		log(fmt.Sprintf("Task: %v", task))

		if task.TaskCode == "5B" && task.TypeCode == "30" {
			recipient := intercomDirectory[task.Params[0]]
			message := fmt.Sprintf("Just receivd Input %v on Intercom %v", task.Params[1], task.Params[0])
			sendSMS(recipient, message)
		}

	}
	waitForMessage(conn)
}

func getTaskFromMessage(message []byte) Task {
	taskCode, _ := strconv.Atoi(getStringFromRange(message, 4, 5))
	var task Task

	if taskCode < 60 {
		// Short Task
		task = Task{
			SystemReceiver: getStringFromRange(message, 2, 3),
			SystemSender:   getStringFromRange(message, 6, 7),
			TaskCode:       getStringFromRange(message, 4, 5),
			TypeCode:       getStringFromRange(message, 16, 17),
			Params:         []string{getStringFromRange(message, 8, 11), getStringFromRange(message, 12, 15)},
		}
	} else if taskCode < 80 {
		// Long Task
		task = Task{
			SystemReceiver: getStringFromRange(message, 2, 3),
			TaskCode:       getStringFromRange(message, 4, 5),
			SystemSender:   getStringFromRange(message, 6, 7),
			TypeCode:       getStringFromRange(message, 8, 9),
			Params:         []string{getStringFromRange(message, 10, 13)},
			Data:           getStringFromRange(message, 14, len(message)),
		}
	}

	return task
}

func getStringFromRange(message []byte, start int, end int) string {
	var returnString string
	for i := start; i <= end; i++ {
		returnString = returnString + string(message[i])
	}
	return returnString
}

func sendSMS(recipient string, message string) {
	accountSid := "TWILIOACCOUNTSID"
	authToken := "TWILIOACCOUNTUTHTOKEN"
	urlStr := "https://api.twilio.com/2010-04-01/Accounts/" + accountSid + "/Messages.json"

	v := url.Values{
		"To":   {recipient},
		"From": {"TWILIOFROMNUMBER"},
		"Body": {message},
	}

	rb := *strings.NewReader(v.Encode())
	client := &http.Client{}
	req, _ := http.NewRequest("POST", urlStr, &rb)
	req.SetBasicAuth(accountSid, authToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	//   // Make request
	_, _ = client.Do(req)
}

func log(msg string) {
	fmt.Printf("[%s] %v\n", time.Now().Format("02/01/2006 15:04:05.000"), msg)
}
