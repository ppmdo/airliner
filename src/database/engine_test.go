package database

import (
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"reflect"
	"testing"
	"time"
)

func Test_write_event_with_fluent_Style(t *testing.T) {
	tests := []struct {
		name  string
		f     func(influxdb2.Client, []AirlineOffer)
		datas []AirlineOffer
	}{
		{
			name: "Write new record with fluent style",
			// Your data Points
			datas: mockData,
			f: func(c influxdb2.Client, datas []AirlineOffer) {
				// Send all the data to the DB
				for _, data := range datas {
					Write_event_with_fluent_Style(c, data, testBucket)
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
