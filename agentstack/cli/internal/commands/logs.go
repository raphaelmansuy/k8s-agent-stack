/*
 * Copyright 2025 Raphaël MANSUY
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/raphaelmansuy/agentstack/cli/internal/output"
	"github.com/raphaelmansuy/agentstack/sdk"
)

func newLogsCmd() *cobra.Command {
	var follow bool
	var tail int
	var timestamps bool
	cmd := &cobra.Command{
		Use:   "logs <agent-or-deployment>",
		Short: "View logs from an agent or deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			var agentID, deploymentID string
			if strings.HasPrefix(target, "deploy/") || strings.HasPrefix(target, "deployment/") {
				parts := strings.SplitN(target, "/", 2)
				deploymentID = parts[1]
			} else {
				agentID = target
			}
			if follow {
				return followLogs(cmd, agentID, deploymentID, tail, timestamps)
			}
			return viewLogs(cmd, agentID, deploymentID, tail, timestamps)
		},
	}
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow log output")
	cmd.Flags().IntVar(&tail, "tail", 100, "number of lines to show")
	cmd.Flags().BoolVar(&timestamps, "timestamps", false, "show timestamps")
	return cmd
}

func viewLogs(cmd *cobra.Command, agentID, deploymentID string, limit int, showTimestamps bool) error {
	ctx := cmd.Context()
	var resp *sdk.ListLogsResponse
	var err error
	if deploymentID != "" {
		resp, err = client.Logs.GetDeploymentLogs(ctx, deploymentID, limit)
	} else {
		resp, err = client.Logs.GetAgentLogs(ctx, agentID, limit)
	}
	if err != nil {
		return fmt.Errorf("failed to get logs: %w", err)
	}
	formatter, fmtErr := getFormatter()
	if fmtErr != nil {
		return fmtErr
	}
	if formatter.Format() == output.FormatJSON || formatter.Format() == output.FormatYAML {
		return formatter.Print(resp.Logs)
	}
	for _, log := range resp.Logs {
		printLogEntry(log, showTimestamps)
	}
	return nil
}

func followLogs(cmd *cobra.Command, agentID, deploymentID string, limit int, showTimestamps bool) error {
	ctx := cmd.Context()
	// Follow mode: poll for new logs
	lastSeen := time.Time{}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var logs []sdk.LogEntry
			var err error
			logs, err = client.Logs.Tail(ctx, agentID, deploymentID, limit)
			if err != nil {
				return fmt.Errorf("log stream error: %w", err)
			}
			for _, log := range logs {
				if log.Timestamp.After(lastSeen) {
					printLogEntry(log, showTimestamps)
					lastSeen = log.Timestamp
				}
			}
			time.Sleep(2 * time.Second)
		}
	}
}

func printLogEntry(log sdk.LogEntry, showTimestamp bool) {
	var levelColor string
	switch strings.ToLower(log.Level) {
	case "error", "fatal":
		levelColor = output.ColorRed
	case "warn", "warning":
		levelColor = output.ColorYellow
	case "info":
		levelColor = output.ColorBlue
	case "debug":
		levelColor = output.ColorPurple
	default:
		levelColor = output.ColorWhite
	}
	level := strings.ToUpper(log.Level)
	if len(level) > 5 {
		level = level[:5]
	}
	level = fmt.Sprintf("%-5s", level)
	if showTimestamp {
		fmt.Printf("%s %s %s\n",
			output.Colorize(log.Timestamp.Format("2006-01-02T15:04:05"), output.ColorCyan),
			output.Colorize(level, levelColor),
			log.Message,
		)
	} else {
		fmt.Printf("%s %s\n", output.Colorize(level, levelColor), log.Message)
	}
}
