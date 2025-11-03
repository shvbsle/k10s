package k8s

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// LogLine represents a single line from container logs.
type LogLine struct {
	LineNum   int
	Timestamp string
	Content   string
}

// GetContainerLogs retrieves the last N lines of logs for a specific container.
// Returns a slice of LogLine structs with line numbers, optional timestamps, and content.
// Returns an error if the client is not connected or if the API request fails.
func (c *Client) GetContainerLogs(podName, namespace, containerName string, tailLines int, withTimestamps bool) ([]LogLine, error) {
	if !c.isConnected || c.clientset == nil {
		return nil, fmt.Errorf("not connected to cluster")
	}

	ctx := context.Background()

	tail := int64(tailLines)
	logOptions := &corev1.PodLogOptions{
		Container:  containerName,
		TailLines:  &tail,
		Timestamps: withTimestamps,
	}

	req := c.clientset.CoreV1().Pods(namespace).GetLogs(podName, logOptions)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		c.isConnected = false
		return nil, err
	}
	defer func() {
		if closeErr := podLogs.Close(); closeErr != nil {
			slog.Error("error closing log stream", "error", closeErr)
		}
	}()

	var logLines []LogLine
	scanner := bufio.NewScanner(podLogs)
	lineNum := 1

	for scanner.Scan() {
		line := scanner.Text()

		var timestamp string
		var logContent string

		if withTimestamps {
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 {
				timestamp = parts[0]
				logContent = parts[1]
			} else {
				logContent = line
			}
		} else {
			logContent = line
		}

		logLines = append(logLines, LogLine{
			LineNum:   lineNum,
			Timestamp: timestamp,
			Content:   logContent,
		})
		lineNum++
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return nil, err
	}

	return logLines, nil
}
