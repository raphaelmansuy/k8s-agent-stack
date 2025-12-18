package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/raphaelmansuy/agentstack/cli/internal/output"
	"github.com/raphaelmansuy/agentstack/sdk"
	"github.com/spf13/cobra"
)

func newChatCmd() *cobra.Command {
	var sessionID string
	var stream bool
	cmd := &cobra.Command{
		Use:   "chat <agent-id>",
		Short: "Start an interactive chat session with an agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := args[0]
			ctx := cmd.Context()
			fmt.Printf("Starting chat with agent %s...\n", agentID)
			fmt.Println("Type /quit to exit, /help for commands.")
			fmt.Println()
			reader := bufio.NewReader(os.Stdin)
			var history []sdk.ChatMessage
			for {
				fmt.Print(output.Colorize("You: ", output.ColorCyan))
				input, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("failed to read input: %w", err)
				}
				input = strings.TrimSpace(input)
				if input == "" {
					continue
				}
				if strings.HasPrefix(input, "/") {
					switch input {
					case "/quit", "/exit":
						fmt.Println("Goodbye!")
						return nil
					case "/clear":
						history = nil
						sessionID = ""
						fmt.Println("Conversation cleared.")
						continue
					case "/history":
						printHistory(history)
						continue
					case "/help":
						printChatHelp()
						continue
					default:
						fmt.Println("Unknown command. Type /help for help.")
						continue
					}
				}
				history = append(history, sdk.ChatMessage{Role: "user", Content: input})
				fmt.Print(output.Colorize("Agent: ", output.ColorGreen))
				if stream {
					response, newSessionID, err := streamChat(ctx, client, agentID, history, sessionID)
					if err != nil {
						fmt.Println(output.Error(err.Error()))
						continue
					}
					fmt.Println()
					sessionID = newSessionID
					history = append(history, sdk.ChatMessage{Role: "assistant", Content: response})
				} else {
					response, err := sendChat(ctx, client, agentID, history, sessionID)
					if err != nil {
						fmt.Println(output.Error(err.Error()))
						continue
					}
					fmt.Println(response.Message.Content)
					if response.SessionID != "" {
						sessionID = response.SessionID
					}
					history = append(history, sdk.ChatMessage{Role: "assistant", Content: response.Message.Content})
				}
				fmt.Println()
			}
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "continue a previous session")
	cmd.Flags().BoolVar(&stream, "stream", true, "enable streaming responses")
	return cmd
}

func sendChat(ctx context.Context, c *sdk.Client, agentID string, messages []sdk.ChatMessage, sessionID string) (*sdk.ChatResponse, error) {
	req := &sdk.ChatRequest{
		AgentID:   agentID,
		SessionID: sessionID,
		Messages:  messages,
		Stream:    false,
	}
	return c.Chat.Send(ctx, req)
}

func streamChat(ctx context.Context, c *sdk.Client, agentID string, messages []sdk.ChatMessage, sessionID string) (string, string, error) {
	req := &sdk.ChatRequest{
		AgentID:   agentID,
		SessionID: sessionID,
		Messages:  messages,
		Stream:    true,
	}
	streamResp, err := c.Chat.Stream(ctx, req)
	if err != nil {
		return "", sessionID, err
	}
	var fullContent strings.Builder
	var newSessionID = sessionID
	for {
		select {
		case event, ok := <-streamResp.Events:
			if !ok {
				return fullContent.String(), newSessionID, nil
			}
			if event.Content != "" {
				fmt.Print(event.Content)
				fullContent.WriteString(event.Content)
			}
			if event.SessionID != "" {
				newSessionID = event.SessionID
			}
			if event.Done {
				return fullContent.String(), newSessionID, nil
			}
		case err := <-streamResp.Errors:
			if err != nil {
				return fullContent.String(), newSessionID, err
			}
		case <-streamResp.Done:
			return fullContent.String(), newSessionID, nil
		case <-ctx.Done():
			return fullContent.String(), newSessionID, ctx.Err()
		}
	}
}

func printHistory(history []sdk.ChatMessage) {
	if len(history) == 0 {
		fmt.Println("No conversation history.")
		return
	}
	fmt.Println("\n--- Conversation History ---")
	for _, msg := range history {
		var color, label string
		if msg.Role == "user" {
			color = output.ColorCyan
			label = "You"
		} else {
			color = output.ColorGreen
			label = "Agent"
		}
		fmt.Printf("%s: %s\n", output.Colorize(label, color), msg.Content)
	}
	fmt.Println("--- End of History ---")
	fmt.Println()
}

func printChatHelp() {
	fmt.Println(`Available commands:
  /quit, /exit  - Exit the chat session
  /clear        - Clear conversation history
  /history      - Show conversation history
  /help         - Show this help message`)
}
