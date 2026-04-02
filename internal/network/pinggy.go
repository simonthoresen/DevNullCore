package network

import (
	"bufio"
	"os"
	"strings"
)

// PinggyStatus holds the parsed contents of the Pinggy status file.
type PinggyStatus struct {
	LogLines    []string
	TcpAddress  string
	JoinCommand string
}

// ReadPinggyStatus reads and parses the Pinggy status file.
func ReadPinggyStatus(statusFile string) (*PinggyStatus, error) {
	file, err := os.Open(statusFile)
	if err != nil {
		return nil, err
	}
	defer file.Close() //nolint:errcheck

	status := &PinggyStatus{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "PINGGY_LOG="):
			message := strings.TrimSpace(strings.TrimPrefix(line, "PINGGY_LOG="))
			if message != "" {
				status.LogLines = append(status.LogLines, message)
			}
		case strings.HasPrefix(line, "PINGGY_TCP="):
			status.TcpAddress = strings.TrimSpace(strings.TrimPrefix(line, "PINGGY_TCP="))
		case strings.HasPrefix(line, "PINGGY_JOIN="):
			status.JoinCommand = strings.TrimSpace(strings.TrimPrefix(line, "PINGGY_JOIN="))
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return status, nil
}
