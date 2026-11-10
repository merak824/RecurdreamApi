package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAIResponsesOrphanPrecheck(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{"empty object", `{}`, false},
		{"array root", `[{"input":[]}]`, false},
		{"string root", `"input"`, false},
		{"number root", `1`, false},
		{"null root", `null`, false},
		{"leading whitespace", "\r\n\t {\"input\":[{\"type\":\"message\"}]} ", false},
		{"empty array", `{"input":[]}`, false},
		{"message", `{"input":[{"type":"message","content":"plain text"}]}`, false},
		{"escaped content", `{"input":[{"content":"quoted \"type\":\"function_call_output\"\ntext","type":"message"}]}`, false},
		{"nested output type", `{"input":[{"type":"message","content":[{"type":"function_call_output"}]}]}`, false},
		{"scalar items", `{"input":[null,true,false,1,"escaped\ntext",[],{"type":"message"},null]}`, false},
		{"final scalar whitespace", "{\"input\":[{}, \t 12.5 \r\n ]}", false},
		{"previous response", `{"previous_response_id":" resp_1 ","input":[{"type":"function_call_output","call_id":"missing"}]}`, false},
		{"previous response unicode whitespace", `{"input":[{"type":"function_call_output"}],"previous_response_id":"　resp_1　"}`, false},
		{"blank previous response", `{"previous_response_id":"　 ","input":[{"type":"function_call_output"}]}`, true},
		{"non string previous response", `{"previous_response_id":123,"input":[{"type":"function_call_output"}]}`, true},
		{"null previous response", `{"previous_response_id":null,"input":[{"type":"function_call_output"}]}`, true},
		{"orphan output", `{"input":[{"type":"function_call_output","call_id":"missing"}]}`, true},
		{"trimmed output type", `{"input":[{"type":"　function_call_output　"}]}`, true},
		{"named standalone output", `{"input":[{"type":"function_call_output","name":"delegate","output":"task"}]}`, true},
		{"matched id fallback", `{"input":[{"type":"function_call","id":"fc_1"},{"type":"function_call_output","call_id":"fc_1"}]}`, true},
		{"matched reference", `{"input":[{"type":"item_reference","id":"ref_1"},{"type":"custom_tool_call_output","call_id":"ref_1"}]}`, true},
		{"search output", `{"input":[{"type":"tool_search_output"}]}`, true},
		{"mcp output", `{"input":[{"type":"mcp_tool_call_output"}]}`, true},
		{"non string type", `{"input":[{"type":123},{"type":null},{"type":{"type":"function_call_output"}}]}`, false},
		{"escaped input key", `{"inpu\u0074":[{"type":"function_call_output"}]}`, true},
		{"escaped type key", `{"input":[{"ty\u0070e":"function_call_output"}]}`, true},
		{"escaped output type", `{"input":[{"type":"function_call_\u006futput"}]}`, true},
		{"escaped previous key", `{"previous_response_\u0069d":"resp_1","input":[{"type":"function_call_output"}]}`, true},
		{"escaped previous value", `{"previous_response_id":"\u0020","input":[{"type":"function_call_output"}]}`, true},
		{"duplicate type last output", `{"input":[{"type":"message","type":"function_call_output"}]}`, true},
		{"duplicate type last message", `{"input":[{"type":"function_call_output","type":"message"}]}`, true},
		{"duplicate previous last null", `{"previous_response_id":"resp_1","previous_response_id":null,"input":[{"type":"function_call_output"}]}`, true},
		{"duplicate previous last present", `{"previous_response_id":null,"previous_response_id":"resp_1","input":[{"type":"function_call_output"}]}`, true},
		{"duplicate input last output", `{"input":[],"input":[{"type":"function_call_output"}]}`, true},
		{"duplicate input last null", `{"input":[{"type":"function_call_output"}],"input":null}`, true},
		{"first input non array keeps old gate", `{"input":null,"input":[{"type":"function_call_output"}]}`, false},
		{"empty body", ``, false},
		{"malformed without input", `{"model":`, false},
		{"malformed non array input", `{"input":"text",`, false},
		{"trailing non array input", `{"input":null} {}`, false},
		{"garbage prefix array input", `junk{"input":[]}`, true},
		{"garbage prefix non array input", `junk{"input":null}`, false},
		{"truncated", `{"input":[{"type":"message"}]`, true},
		{"trailing object", `{"input":[{"type":"message"}]} {}`, true},
		{"trailing junk", `{"input":[]}x`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte(tt.body)
			require.Equal(t, tt.want, openAIResponsesInputMayNeedOrphanCleanup(body))
			require.Equal(t, tt.body, string(body), "precheck must not mutate borrowed input")
		})
	}
}

