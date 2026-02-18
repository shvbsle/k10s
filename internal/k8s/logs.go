package k8s

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/shvbsle/k10s/internal/log"
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
			log.G().Error("error closing log stream", "error", closeErr)
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

// StreamContainerLogs starts streaming logs and sends lines to the provided channel.
// Returns a cancel function to stop the stream.
func (c *Client) StreamContainerLogs(
	podName, namespace, containerName string,
	tailLines int,
	withTimestamps bool,
	linesChan chan<- LogLine,
) (cancel func(), err error) {
	if !c.isConnected || c.clientset == nil {
		return nil, fmt.Errorf("not connected to cluster")
	}

	ctx, cancelFunc := context.WithCancel(context.Background())

	tail := int64(tailLines)
	logOptions := &corev1.PodLogOptions{
		Container:  containerName,
		TailLines:  &tail,
		Timestamps: withTimestamps,
		Follow:     true, // Enable streaming
	}

	req := c.clientset.CoreV1().Pods(namespace).GetLogs(podName, logOptions)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		cancelFunc()
		// Don't mark disconnected — streaming may fail even when the cluster is reachable
		// (e.g., Follow not supported for certain pod states)
		return nil, err
	}

	// Start goroutine to read from stream
	go func() {
		defer func() {
			if closeErr := podLogs.Close(); closeErr != nil {
				log.G().Error("error closing log stream", "error", closeErr)
			}
			close(linesChan)
		}()

		scanner := bufio.NewScanner(podLogs)
		lineNum := 1

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
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

				logLine := LogLine{
					LineNum:   lineNum,
					Timestamp: timestamp,
					Content:   logContent,
				}

				select {
				case linesChan <- logLine:
					lineNum++
				case <-ctx.Done():
					return
				}
			}
		}

		if err := scanner.Err(); err != nil && err != io.EOF {
			log.G().Error("error reading log stream", "error", err)
		}
	}()

	return cancelFunc, nil
}

// CalculateTailLines calculates the initial tail lines based on viewport height
func CalculateTailLines(viewportHeight int) int {
	const minTailLines = 100
	const tailLinesMultiplier = 2

	tailLines := viewportHeight * tailLinesMultiplier
	if tailLines < minTailLines {
		tailLines = minTailLines
	}
	return tailLines
}
