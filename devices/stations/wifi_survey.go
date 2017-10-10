package stations

import (
	"strconv"
	"strings"
)

// WifiSurvey ...
type WifiSurvey struct {
	Frequency           string `json:"frequency"`
	ChannelActiveTime   string `json:"channel active time,omitempty"`
	ChannelBusyTime     string `json:"channel busy time,omitempty"`
	ChannelReceiveTime  string `json:"channel receive time,omitempty"`
	ChannelTransmitTime string `json:"channel transmit time,omitempty"`
	Noise               string `json:"noise,omitempty"`
	InUse               bool   `json:"in use,omitempty"`
}

// ParseWifiSurvey ...
func ParseWifiSurvey(survey string) (map[string]*WifiSurvey, error) {
	chunks, err := chunkLinesByPrefix(survey, "Survey", false)
	if err != nil {
		return nil, err
	}

	surveys := make(map[string]*WifiSurvey)
	var current *WifiSurvey

	for _, chunk := range chunks {
		current = new(WifiSurvey)

		for _, line := range chunk {
			key, value := splitLine(line)
			switch key {
			case "frequency":
				current.Frequency, current.InUse = parseFrequency(value)
			case "channel active time":
				current.ChannelActiveTime = value
			case "channel busy time":
				current.ChannelBusyTime = value
			case "channel recive time":
				current.ChannelReceiveTime = value
			case "channel transmit time":
				current.ChannelTransmitTime = value
			case "noise":
				current.Noise = value
			default:
				continue
			}
		}

		surveys[current.Frequency] = chooseWifiSurvey(current, surveys[current.Frequency])
	}

	return surveys, nil
}

func compileWifiSurveyMessage(msg *WifiPollMessage) (map[string]*WifiSurvey, error) {
	res := make(map[string]*WifiSurvey)

	for _, d := range msg.Interfaces {

		surveys, err := ParseWifiSurvey(d.Survey)
		if err != nil {
			return nil, err
		}

		for freq, s := range surveys {
			res[freq] = chooseWifiSurvey(s, res[freq])
		}
	}

	return res, nil
}

func parseFrequency(val string) (string, bool) {
	if strings.HasSuffix(val, "[in use]") {
		return val[:8], true
	}
	return val, false
}

func chooseWifiSurvey(a, b *WifiSurvey) *WifiSurvey {
	if b == nil || parseActiveTime(a) > parseActiveTime(b) {
		return a
	}
	return b
}

func parseActiveTime(s *WifiSurvey) int {
	cat := strings.Replace(s.ChannelActiveTime, " ms", "", 1)
	i, err := strconv.Atoi(strings.TrimSpace(cat))
	if err != nil {
		return 0
	}
	return i
}
