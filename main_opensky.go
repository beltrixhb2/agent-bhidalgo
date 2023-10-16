
package main

import (
	"fmt"
	loggly "github.com/jamespearly/loggly"
	"net/http"
	"bufio"
	"os"
	"encoding/json"
	"strconv"
	"time"
	"net"
)


type ResponseData struct {
	Time int `json:"time"`
	AircraftList [][]interface{} `json:"states"`
}

func fetchData(client *loggly.ClientType) {

	apiURL := "https://opensky-network.org/api/states/all?lamax=44&lomin=-80&lomax=-75&lamin=43"

	//Create API request
	request, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		client.EchoSend("error","Error creating request")
		return
	}

	// Read API credentials from environment variables
	apiUsername := os.Getenv("API_USERNAME")
	apiPassword := os.Getenv("API_PASSWORD")

	// Check if credentials are available
	if apiUsername == "" || apiPassword == "" {
		client.EchoSend("error","API credentials not set. Please set API_USERNAME and API_PASSWORD environment variables.")
		return
	}

	// Set the Authorization header for basic authentication
	request.SetBasicAuth(apiUsername, apiPassword)

	//Set the time of the timeout
	timeout := 5 * time.Second

	// Make a GET request to the API
	http_client := http.Client{
		Timeout: timeout,
	}
	response, err := http_client.Do(request)

	//Check request errors
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			client.EchoSend("error","HTTP: request timed out")
		} else {
			client.EchoSend("error","Error in the API request")
		}
		return
	}
	// Check the HTTP status code
        if response.StatusCode != http.StatusOK {
                if response.StatusCode == http.StatusBadGateway {
                        // Handle 502 Bad Gateway error
                        client.EchoSend("error","API returned a 502 Bad Gateway error.")
                        return
                } else {
                        // Handle other HTTP status codes
                        client.EchoSend("error","API returned an unexpected status code:")
                        return
                }
        }

	// Decode the JSON response into a struct
	var responseData ResponseData
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&responseData)
	if err != nil {
		fmt.Println(err)
		client.EchoSend("error", "Error decoding JSON")
		return
	}

	// Ask 4 more times for the states of the aircrafts if the states were empty
	// Error in the API sent sometimes empty JSON as if there were no flying aircrafts
	if responseData.AircraftList== nil{
		client.Send("warning","API error, no aircraft states received: attempt " + strconv.Itoa(1))
		for attempt := 1; attempt<5; attempt++{
			response, err := http_client.Do(request)
			// Check the HTTP status code
			if response.StatusCode != http.StatusOK {
			        if response.StatusCode == http.StatusBadGateway {
			                // Handle 502 Bad Gateway error
	        		        client.EchoSend("error","API returned a 502 Bad Gateway error.")
					return
	      			} else {
	      			        // Handle other HTTP status codes
	      			        client.EchoSend("error","API returned an unexpected status code:")
					return
	     			}
			}
			// Check request errors
		        if err != nil {
               			 if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
                       			 client.EchoSend("error","HTTP: request timed out")
               			 } else {
                       			 client.EchoSend("error","Error in the API request")
               			 }
               			 return
       			 }
		        defer response.Body.Close()

			// Decode JSON
			decoder := json.NewDecoder(response.Body)
		        err = decoder.Decode(&responseData)
		        if err != nil {
		                fmt.Println(err)
		                client.EchoSend("error", "Error decoding JSON")
		                return
		        }

			// Check again if the list is empty
			if responseData.AircraftList== nil{
	                client.Send("warning","API error, no aircraft states received: attempt " + strconv.Itoa(attempt+1))
			}
			if attempt == 4 {
				client.EchoSend("error","5 failed attempts to fetch aircraft data, maybe there are not aircrafts at this time")
				return
			}
	        }
	}
	response_size := float64(response.ContentLength)/1024.0
	client.Send("info","Succesfull API request. Response size="+strconv.FormatFloat(response_size, 'f', 5, 64)+"KB")
	fmt.Println("There are ",len(responseData.AircraftList)," aircrafts flying over Lake Ontario and surroundings")
}

func main() {

	var tag string
	tag = "My-Go-Demo"

	// Instantiate the client
	client := loggly.New(tag)
	client.Send("info","Execution started")

	fmt.Println("Press Enter to fetch the number of aircrafts flying over Lake Ontario and surroundings.\n Type 'exit' to exit.")

	// Create a scanner to read user input
	scanner := bufio.NewScanner(os.Stdin)

	// Infinite loop
	for {
		// Wait for user input
		fmt.Print(">")
		scanner.Scan()
		input := scanner.Text()

		// Check if the user wants to exit
		if input == "exit" {
			fmt.Println("Exiting...")
			break
		}

		// Fetch data when the user presses Enter
		if input == "" {
			fetchData(client)
		}
	}

}
