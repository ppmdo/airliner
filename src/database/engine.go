package database

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"testing"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/joho/godotenv"
)

const Bucket = "airliner"
const testBucket = Bucket + "TEST"

const org = "iot"

type AirlineOffer struct {
	Url           string
	FromAirport   string
	ToAirport     string
	DepartureDate time.Time
	ReturnDate    time.Time
	Price         float64
	CreatedOn     time.Time
}

type DBClient = influxdb2.Client

var mockData = []AirlineOffer{
	{
		Url:           "www.example.com",
		FromAirport:   "MUC",
		ToAirport:     "LIS",
		DepartureDate: time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC),
		ReturnDate:    time.Date(2023, 4, 1, 0, 0, 0, 0, time.UTC),
		Price:         242.66,
		CreatedOn:     time.Now().Round(1 * time.Second).UTC(),
	},
}

func Test_write_event_with_line_protocol(t *testing.T) {
	tests := []struct {
		name  string
		f     func(influxdb2.Client, []AirlineOffer)
		datas []AirlineOffer
	}{
		{
			name: "Write new record with line protocol",
			// Your data Points
			datas: mockData,
			f: func(c influxdb2.Client, datas []AirlineOffer) {
				// Send all the data to the DB
				for _, data := range datas {
					write_event_with_line_protocol(c, data, testBucket)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := init_testDB(t)

			// call function to test
			tt.f(client, tt.datas)
			// test can be flicky if the query is done before that data is ready in the database
			time.Sleep(time.Millisecond * 1000)

			// Option one: QueryTableResult
			results := read_events_as_query_table_result(client, testBucket)
			// convert results to array to compare with data
			resultsArr := []AirlineOffer{}
			for _, v := range results {
				resultsArr = append(resultsArr, v)
			}

			if eq := reflect.DeepEqual(resultsArr, tt.datas); !eq {
				t.Errorf("want %v, got %v", tt.datas, resultsArr)
			}

			// Option two: query raw data
			// TODO add validation
			read_events_as_raw_string(client, testBucket)

			client.Close()
		})
	}
}

func Test_write_event_with_params_constructor(t *testing.T) {
	tests := []struct {
		name  string
		f     func(influxdb2.Client, []AirlineOffer)
		datas []AirlineOffer
	}{
		{
			name: "Write new record with line protocol",
			// Your data Points
			datas: mockData,
			f: func(c influxdb2.Client, datas []AirlineOffer) {
				// Send all the data to the DB
				for _, data := range datas {
					write_event_with_params_constror(c, data, testBucket)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// helper to initialise and clean the database
			client := init_testDB(t)
			// call function under test
			tt.f(client, tt.datas)
			// TODO Validate the data
		})
	}
}

func init_testDB(t *testing.T) DBClient {
	t.Helper() // Tells `go test` that this is an helper

	godotenv.Load("../../test_influxdb.env") //load environement variable
	client, err := ConnectToInfluxDB()       // create the client

	if err != nil {
		t.Errorf("impossible to connect to DB")
	}

	// Clean the database by deleting the Bucket
	ctx := context.Background()
	bucketsAPI := client.BucketsAPI()
	dBucket, err := bucketsAPI.FindBucketByName(ctx, testBucket)
	if err == nil {
		client.BucketsAPI().DeleteBucketWithID(context.Background(), *dBucket.Id)
	}

	// create new empty Bucket
	dOrg, _ := client.OrganizationsAPI().FindOrganizationByName(ctx, org)
	_, err = client.BucketsAPI().CreateBucketWithNameWithID(ctx, *dOrg.Id, testBucket)

	if err != nil {
		t.Errorf("impossible to new create Bucket")
	}

	return client
}

func InitDB(envPath string) influxdb2.Client {
	err := godotenv.Load(envPath) //load environement variable
	if err != nil {
		log.Println(err)
		log.Println("Couldn't load ENVARS")
	}

	client, err := ConnectToInfluxDB() // create the client

	if err != nil {
		log.Println(err)
		log.Fatal("impossible to connect to DB")
	}

	ctx := context.Background()
	bucketsAPI := client.BucketsAPI()
	existingBucket, err := bucketsAPI.FindBucketByName(ctx, Bucket)
	if existingBucket == nil {
		log.Println("Didn't find an existing bucket. Creating a new one.")
		// create new empty Bucket
		dOrg, _ := client.OrganizationsAPI().FindOrganizationByName(ctx, org)
		_, err = client.BucketsAPI().CreateBucketWithNameWithID(ctx, *dOrg.Id, Bucket)

		if err != nil {
			log.Fatal("impossible to new create Bucket")
		}
	} else {
		log.Printf("Will use existing bucket with ID: %s \n", *existingBucket.Id)
	}

	return client
}

func write_event_with_line_protocol(client influxdb2.Client, t AirlineOffer, dbBucket string) {
	// get non-blocking write client
	writeAPI := client.WriteAPI(org, dbBucket)
	// write line protocol
	writeAPI.WriteRecord(
		fmt.Sprintf("airlineOffer,unit=euro,fromAirport=%s,toAirport=%s,departureDate=%s,returnDate=%s, price=%f,url=%s %d",
			t.FromAirport, t.ToAirport, t.DepartureDate, t.ReturnDate, t.Price, t.Url, t.CreatedOn.Unix()),
	)
	// Flush writes
	writeAPI.Flush()
}

func Write_event_with_fluent_Style(client influxdb2.Client, t AirlineOffer, dbBucket string) {
	log.Println("Writing offer to DB.")
	// Use blocking write client for writes to desired Bucket
	writeAPI := client.WriteAPI(org, dbBucket)
	// create point using fluent style
	p := influxdb2.NewPointWithMeasurement("airlineOffer").
		AddTag("unit", "euro").
		AddField("url", t.Url).
		AddTag("fromAirport", t.FromAirport).
		AddTag("toAirport", t.ToAirport).
		AddTag("departureDate", t.DepartureDate.Format("2006-01-02")).
		AddTag("returnDate", t.ReturnDate.Format("2006-01-02")).
		AddField("price", t.Price).
		SetTime(t.CreatedOn)
	writeAPI.WritePoint(p)
	// Flush writes
	writeAPI.Flush()
}

func write_event_with_params_constror(client influxdb2.Client, t AirlineOffer, dbBucket string) {
	// Use blocking write client for writes to desired Bucket
	writeAPI := client.WriteAPI(org, dbBucket)
	// Create point using full params constructor
	p := influxdb2.NewPoint("airlineOffer",
		map[string]string{"unit": "euro"},
		map[string]interface{}{
			"url": t.Url, "fromAirport": t.FromAirport, "toAirport": t.ToAirport, "departureDate": t.DepartureDate, "returnDate": t.ReturnDate, "price": t.Price,
		},
		t.CreatedOn)
	writeAPI.WritePoint(p)
	// Flush writes
	writeAPI.Flush()
}

func write_event_with_blocking_write(client influxdb2.Client, dbBucket string) {
	// Get blocking write client
	writeAPI := client.WriteAPIBlocking(org, dbBucket)

	// write line protocol
	writeAPI.WriteRecord(context.Background(), fmt.Sprintf("stat,unit=temperature1 avg=%f,max=%f", 23.5, 45.0))
}

func read_events_as_query_table_result(client influxdb2.Client, dbBucket string) map[time.Time]AirlineOffer {

	// Get query client
	queryAPI := client.QueryAPI(org)

	// Query. You need to change a bit the Query from the Query Builder
	// Otherwise it won't work
	fluxQuery := fmt.Sprintf(`from(bucket: "` + dbBucket + `")
|> range(start: -1h)
|> filter(fn: (r) => r["_measurement"] == "airlineOffer")
|> yield(name: "mean")`)

	result, err := queryAPI.Query(context.Background(), fluxQuery)

	// Putting back the data in share requires a bit of work
	var resultPoints map[time.Time]AirlineOffer
	resultPoints = make(map[time.Time]AirlineOffer)

	if err == nil {
		// Iterate over query response
		for result.Next() {
			// Notice when group key has changed
			if result.TableChanged() {
				fmt.Printf("table: %s\n", result.TableMetadata().String())
			}

			val, ok := resultPoints[result.Record().Time()]

			if !ok {
				val = AirlineOffer{
					Url: fmt.Sprintf("%v", result.Record().ValueByKey("url")),
				}
			}

			switch field := result.Record().Field(); field {
			case "url":
				val.Url = result.Record().Value().(string)
			case "price":
				val.Price = result.Record().Value().(float64)
			default:
				fmt.Printf("unrecognized field %s.\n", field)
			}

			for k, v := range result.Record().Values() {
				switch k {
				case "fromAirport":
                    val.FromAirport = v.(string)
				case "toAirport":
					val.ToAirport = v.(string)
				case "departureDate":
                    val.DepartureDate, _ = time.Parse("2006-01-02", v.(string))
				case "returnDate":
					val.ReturnDate, _ = time.Parse("2006-01-02", v.(string))
				default:
					fmt.Printf("unrecognized field %s.\n", k)
				}
			}
            val.CreatedOn = result.Record().Time()
			resultPoints[result.Record().Time()] = val

		}
		// check for an error
		if result.Err() != nil {
			fmt.Printf("query parsing error: %s\n", result.Err().Error())
		}
	} else {
		panic(err)
	}

	return resultPoints

}

func read_events_as_raw_string(client influxdb2.Client, dbBucket string) {
	// Get query client
	queryAPI := client.QueryAPI(org)

	// Query
	fluxQuery := fmt.Sprintf(`from(bucket: "` + dbBucket + `")
|> range(start: -1h)
|> filter(fn: (r) => r["_measurement"] == "airlineOffer")
|> yield(name: "mean")`)

	result, err := queryAPI.QueryRaw(context.Background(), fluxQuery, influxdb2.DefaultDialect())
	if err == nil {
		fmt.Println("QueryResult:")
		fmt.Println(result)
	} else {
		panic(err)
	}
}
