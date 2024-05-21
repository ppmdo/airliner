package database

import (
	"context"
	"testing"

	"github.com/influxdata/influxdb-client-go/v2/domain"
	"github.com/joho/godotenv"
)

func Test_connectToInfluxDB(t *testing.T) {

	//load environment variable from a file for test purposes
	godotenv.Load("../../test_influxdb.env")

	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "Successful connection to InfluxDB",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConnectToInfluxDB()
			if (err != nil) != tt.wantErr {
				t.Errorf("ConnectToInfluxDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			health, err := got.Health(context.Background())
			if (err != nil) && health.Status == domain.HealthCheckStatusPass {
				t.Errorf("connectToInfluxDB() error. database not healthy")
				return
			}
			got.Close()
		})
	}
}
