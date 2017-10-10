package stations

import (
	"bufio"
	"errors"
	"regexp"
	"strings"
)

// Port ...
type Port struct {
	Link    string `json:"link"`
	Speed   string `json:"speed,omitempty"`
	Gateway bool   `json:"gateway,omitempty"`
}

// SwitchState ...
// key is the port number as string
type SwitchState map[string]Port

// ParseSwitch ...
// The second return value is a map from MAC address to port number.
func ParseSwitch(text string, excludePorts ...string) (SwitchState, map[string]string, error) {
	ports := SwitchState{}
	macs := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(text))

	exclPorts := make(map[string]bool, len(excludePorts))
	for _, p := range excludePorts {
		exclPorts[p] = true
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "link: ") {
			pNum, port, err := parsePort(line)
			if err == nil && !exclPorts[pNum] {
				ports[pNum] = port
			}
		} else if strings.HasPrefix(line, "Port ") {
			pNum, mac, err := parseMAC(line)
			if err == nil && !exclPorts[pNum] {
				macs[mac] = pNum
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	return ports, macs, nil
}

var errInvalidLine = errors.New("invalid line")

func parsePort(line string) (string, Port, error) {
	// remove "link: " prefix and split into three parts: port, link, speed
	parts := strings.SplitN(line[6:], " ", 3)

	// valid entries need to have at least port and link
	if len(parts) < 2 {
		return "", Port{}, errInvalidLine
	}

	p := Port{Link: parts[1][5:]} // remove "link:"
	if len(parts) > 2 {
		p.Speed = parts[2][6:] // remove "speed:"
	}

	return parts[0][5:], p, nil
}

var port2macRegex = regexp.MustCompile(`^Port\s+(\d+):\s+MAC\s+(([a-fA-F\d][a-fA-F\d]:?){6})`)

func parseMAC(line string) (string, string, error) {
	m := port2macRegex.FindStringSubmatch(line)

	if m == nil || len(m) < 3 {
		return "", "", errInvalidLine
	}

	return m[1], m[2], nil
}
