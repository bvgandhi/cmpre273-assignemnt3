package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//
type postUberResponse struct {
	Driver          interface{} `json:"driver"`
	Eta             int         `json:"eta"`
	Location        interface{} `json:"location"`
	RequestID       string      `json:"request_id"`
	Status          string      `json:"status"`
	SurgeMultiplier int         `json:"surge_multiplier"`
	Vehicle         interface{} `json:"vehicle"`
}

//trip sequence struct
type seqstruct struct {
	ID  string `json:"_id"`
	Seq int    `json:"seq"`
}

//Post Reqt Struct
type tripPostReqtStruct struct {
	LocationIds            []string `json:"location_ids"`
	StartingFromLocationID string   `json:"starting_from_location_id"`
}

type tripPostStruct struct {
	//id of the trip, static counter increasing per trip palnned by user..1 per a to bcd
	Tripid                 int `json:"id"`
	Status                 string
	StartingFromLocationID string
	BestRouteLocationIds   []string
	TotalUberCosts         int
	TotalUberDuration      int
	TotalDistance          float64
	LocationPtr            int `json:"-"`
}

// for getting from mongo the lat n long for a locationid previulsy stored
type locationstruct struct {
	MyID       int    `json:"id"`
	Name       string `json:"name"`
	Address    string `json:"address"`
	City       string `json:"city"`
	State      string `json:"state"`
	Zip        int    `json:"zip"`
	Coordinate struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"coordinate"`
}

type uberResponse struct {
	Prices []struct {
		CurrencyCode         string  `json:"currency_code"`
		DisplayName          string  `json:"display_name"`
		Distance             float64 `json:"distance"`
		Duration             int     `json:"duration"`
		Estimate             string  `json:"estimate"`
		HighEstimate         int     `json:"high_estimate"`
		LocalizedDisplayName string  `json:"localized_display_name"`
		LowEstimate          int     `json:"low_estimate"`
		Minimum              int     `json:"minimum"`
		ProductID            string  `json:"product_id"`
		SurgeMultiplier      int     `json:"surge_multiplier"`
	} `json:"prices"`
}
type unitTripInfo struct {
	uberCosts    int
	uberDuration int
	distance     float64
	productID    string
}

type tripPutRespStrc struct {
	//id of the trip, which
	Tripid                 string `json:"id"`
	Status                 string `json:"status"`
	StartingFromLocationID string `json:"starting_from_location_id"`
	NextDestnLocnID        string `json:"next_destination_location_id"`

	BestRouteLocationIds []string `json:"best_route_location_ids"`
	TotalUberCosts       int      `json:"total_uber_costs"`
	TotalUberDuration    int      `json:"total_uber_duration"`
	TotalDistance        float64  `json:"total_distance"`
	UberWaitTimeEta      int      `json:"uberWaitTimeEta"`
}

//var values = [][]string{}
var counter int

//var optimalRoute = []string{}

const finishConst string = "finished"
const plngConst string = "planning"
const reqtConst string = "requesting"
const accessToken string = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzY29wZXMiOlsicmVxdWVzdCJdLCJzdWIiOiI2M2M0NTc3Zi05ZDk4LTQ4NTUtODg2MC03YTIwMmFlMThmNTkiLCJpc3MiOiJ1YmVyLXVzMSIsImp0aSI6IjJlOGYzNjA2LTdkMWItNGUxNy05ZTg1LTg4ZDZkZWQ0ODNkMSIsImV4cCI6MTQ1MDY5NTAwMywiaWF0IjoxNDQ4MTAzMDAyLCJ1YWN0IjoiOXdwNGFscEF4cnFzN3VkMlFIU1hFN3N0RDZEZzN5IiwibmJmIjoxNDQ4MTAyOTEyLCJhdWQiOiJndjZla1hTNW9Ebm1SRGM0SGVMSndOQllVcERpX2p1WiJ9.YbYZgEP5Fj4Nqd8_fR2Q1ZfJEve8rBfKII4BDZJvg4eAXo1HUs_80bkwkW-FFTQr6qdQkY_djWGcIGBxal3M5HoW9-AV9r-kX4JZnRZrqimfVzf7RTLh5ve4tMFmiHTGB_N5oqilKtCnwW1JSj3N0GdkwKelysytPh8Dn_YqgLQmC5mpZk0L08x0NIjm1TZaaEZKxYF2KVNTHcbpbM0VmntLN61cjXKLgnUUYlXe20DMJMvaYIRYLrMaN940OuKZZKU_xlBMTZUA_7NE2s6JEqyn_AASoDjvkdDlKsDMPgI-Ea88h84YwgRqXpwU4AINeS7i_Dx1W5M6rGIcxWO9SA"

func permute(arrfunc []string, k int, valuesptr *[][]string) [][]string {

	for i := k; i < len(arrfunc); i++ {

		temp := arrfunc[i]
		arrfunc[i] = arrfunc[k]
		arrfunc[k] = temp
		permute(arrfunc, k+1, valuesptr)
		arrfunc[k] = arrfunc[i]
		arrfunc[i] = temp

		if k == (len(arrfunc) - 1) {

			counter = counter + 1
			cobn := make([]string, len(arrfunc))
			fmt.Println(counter)
			for x := 0; x < len(arrfunc); x++ {
				cobn[x] = arrfunc[x]
			}
			// Append each row to the two-dimensional slice.
			*valuesptr = append(*valuesptr, cobn)
		}
	}
	return *valuesptr
}

func postOptimalRoute(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {

	tripReqt := tripPostReqtStruct{}
	var unitTrip unitTripInfo
	var optimalRoute = []string{}
	values := [][]string{}
	totalTripInfo := unitTripInfo{}
	//fmt.Println(json.NewDecoder(req.Body))
	err := json.NewDecoder(req.Body).Decode(&tripReqt)

	permute(tripReqt.LocationIds, 0, &values)
	if len(values) == 0 {
		panic(errors.New("Length of Trip Array Cannot be 0"))
	}

	totalTripInfo = unitTripInfo{9999999, 9999999, 0, ""}

	optimalRoute = values[0]
	var j int
	for _, v := range values {
		//fmt.Printf("Array Row %d = %v\n", i, v)
		currtotalTripInfo := unitTripInfo{}
		prevLocation := tripReqt.StartingFromLocationID
		for j = 0; j < len(v); j++ {

			unitTrip = getCostFor2Location(prevLocation, v[j])
			prevLocation = v[j]
			//fmt.Println(unitTrip)
			currtotalTripInfo.uberCosts += unitTrip.uberCosts
			currtotalTripInfo.uberDuration += unitTrip.uberDuration
			currtotalTripInfo.distance += unitTrip.distance
			//fmt.Println(totalTripInfo)
		}
		getCostFor2Location(v[j-1], tripReqt.StartingFromLocationID)
		currtotalTripInfo.uberCosts += unitTrip.uberCosts
		currtotalTripInfo.uberDuration += unitTrip.uberDuration
		currtotalTripInfo.distance += unitTrip.distance
		// now we ahve calc currtotalTripInfo we will compare it with prev optimal soln
		// if its less then we will chnage optimal route and totalTripInfo struct
		if currtotalTripInfo.uberCosts < totalTripInfo.uberCosts {
			totalTripInfo = currtotalTripInfo
			//v is is 1 combn of routes
			optimalRoute = v
		}
	}

	uri := "mongodb://bhumikgandhi05:b05051988@ds041404.mongolab.com:41404/bg273"

	sess, err := mgo.Dial(uri)
	if err != nil {
		fmt.Printf("Can't connect to mongo, go error %v\n", err)
		os.Exit(1)
	}
	defer sess.Close()

	sess.SetSafe(&mgo.Safe{})

	//increment  the sequence in db, create the structure to store db document/row
	seqstructins := seqstruct{}

	change := mgo.Change{
		Update:    bson.M{"$inc": bson.M{"seq": 1}},
		ReturnNew: true,
	}
	collection1 := sess.DB("bg273").C("tripsequence")
	_, err1 := collection1.Find(bson.M{"_id": "tripid"}).Apply(change, &seqstructins)
	if err1 != nil {
		fmt.Println("got an error finding a doc")
		os.Exit(1)
	}

	resp := tripPostStruct{}

	resp.Tripid = seqstructins.Seq
	resp.Status = plngConst
	resp.StartingFromLocationID = tripReqt.StartingFromLocationID
	resp.BestRouteLocationIds = optimalRoute
	resp.TotalUberCosts = totalTripInfo.uberCosts
	resp.TotalUberDuration = totalTripInfo.uberDuration
	resp.TotalDistance = totalTripInfo.distance
	//to keep track of location where the planner will go in put
	resp.LocationPtr = -1
	//	resp.ID = bson.NewObjectId()
	collection := sess.DB("bg273").C("trip")
	err = collection.Insert(resp)
	if err != nil {
		fmt.Printf("Can't insert document: %v\n", err)
		os.Exit(1)
	}

	repjson, err := json.Marshal(resp)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	rw.WriteHeader(201)
	fmt.Fprintf(rw, "%s", repjson)

}

func getCostFor2Location(x string, y string) unitTripInfo {

	intx, err := strconv.Atoi(x)
	if err != nil {
		fmt.Printf("Incorrect input location array %v\n", err)
		os.Exit(1)
	}

	inty, err := strconv.Atoi(y)
	if err != nil {
		fmt.Printf("Incorrect input location array %v\n", err)
		os.Exit(1)
	}

	var unittrip unitTripInfo
	uri := "mongodb://bhumikgandhi05:b05051988@ds041404.mongolab.com:41404/bg273"

	sess, err := mgo.Dial(uri)
	if err != nil {
		fmt.Printf("Can't connect to mongo, go error %v\n", err)
		os.Exit(1)
	}
	defer sess.Close()
	sess.SetSafe(&mgo.Safe{})
	collection := sess.DB("bg273").C("resp")

	responsex := locationstruct{}
	err1 := collection.Find(bson.M{"myid": intx}).One(&responsex)
	if err1 != nil {
		panic(err1)
	}

	responsey := locationstruct{}
	err1 = collection.Find(bson.M{"myid": inty}).One(&responsey)
	if err1 != nil {
		panic(err1)
	}
	var suberResponse uberResponse
	var buffer bytes.Buffer

	buffer.WriteString("https://api.uber.com/v1/estimates/price?start_latitude=")
	buffer.WriteString(strconv.FormatFloat(responsex.Coordinate.Lat, 'f', -1, 64))
	buffer.WriteString("&start_longitude=")
	buffer.WriteString(strconv.FormatFloat(responsex.Coordinate.Lng, 'f', -1, 64))
	buffer.WriteString("&end_latitude=")
	buffer.WriteString(strconv.FormatFloat(responsey.Coordinate.Lat, 'f', -1, 64))
	buffer.WriteString("&end_longitude=")
	buffer.WriteString(strconv.FormatFloat(responsey.Coordinate.Lng, 'f', -1, 64))
	buffer.WriteString("&server_token=caCDC7m1RX9C1pjG3m73g7Ezvja2E0NUsuSBd4Rl")

	response, err := http.Get(buffer.String())
	if err != nil {
		fmt.Printf("error occured")
		fmt.Printf("%s", err)
		os.Exit(1)
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("%s", err)
			os.Exit(1)
		}
		json.Unmarshal([]byte(contents), &suberResponse)

		//	log.Printf("#########x to y " + x + "y: " + y)

		unittrip.uberCosts = suberResponse.Prices[0].LowEstimate
		unittrip.uberDuration = suberResponse.Prices[0].Duration
		unittrip.distance = suberResponse.Prices[0].Distance
		unittrip.productID = suberResponse.Prices[0].ProductID

	}
	return unittrip
}

//get for a trip
func getTrip(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
	//fmt.Fprintf(rw, "Hello, %s!\n", p.ByName("locationid"))

	// connect to mongo
	uri := "mongodb://bhumikgandhi05:b05051988@ds041404.mongolab.com:41404/bg273"

	sess, err := mgo.Dial(uri)
	if err != nil {
		fmt.Printf("Can't connect to mongo, go error %v\n", err)
		os.Exit(1)
	}
	defer sess.Close()

	sess.SetSafe(&mgo.Safe{})
	collection := sess.DB("bg273").C("trip")
	response := tripPostStruct{}
	intid, _ := strconv.Atoi(p.ByName("tripid"))
	err1 := collection.Find(bson.M{"tripid": intid}).One(&response)
	if err1 != nil {
		panic(err1)
	}

	//marshal struct to json
	repjson, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	rw.WriteHeader(200)
	fmt.Fprintf(rw, "%s", repjson)
}

func putCommenceTrip(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {

	//left part of url
	intid, err := strconv.Atoi(p.ByName("tripid"))

	// connect to mongo
	uri := "mongodb://bhumikgandhi05:b05051988@ds041404.mongolab.com:41404/bg273"

	sess, err := mgo.Dial(uri)
	if err != nil {
		fmt.Printf("Can't connect to mongo, go error %v\n", err)
		os.Exit(1)
	}
	defer sess.Close()
	response := tripPostStruct{}
	sess.SetSafe(&mgo.Safe{})
	collection := sess.DB("bg273").C("trip")
	err1 := collection.Find(bson.M{"tripid": intid}).One(&response)
	if err1 != nil {
		panic(err1)
	}
	var putResp tripPutRespStrc
	if response.Status == finishConst {
		putResp = createFinishedTripStruct(response)
	} else {
		var startLocnID, destLocnID int
		if response.LocationPtr == -1 {
			startLocnID, err = strconv.Atoi(response.StartingFromLocationID)
		} else {
			startLocnID, err = strconv.Atoi(response.BestRouteLocationIds[response.LocationPtr])
		}
		response.LocationPtr++
		putResp = tripPutRespStrc{}
		/*fmt.Println("============")
		fmt.Println((len(response.BestRouteLocationIds)))
		fmt.Println(response.LocationPtr)*/

		if response.LocationPtr == (len(response.BestRouteLocationIds)) {
			putResp.Status = finishConst
			destLocnID, err = strconv.Atoi(response.StartingFromLocationID)
		} else {
			putResp.Status = reqtConst
			destLocnID, err = strconv.Atoi(response.BestRouteLocationIds[response.LocationPtr])
		}

		colQuerier := bson.M{"tripid": intid}
		change := bson.M{"$set": bson.M{"status": putResp.Status, "locationptr": response.LocationPtr}}
		err = collection.Update(colQuerier, change)
		if err != nil {
			panic(err)
		}

		collection = sess.DB("bg273").C("resp")

		responsex := locationstruct{}

		err1 = collection.Find(bson.M{"myid": startLocnID}).One(&responsex)
		if err1 != nil {
			panic(err1)
		}

		responsey := locationstruct{}
		//intid, _ := strconv.Atoi(p.ByName("locationid"))
		err1 = collection.Find(bson.M{"myid": destLocnID}).One(&responsey)
		if err1 != nil {
			panic(err1)
		}

		// generate request the uber api	postUberReply
		client := &http.Client{}

		startlatitude := strconv.FormatFloat(responsex.Coordinate.Lat, 'f', -1, 64)
		startlongitude := strconv.FormatFloat(responsex.Coordinate.Lng, 'f', -1, 64)
		endlatitude := strconv.FormatFloat(responsey.Coordinate.Lat, 'f', -1, 64)
		endlongitude := strconv.FormatFloat(responsey.Coordinate.Lng, 'f', -1, 64)

		unitTrip := getCostFor2Location(strconv.Itoa(startLocnID), strconv.Itoa(destLocnID))
		//unitTripInfo
		jsonprep := `{"start_latitude":"` + startlatitude + `","start_longitude":"` + startlongitude +
			`","end_latitude":"` + endlatitude + `","end_longitude":"` + endlongitude + `","product_id":"` + unitTrip.productID + `"}`
		fmt.Println(jsonprep)

		var jsonStr = []byte(jsonprep)
		r, _ := http.NewRequest("POST", "https://sandbox-api.uber.com/v1/requests", bytes.NewBuffer(jsonStr))
		//	r, _ := http.NewRequest("POST", posturl, nil)
		r.Header.Set("Accept", "application/json")
		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
		r.Header.Set("Content-Type", "application/json")
		//Call to the uber POST API which returns reqtid and ETA
		resp, _ := client.Do(r)
		fmt.Println(resp.Status)

		uberReqtresponse := postUberResponse{}
		if err != nil {
			fmt.Printf("error occured")
			fmt.Printf("%s", err)
			os.Exit(1)
		} else {
			defer resp.Body.Close()
			contents, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("%s", err)
				os.Exit(1)
			}

			json.Unmarshal([]byte(contents), &uberReqtresponse)

			putResp.UberWaitTimeEta = uberReqtresponse.Eta
			if putResp.Status == reqtConst {
				putResp.NextDestnLocnID = response.BestRouteLocationIds[response.LocationPtr]
			} else {
				putResp.NextDestnLocnID = response.StartingFromLocationID
			}

			putResp.BestRouteLocationIds = response.BestRouteLocationIds
			putResp.StartingFromLocationID = response.StartingFromLocationID
			putResp.Tripid = strconv.Itoa(response.Tripid)

			putResp.TotalUberCosts = response.TotalUberCosts
			putResp.TotalUberDuration = response.TotalUberDuration
			putResp.TotalDistance = response.TotalDistance

		}
	}
	repjson, err := json.Marshal(putResp)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	rw.WriteHeader(201)
	fmt.Fprintf(rw, "%s", repjson)

}

func createFinishedTripStruct(response tripPostStruct) tripPutRespStrc {
	putResp := tripPutRespStrc{}
	putResp.UberWaitTimeEta = 0
	putResp.Status = finishConst
	putResp.NextDestnLocnID = response.StartingFromLocationID

	putResp.BestRouteLocationIds = response.BestRouteLocationIds
	putResp.StartingFromLocationID = response.StartingFromLocationID
	putResp.Tripid = strconv.Itoa(response.Tripid)

	putResp.TotalUberCosts = response.TotalUberCosts
	putResp.TotalUberDuration = response.TotalUberDuration
	putResp.TotalDistance = response.TotalDistance
	return putResp
}

func getUberPostHeader(accessToken string) http.Header {
	header := make(http.Header)
	header.Set("Accept", "application/json")
	header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	return header
}

func main() {
	mux := httprouter.New()
	mux.GET("/trips/:tripid", getTrip)
	mux.PUT("/trips/:tripid", putCommenceTrip)
	/*	mux.DELETE("/location/:locationid", deleteLocation)*/
	mux.POST("/trips", postOptimalRoute)
	server := http.Server{
		Addr:    "localhost:8080",
		Handler: mux,
	}
	server.ListenAndServe()
}
