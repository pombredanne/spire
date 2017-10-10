package stations

import (
	"bufio"
	"os"
	"strings"
)

var ouiDB map[string]string

func init() {
	ouiDB = make(map[string]string)

	if f, err := os.Open("oui.txt"); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)

		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "#") {
				continue
			}

			if parts := strings.SplitN(line, " ", 2); len(parts) >= 2 {
				ouiDB[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}

		if err := scanner.Err(); err != nil {
			panic(err)
		}
	} else {
		panic(err)
	}
}

func chunkLinesByPrefix(text, prefix string, includeFirstLine bool) ([][]string, error) {
	res := [][]string{}
	var current []string
	scanner := bufio.NewScanner(strings.NewReader(text))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		if strings.HasPrefix(line, prefix) {
			if current != nil && len(current) > 0 {
				res = append(res, current)
			}

			if includeFirstLine {
				current = []string{line}
			} else {
				current = []string{}
			}
		} else {
			current = append(current, line)
		}
	}

	if len(current) > 0 {
		res = append(res, current)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return res, nil
}

func splitLine(line string) (string, string) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", ""
	}

	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func parseInterfaceName(iface string) (mode string, radio string) {
	parts := strings.Split(iface, "-")

	if len(parts) > 1 {
		mode = parts[1]

		if len(parts) > 2 {
			radio = parts[2]
		}
	}
	return
}

func vendorFromMAC(mac string) string {
	return ouiDB[mac[:8]]
}
