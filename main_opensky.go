package main

import (
	"github.com/aws/aws-sdk-go/aws"
   	"github.com/aws/aws-sdk-go/aws/session"
   	"github.com/aws/aws-sdk-go/service/dynamodb"
   	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"fmt"
	loggly "github.com/jamespearly/loggly"
	"net/http"
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

type AircraftState struct {
	Icao24        string  `json:"icao24"`
	Callsign      string  `json:"callsign"`
	OriginCountry string  `json:"origin_country"`
	Longitude     float64 `json:"longitude"`
	Latitude      float64 `json:"latitude"`
	BaroAltitude  float64 `json:"baro_altitude"`
	OnGround      bool    `json:"on_ground"`
	Velocity      float64 `json:"velocity"`
	TrueTrack     float64 `json:"true_track"`
	VerticalRate  float64 `json:"vertical_rate"`
	GeoAltitude   float64 `json:"geo_altitude"`
}


type StoreType struct{
    AircraftList string `json:"aircraft_state_list"`
	Time int
}


func convertAircraftList(original ResponseData) StoreType {
	var result StoreType
	result.Time = original.Time
	var list []AircraftState

	for _, state := range original.AircraftList {
		if len(state) >= 17 {
			icao24, _ := state[0].(string)
			callsign, _ := state[1].(string)
			originCountry, _ := state[2].(string)
			longitude, _ := state[5].(float64)
			latitude, _ := state[6].(float64)
			baroAltitude, _ := state[7].(float64)
			onGround, _ := state[8].(bool)
			velocity, _ := state[9].(float64)
			trueTrack, _ := state[10].(float64)
			verticalRate, _ := state[11].(float64)
			geoAltitude, _ := state[13].(float64)

			list = append(list, AircraftState{
				Icao24:        icao24,
				Callsign:      callsign,
				OriginCountry: originCountry,
				Longitude:     longitude,
				Latitude:      latitude,
				BaroAltitude:  baroAltitude,
				OnGround:      onGround,
				Velocity:      velocity,
				TrueTrack:     trueTrack,
				VerticalRate:  verticalRate,
				GeoAltitude:   geoAltitude,
			})
		}
	}
	jsonData, _ := json.Marshal(list)
	result.AircraftList = string(jsonData)
	return result
}


func fetchData(client *loggly.ClientType) StoreType {

	apiURL := "https://opensky-network.org/api/states/all?lamax=44&lomin=-80&lomax=-75&lamin=43"

	//Create API request
	request, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		client.EchoSend("error","Error creating request")
		return StoreType{}
	}

	// Read API credentials from environment variables
	apiUsername := os.Getenv("API_USERNAME")
	apiPassword := os.Getenv("API_PASSWORD")

	// Check if credentials are available
	if apiUsername == "" || apiPassword == "" {
		client.EchoSend("error","API credentials not set. Please set API_USERNAME and API_PASSWORD environment variables.")
		return StoreType{}
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
		return StoreType{}
	}
	// Check the HTTP status code
        if response!=nil && response.StatusCode != http.StatusOK {
                if response.StatusCode == http.StatusBadGateway {
                        // Handle 502 Bad Gateway error
                        client.EchoSend("error","API returned a 502 Bad Gateway error.")
                        return StoreType{}
                } else {
                        // Handle other HTTP status codes
                        client.EchoSend("error","API returned an unexpected status code:")
                        return StoreType{}
                }
        }

	// Decode the JSON response into a struct
	var responseData ResponseData
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&responseData)
	if err != nil {
		fmt.Println(err)
		client.EchoSend("error", "Error decoding JSON")
		return StoreType{}
	}

	// Ask 4 more times for the states of the aircrafts if the states were empty
	// Error in the API sent sometimes empty JSON as if there were no flying aircrafts
	if responseData.AircraftList== nil{
		client.EchoSend("warning","API error, no aircraft states received")
		for attempt := 1; attempt<5; attempt++{
			response, err := http_client.Do(request)
			// Check the HTTP status code
			if response!=nil && response.StatusCode != http.StatusOK {
			        if response.StatusCode == http.StatusBadGateway {
			                // Handle 502 Bad Gateway error
	        		        client.EchoSend("error","API returned a 502 Bad Gateway error.")
					return StoreType{}
	      			} else {
	      			        // Handle other HTTP status codes
	      			        client.EchoSend("error","API returned an unexpected status code:")
					return StoreType{}
	     			}
			}
			// Check request errors
		        if err != nil {
               			 if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
                       			 client.EchoSend("error","HTTP: request timed out")
               			 } else {
                       			 client.EchoSend("error","Error in the API request")
               			 }
               			 return StoreType{}
       			 }
			// Decode JSON
			decoder := json.NewDecoder(response.Body)
		        err = decoder.Decode(&responseData)
		        if err != nil {
		                fmt.Println(err)
		                client.EchoSend("error", "Error decoding JSON")
		                return StoreType{}
		        }
			// Check again if the list is empty
			if responseData.AircraftList== nil{
	                	client.EchoSend("warning","API error, no aircraft states received: attempt")
				if attempt == 4 {
					client.EchoSend("error","5 failed attempts to fetch aircraft data, maybe there are not aircrafts at this time")
					return StoreType{}
				}
			}
	        }
	}
	response_size := float64(response.ContentLength)/1024.0
	client.EchoSend("info","Succesfull API request. Response size="+strconv.FormatFloat(response_size, 'f', 5, 64)+"KB")
	fmt.Println("There are ",len(responseData.AircraftList)," aircrafts flying over Lake Ontario and surroundings")
	return convertAircraftList(responseData)
}

func store_in_dynamo(client *loggly.ClientType, data StoreType, svc *dynamodb.DynamoDB){
	if data.Time!=0{
                        tableName := "bhidalgo_Aircraft_States"

                        av, err := dynamodbattribute.MarshalMap(data)
                        if err != nil {
                                client.EchoSend("error","Got error marshalling item")
                                return
                        }

                        input := &dynamodb.PutItemInput{
                                Item:      av,
                                TableName: aws.String(tableName),
                        }

                        _, err = svc.PutItem(input)
                        if err != nil {
                                client.EchoSend("error","Error putting item in the table")
				fmt.Printf("%+v\n", data)
                                fmt.Println(err)
                                return
                        }
                }
}

func main() {

	var tag string
	tag = "My-Go-Demo"

	// Instantiate the client
	client := loggly.New(tag)
	client.Send("info","Execution started")

	sess := session.Must(session.NewSessionWithOptions(session.Options{
   		 SharedConfigState: session.SharedConfigEnable,
	}))
	svc := dynamodb.New(sess)
	client.Send("info","Conected to AWS")

	data := fetchData(client)
	if data.Time!=0{
            store_in_dynamo(client, data, svc)
        }

	ticker := time.NewTicker(2 * time.Minute)

	// Run a goroutine to execute your function on each tick
	go func() {
		for {
			select {
			case <-ticker.C:
				data := fetchData(client)
				if data.Time!=0{
			        	store_in_dynamo(client, data, svc)
			        }
			}
		}
	}()
    
    select{}
}
