package web

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/umahmood/haversine"

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

func getIPInfo(addr string) results.IPInfoResponse {
	var ret results.IPInfoResponse
	resp, err := http.DefaultClient.Get(getIPInfoURL(addr))
	if err != nil {
		slog.Error("getting response from ipinfo.io", slog.Any("error", err))
		return ret
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("reading response from ipinfo.io", slog.Any("error", err))
		return ret
	}
	defer resp.Body.Close()

	if err := json.Unmarshal(raw, &ret); err != nil {
		slog.Error("parsing response from ipinfo.io", slog.Any("error", err))
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
	resp, err := http.DefaultClient.Get(getIPInfoURL(""))
	if err != nil {
		slog.Error("getting response from ipinfo.io", slog.Any("error", err))
		return
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("reading response from ipinfo.io", slog.Any("error", err))
		return
	}
	defer resp.Body.Close()

	if err := json.Unmarshal(raw, &ret); err != nil {
		slog.Error("parsing response from ipinfo.io", slog.Any("error", err))
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
