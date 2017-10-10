package stations

import (
	"regexp"
	"strconv"
)

// WifiStation is the structured representation of the output of "iw dev $DEVICE station dump" for
// one particular station (i.e. MAC address). Using interface{} as the map value is not great
// but the alternative is defining a huge struct, similar to Survey, except worse.
// Most of the values are strings and never processed by this code. The only exception is
// "inactive time", which is converted to an integer and normalized to seconds.
type WifiStation map[string]interface{}

// WifiPollMessage ...
type WifiPollMessage struct {
	Version   int64 `json:"version"`
	Timestmap int64 `json:"timestamp"`

	Interfaces map[string]struct {
		Info     string `json:"info"`
		Stations string `json:"stations"`
		MPath    string `json:"mpath"`
		Survey   string `json:"survey"`
	} `json:"dev"`
}

// WifiEventMessage ...
type WifiEventMessage struct {
	Action string `json:"action"`
	MAC    string `json:"station"`
}

var macRegex = regexp.MustCompile(`([a-fA-F\d][a-fA-F\d]:?){6}`)

// ParseWifiStations ...
func ParseWifiStations(text, iface string) (map[string]WifiStation, error) {
	chunks, err := chunkLinesByPrefix(text, "Station", true)
	if err != nil {
		return nil, err
	}

	mode, radio := parseInterfaceName(iface)
	stations := make(map[string]WifiStation)
	var current WifiStation

	for _, chunk := range chunks {
		for i, line := range chunk {

			if i == 0 {
				mac := macRegex.FindString(line)
				current = WifiStation{"mac": mac, "vendor": vendorFromMAC(mac)}
				stations[mac] = current

				if len(mode) > 0 {
					current["mode"] = mode
				}
				if len(radio) > 0 {
					current["radio"] = radio
				}
			} else {
				key, value := splitLine(line)
				current[key] = value
			}
		}

		inactiveTime, exists := current["inactive time"].(string)
		if exists {
			current["inactive_time"] = normalizeInactiveTime(inactiveTime)
			delete(current, "inactive time")
		} else {
			current["inactive_time"] = 0
		}
	}
	return stations, nil
}

var inactiveTimeRegex = regexp.MustCompile(`(\d+)\s+(\S+)`)

func normalizeInactiveTime(s string) int {
	submatches := inactiveTimeRegex.FindStringSubmatch(s)
	if len(submatches) < 3 {
		return 0
	}

	t, err := strconv.Atoi(submatches[1])
	if err != nil {
		return 0
	}

	if submatches[2] == "ms" {
		return t / 1000
	}
	return t
}

// merge b into a and return it
func merge(a, b map[string]WifiStation) map[string]WifiStation {
	for k, v := range b {
		a[k] = v
	}
	return a
}
