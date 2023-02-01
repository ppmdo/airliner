package kayak

import (
    "fmt"
    "sync"
    "time"
    "testing"
    md "airliner/model"
)

func TestCreatePayloads(t *testing.T) {
    var wg sync.WaitGroup
    initialDate, _ := time.Parse("2006-01-02", "2023-01-01")
    tripLength := 10
    daysToLookAhead := 5
    result := make([]*md.Payload, 5)

    ch := make(chan *md.Payload)

    wg.Add(1)
    go CreatePayloads("LIS", "MUC", initialDate, tripLength, daysToLookAhead, ch, &wg)

    i := 0
    for v := range ch {
        result[i] = v
        i++
    }

    wg.Wait()

    expected := []struct {from string; to string; departure time.Time; returndate time.Time; id int} {
        {"LIS", "MUC", createDate("2023-01-01"), createDate("2023-01-11"), 0},
        {"LIS", "MUC", createDate("2023-01-02"), createDate("2023-01-12"), 1},
        {"LIS", "MUC", createDate("2023-01-03"), createDate("2023-01-13"), 2},
        {"LIS", "MUC", createDate("2023-01-04"), createDate("2023-01-14"), 3},
        {"LIS", "MUC", createDate("2023-01-05"), createDate("2023-01-15"), 4},
    }

    for i, e := range expected {
        if result[i].DepartureDate != e.departure {
            t.Log(fmt.Sprintf("DepartureDate should be %s but got %s", e.departure, result[i].DepartureDate))
            t.Fail()
        }
        if result[i].ReturnDate != e.returndate {
            t.Log(fmt.Sprintf("ReturnDate should be %s but got %s", e.returndate, result[i].ReturnDate))
            t.Fail()
        }
        if result[i].Id != e.id {
            t.Log(fmt.Sprintf("Id should be %d but got %d", e.id, result[i].Id))
            t.Fail()
        }
    }
}

func createDate(input string) time.Time {
    v, _ := time.Parse("2006-01-02", input)
    return v
}