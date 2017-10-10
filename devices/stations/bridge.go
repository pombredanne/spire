package stations

import (
	"bufio"
	"strconv"
	"strings"
)

// BridgeInfo ...
type BridgeInfo struct {
	Local bool
	Age   float64
}

// ParseBridgeMACs parses the string in "net" messages at msg["bridge"]["macs"]["public"] (and "private" resp.)
// Keys in the returned map are MAC addresses.
func ParseBridgeMACs(text string) (map[string]BridgeInfo, error) {
	res := make(map[string]BridgeInfo)
	scanner := bufio.NewScanner(strings.NewReader(text))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) != 5 {
			return res, nil
		}

		if strings.HasPrefix(parts[0], "port no") {
			continue
		}

		age, err := strconv.ParseFloat(strings.TrimSpace(parts[4]), 64)
		if err != nil {
			return nil, err
		}

		res[strings.TrimSpace(parts[1])] = BridgeInfo{
			Local: strings.TrimSpace(parts[2]) == "yes",
			Age:   age,
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return res, nil
}
