package web

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/umahmood/haversine"

	resty "github.com/go-resty/resty/v2"

	"github.com/librespeed/speedtest/config"
	"github.com/librespeed/speedtest/results"
)

var (
	serverCoord haversine.Coord
)

func getRandomData(length int) []byte {
	data := make([]byte, length)
	if _, err := rand.Read(data); err != nil {
		panic(fmt.Errorf("failed to generate random data: %s", err))
	}
	return data
}

func getIPInfoURL(address string) string {
	apiKey := config.LoadedConfig().IPInfoAPIKey

	ipInfoURL := `https://ipinfo.io/%s/json`
	if address != "" {
		ipInfoURL = fmt.Sprintf(ipInfoURL, address)
	} else {
		ipInfoURL = "https://ipinfo.io/json"
	}

	if apiKey != "" {
		ipInfoURL += "?token=" + apiKey
	}

	return ipInfoURL
}

var httpClient = resty.New()

func init() {
	httpClient.OnError(func(req *resty.Request, err error) {
		if v, ok := err.(*resty.ResponseError); ok {
			slog.Error("resty error", slog.Any("error", v.Err), slog.Any("response", v.Response))
			return
		}
		slog.Error("resty error", slog.Any("error", err))
	})
}

func getIPInfo(addr string) results.IPInfoResponse {
	var ret results.IPInfoResponse
	_, err := httpClient.R().
		SetResult(&ret).
		Get(getIPInfoURL(addr))

	if err != nil {
		slog.Error("getting response from ipinfo.io", slog.Any("error", err))
		return ret
	}
	return ret
}

func SetServerLocation(conf *config.Config) {
	if conf.ServerLat != 0 || conf.ServerLng != 0 {
		slog.Info("Configured server coordinates",
			slog.Float64("lat", conf.ServerLat),
			slog.Float64("lng", conf.ServerLng))

		serverCoord.Lat = conf.ServerLat
		serverCoord.Lon = conf.ServerLng
		return
	}

	var ret results.IPInfoResponse

	_, err := httpClient.R().
		SetResult(&ret).
		Get(getIPInfoURL(""))
	if err != nil {
		slog.Error("getting response from ipinfo.io", slog.Any("error", err))
		return
	}

	if ret.Location != "" {
		serverCoord, err = parseLocationString(ret.Location)
		if err != nil {
			slog.Error("Cannot get server coordinates", slog.Any("error", err))
			return
		}
	}
	slog.Info("Fetched server coordinates",
		slog.Float64("lat", serverCoord.Lat),
		slog.Float64("lng", serverCoord.Lon),
	)
}

func parseLocationString(location string) (haversine.Coord, error) {
	var coord haversine.Coord

	parts := strings.Split(location, ",")
	if len(parts) != 2 {
		slog.Error("unknown location format", slog.String("location", location))
		return coord, fmt.Errorf("unknown location format: %s", location)
	}

	lat, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		slog.Error("parsing latitude", slog.String("latitude", parts[0]))
		return coord, err
	}

	lng, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		slog.Error("parsing longitude", slog.String("longitude", parts[1]))
		return coord, err
	}

	coord.Lat = lat
	coord.Lon = lng

	return coord, nil
}

func calculateDistance(clientLocation string, unit string) string {
	clientCoord, err := parseLocationString(clientLocation)
	if err != nil {
		slog.Error("parsing client coordinates", slog.Any("error", err))
		return ""
	}

	dist, km := haversine.Distance(clientCoord, serverCoord)
	unitString := " mi"

	switch unit {
	case "km":
		dist = km
		unitString = " km"
	case "NM":
		dist = km * 0.539957
		unitString = " NM"
	}

	return fmt.Sprintf("%.2f%s", dist, unitString)
}
