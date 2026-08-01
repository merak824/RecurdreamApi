package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/stretchr/testify/require"
)

func TestResponsesStreamEventSemanticOutputRejectsPreambleMetadataAndTerminalEvents(t *testing.T) {
	tests := []struct {
		name  string
		event apicompat.ResponsesStreamEvent
		want  bool
	}{
		{name: "created", event: apicompat.ResponsesStreamEvent{Type: "response.created"}},
		{name: "role only item", event: apicompat.ResponsesStreamEvent{Type: "response.output_item.added", Item: &apicompat.ResponsesOutput{Type: "message", Role: "assistant"}}},
		{name: "usage only", event: apicompat.ResponsesStreamEvent{Type: "response.completed", Usage: &apicompat.ResponsesUsage{InputTokens: 1}}},
		{name: "empty text delta", event: apicompat.ResponsesStreamEvent{Type: "response.output_text.delta"}},
		{name: "text delta", event: apicompat.ResponsesStreamEvent{Type: "response.output_text.delta", Delta: "hello"}, want: true},
		{name: "tool arguments delta", event: apicompat.ResponsesStreamEvent{Type: "response.function_call_arguments.delta", Arguments: "{"}, want: true},
		{name: "terminal", event: apicompat.ResponsesStreamEvent{Type: "response.done"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, responsesStreamEventHasSemanticOutput(tt.event))
		})
	}
}

func TestOpenAIStreamDataSemanticOutputRejectsRoleOnlyOutputItem(t *testing.T) {
	require.False(t, openAIStreamDataHasSemanticOutput(`{"type":"response.output_item.added","item":{"type":"message","role":"assistant","content":[]}}`, "response.output_item.added"))
	require.True(t, openAIStreamDataHasSemanticOutput(`{"type":"response.output_text.delta","delta":"hello"}`, "response.output_text.delta"))
	require.False(t, openAIStreamDataHasSemanticOutput(`{"type":"response.completed","response":{"usage":{"input_tokens":1}}}`, "response.completed"))
}