func TestOpenAIResponsesOrphanPrecheckPreservesSanitizer(t *testing.T) {
	for _, body := range []string{
		`{"input":[{"type":"function_call_output","name":"delegate","namespace":"codex_app","output":"task"}]}`,
		`{"input":[{"type":"function_call_output","call_id":"fc_1","output":"ok"},{"type":"function_call","call_id":" ","id":"fc_1"}]}`,
		`{"input":[{"type":"tool_search_output","call_id":"ref_1"},{"type":"item_reference","id":"ref_1"}]}`,
		`{"input":[{"type":"message","type":"function_call_output","call_id":"missing"}]}`,
		`{"input":[{"type":"function_call_output","type":"message","content":"keep"}]}`,
		`{"previous_response_id":"resp_1","previous_response_id":null,"input":[{"type":"function_call_output"}]}`,
		`{"previous_response_id":null,"previous_response_id":"resp_1","input":[{"type":"function_call_output"}]}`,
		`{"input":[],"input":[{"type":"function_call_output"}]}`,
		`{"input":[{"type":"function_call_output"}],"input":null}`,
		`{"inpu\u0074":[{"ty\u0070e":"function_call_\u006futput"}],"vendor":{"count":9007199254740993}}`,
		`{"input":[{"type":"message","content":"unchanged"}],"vendor":{"count":9007199254740993}}`,
	} {
		t.Run(body, func(t *testing.T) {
			var expected, actual map[string]any
			require.NoError(t, decodeOpenAIJSONUseNumber([]byte(body), &expected))
			require.NoError(t, decodeOpenAIJSONUseNumber([]byte(body), &actual))
			apply := func(request map[string]any) bool {
				input, _ := request["input"].([]any)
				return sanitizeOpenAIResponsesOrphanToolOutputs(request, input, firstNonEmptyString(request["previous_response_id"]) != "")
			}
			wantChanged := apply(expected)
			gotChanged := false
			if openAIResponsesInputMayNeedOrphanCleanup([]byte(body)) {
				gotChanged = apply(actual)
			}
			require.Equal(t, wantChanged, gotChanged)
			require.Equal(t, expected, actual)

			account := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth}
			normalized, _, err := normalizeOpenAIResponsesWebSocketCompatibilityBody([]byte(body), account, false)
			require.NoError(t, err)
			var integrated map[string]any
			require.NoError(t, decodeOpenAIJSONUseNumber(normalized, &integrated))
			require.Equal(t, expected, integrated)
		})
	}
}

func TestOpenAIResponsesOrphanPrecheckCompatibilityErrors(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	for _, body := range []string{
		`{"input":[{"type":"message"}]`,
		`{"previous_response_id":"resp_1","input":[]} {}`,
		`{"input":[]} trailing`,
	} {
		t.Run(body, func(t *testing.T) {
			require.True(t, openAIResponsesInputMayNeedOrphanCleanup([]byte(body)))
			_, _, err := normalizeOpenAIResponsesWebSocketCompatibilityBody([]byte(body), account, false)
			require.Error(t, err)
		})
	}
}

// largeEscapedPrecheckBody 构造带大量转义字符的大 payload（约 256KB），
// 用于验证预检只做"借位扫描"而不反转义、不复制字符串。
func largeEscapedPrecheckBody() []byte {
	return []byte(`{"vendor":"` + strings.Repeat(`\n\"`, 1<<16) + `","input":[{"content":"` + strings.Repeat(`\n\"`, 1<<16) + `","type":"message"}]}`)
}

// TestOpenAIResponsesOrphanPrecheckLargeEscapedContent 只断言功能正确性。
//
// 零分配护栏放在 BenchmarkOpenAIResponsesOrphanPrecheckLargeEscapedContent 里：
// testing.AllocsPerRun 读的是进程级 Mallocs 计数，而本包里大量测试使用
// t.Parallel()，异步抢占发生时其他 goroutine 的分配会被计入本次测量，
// 在 CI 上表现为间歇性的 "Should be zero, but was N"（本机单独跑则为 0）。
// 因此把"不复制大字符串"的约束交给 benchmark（b.ReportAllocs，allocs/op 应为 0），
// 它不在 CI 门禁里跑，但可用 `go test -bench OrphanPrecheckLargeEscaped ./internal/service/`
// 随时回归。
func TestOpenAIResponsesOrphanPrecheckLargeEscapedContent(t *testing.T) {
	body := largeEscapedPrecheckBody()
	require.False(t, openAIResponsesInputMayNeedOrphanCleanup(body))

	// 重复调用仍然只返回 false，且大 payload 不会触发解码路径。
	for i := 0; i < 5; i++ {
		require.False(t, openAIResponsesInputMayNeedOrphanCleanup(body))
	}
}

// BenchmarkOpenAIResponsesOrphanPrecheckLargeEscapedContent 守护"借位扫描、零分配"
// 的实现约束：若有人把它改成反转义或复制字符串，allocs/op 会从 0 变成非 0。
func BenchmarkOpenAIResponsesOrphanPrecheckLargeEscapedContent(b *testing.B) {
	body := largeEscapedPrecheckBody()
	b.SetBytes(int64(len(body)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if openAIResponsesInputMayNeedOrphanCleanup(body) {
			b.Fatal("unexpected cleanup candidate")
		}
	}
}

func FuzzOpenAIResponsesOrphanPrecheck(f *testing.F) {
	for _, seed := range []string{
		`{"input":[{"type":"message","content":"text"}]}`,
		`{"input":[null,true,false,1,"text",[],{}]}`,
		`{"input":[{"type":"function_call_output"}]}`,
		`{"input":[{"type":"message","type":"function_call_output"}]}`,
		`{"previous_response_id":"resp_1","previous_response_id":null,"input":[]}`,
		`{"input":[{"ty\u0070e":"function_call_output"}]}`,
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		body := []byte(raw)
		mayNeedCleanup := openAIResponsesInputMayNeedOrphanCleanup(body)
		oldEligible := gjson.GetBytes(body, "input").IsArray()
		if !oldEligible {
			require.False(t, mayNeedCleanup, "must preserve the old non-array gate, including malformed input")
			return
		}
		var request map[string]any
		if err := decodeOpenAIJSONUseNumber(body, &request); err != nil {
			require.True(t, mayNeedCleanup, "must preserve the old decoder's errors")
			return
		}
		// Preserve the old initial first-key array gate, then compare with the
		// authoritative decoder/sanitizer's last-key behavior.
		input, _ := request["input"].([]any)
		if sanitizeOpenAIResponsesOrphanToolOutputs(request, input, firstNonEmptyString(request["previous_response_id"]) != "") && !mayNeedCleanup {
			t.Fatal("precheck missed required cleanup")
		}
	})
}
