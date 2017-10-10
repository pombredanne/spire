package stations

import (
	"bufio"
	"encoding/json"
	"log"
	"strings"
)

// DHCPClient ...
type DHCPClient struct {
	MAC      string `json:"m"`
	IP       string `json:"ip"`
	Hostname string `json:"n"`
	TTL      string `json:"l"`
}

// DHCPState ...
type DHCPState map[string][]DHCPClient

// Get ...
func (s DHCPState) Get(iface string) []DHCPClient {
	return s[iface]
}

// ParseDHCP ...
func ParseDHCP(text []byte) (DHCPState, error) {
	var state DHCPState

	if err := json.Unmarshal(text, &state); err != nil {
		return parseLegacyDHCPMessage(string(text))
	}

	return state, nil
}

func parseLegacyDHCPMessage(text string) (DHCPState, error) {
	res := make(DHCPState)
	scanner := bufio.NewScanner(strings.NewReader(text))
	var iface string

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "\t")

		if len(parts) == 1 {
			iface = parts[0]
			res[iface] = []DHCPClient{}
		} else if len(parts) == 4 {
			c := DHCPClient{
				MAC:      parts[0],
				IP:       parts[1],
				Hostname: parts[3],
				TTL:      parts[2],
			}
			res[iface] = append(res[iface], c)
		} else {
			log.Println("[dhcp] ignoring invalid line in legacy message:", line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

// MarshalJSON ...
func (c DHCPClient) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		MAC      string `json:"mac"`
		IP       string `json:"ip"`
		Hostname string `json:"host_name"`
		TTL      string `json:"ttl"`
	}{
		MAC:      c.MAC,
		IP:       c.IP,
		Hostname: c.Hostname,
		TTL:      c.TTL,
	})
}
