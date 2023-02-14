package database

import (
    "context"
    "fmt"
    "reflect"
    "testing"
    "time"

    influxdb2 "github.com/influxdata/influxdb-client-go/v2"
    "github.com/joho/godotenv"

)

const bucket = "airliner"
const org = "iot"
type AirlineOffer struct {
    Url string
    FromAirport string
    ToAirport string
    DepartureDate time.Time
    ReturnDate time.Time
    Price float64
    CreatedOn time.Time
}
var mockData = []AirlineOffer{
                {
                    Url: "www.example.com",
                    FromAirport: "MUC",
                    ToAirport: "LIS",
                    DepartureDate: time.Date(2023, 3, 15, 11, 25, 0, 0, time.UTC),
                    ReturnDate: time.Date(2023, 4, 1, 10, 0, 0, 0, time.UTC),
                    Price: 242.66,
                    CreatedOn: time.Now().Round(1 * time.Second).UTC(),
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
                    write_event_with_line_protocol(c, data)
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
                    results := read_events_as_query_table_result(client)
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
                    read_events_as_raw_string(client)

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
                    write_event_with_params_constror(c, data)
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


func init_testDB(t *testing.T) influxdb2.Client {
    t.Helper()                                 // Tells `go test` that this is an helper
    godotenv.Load("../../test_influxdb.env")   //load environement variable
    client, err := ConnectToInfluxDB() // create the client

    if err != nil {
        t.Errorf("impossible to connect to DB")
    }

    // Clean the database by deleting the bucket
    ctx := context.Background()
    bucketsAPI := client.BucketsAPI()
    dBucket, err := bucketsAPI.FindBucketByName(ctx, bucket)
    if err == nil {
        client.BucketsAPI().DeleteBucketWithID(context.Background(), *dBucket.Id)
    }

    // create new empty bucket
    dOrg, _ := client.OrganizationsAPI().FindOrganizationByName(ctx, org)
    _, err = client.BucketsAPI().CreateBucketWithNameWithID(ctx, *dOrg.Id, bucket)

    if err != nil {
        t.Errorf("impossible to new create bucket")
    }

    return client
}

func write_event_with_line_protocol(client influxdb2.Client, t AirlineOffer) {
    // get non-blocking write client
    writeAPI := client.WriteAPI(org, bucket)
    // write line protocol
    writeAPI.WriteRecord(
        fmt.Sprintf("airlineOffer,unit=euro,url=%s,fromAirport=%s,toAirport=%s,departureDate=%s,returnDate=%s,price=%f,createdOn=%s",
            t.Url, t.FromAirport, t.ToAirport, t.DepartureDate, t.ReturnDate, t.Price, t.CreatedOn),
        )
    // Flush writes
    writeAPI.Flush()
}


func Write_event_with_fluent_Style(client influxdb2.Client, t AirlineOffer) {
    // Use blocking write client for writes to desired bucket
    writeAPI := client.WriteAPI(org, bucket)
    // create point using fluent style
    p := influxdb2.NewPointWithMeasurement("airlineOffer").
        AddTag("unit", "euro").
        AddField("url", t.Url).
        AddField("fromAirport", t.FromAirport).
        AddField("toAirport", t.ToAirport).
        AddField("departureDate", t.DepartureDate.Unix()).
        AddField("returnDate", t.ReturnDate.Unix()).
        AddField("price", t.Price).
        AddField("createdOn", t.CreatedOn.Unix()).
        SetTime(time.Now())
    writeAPI.WritePoint(p)
    // Flush writes
    writeAPI.Flush()
}

func write_event_with_params_constror(client influxdb2.Client, t AirlineOffer) {
    // Use blocking write client for writes to desired bucket
    writeAPI := client.WriteAPI(org, bucket)
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

func write_event_with_blocking_write(client influxdb2.Client) {
    // Get blocking write client
    writeAPI := client.WriteAPIBlocking(org, bucket)

    // write line protocol
    writeAPI.WriteRecord(context.Background(), fmt.Sprintf("stat,unit=temperature1 avg=%f,max=%f", 23.5, 45.0))
}

func read_events_as_query_table_result(client influxdb2.Client) map[time.Time]AirlineOffer {

    // Get query client
    queryAPI := client.QueryAPI(org)

    // Query. You need to change a bit the Query from the Query Builder
    // Otherwise it won't work
    fluxQuery := fmt.Sprintf(`from(bucket: "airliner")
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
            case "fromAirport":
                val.FromAirport = result.Record().Value().(string)
                case "toAirport":
                    val.ToAirport = result.Record().Value().(string)
                case "departureDate":
                    val.DepartureDate = time.Unix(result.Record().Value().(int64), 0).UTC()
            case "returnDate":
                val.ReturnDate = time.Unix(result.Record().Value().(int64), 0).UTC()
            case "price":
                val.Price = result.Record().Value().(float64)
            case "createdOn":
                val.CreatedOn = time.Unix(result.Record().Value().(int64), 0).UTC()
                    default:
                        fmt.Printf("unrecognized field %s.\n", field)
            }

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

func read_events_as_raw_string(client influxdb2.Client) {
    // Get query client
    queryAPI := client.QueryAPI(org)

    // Query
    fluxQuery := fmt.Sprintf(`from(bucket: "airliner")
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
