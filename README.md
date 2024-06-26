# The Airliner ✈️  

A Go based CLI tool to find great deals on flight tickets! The application saves results to an InfluxDB and sends notifications via a Telegram Bot. 🛫🔍📲

The tool scrapes data from online providers and finds the best flight tickets given a set of criteria:
- Departure City 🏙️
- Destination City 🌆
- Direct / Non-Direct Flights 🔄
- Length of Stay ⏳
- Initial Search Date 📅
- Days to Look Ahead 🔍👀

The application will look for flights starting from the Initial Search Date, plus the Days To Look Ahead and notify about the best option for the given criteria. The Telegram notification includes a description of the flight plan, the price and a screenshot of the found offer. 📲

Searches can run concurrently if your hardware allows for it (I designed this to run on a Raspberry PI 3). 


# Requirements
The following environment variables are needed:
```bash
INFLUXDB_USERNAME=...
INFLUXDB_PASSWORD=...
INFLUXDB_TOKEN=...
INFLUXDB_URL=...
TELEGRAM_CHAT_ID=...
TELEGRAM_BOT_TOKEN=...
```

# Usage

The application takes the following arguments:
```
  -concurrency int
        max num. of concurrent jobs (default 2)

  -direct
        set to false to look for non-direct flights too (default true)

  -duration int
        journey duration (default -1)

  -from string
        3 letter uppercase code for the city flying from.

  -look-ahead int
        number of days to look ahead (default -1)

  -start-date string
        initial day to lookup

  -to string
        3 letter uppercase code for the city flying to.
```
