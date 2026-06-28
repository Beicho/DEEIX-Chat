package conversation

import "testing"

func TestSplitThinkingContentOnlyAcceptsLeadingClosedBlock(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantVisible string
		wantThink   string
	}{
		{
			name:        "leading think block",
			input:       "<think>hidden</think>visible",
			wantVisible: "visible",
			wantThink:   "hidden",
		},
		{
			name:        "leading thinking block with attributes",
			input:       "\n<thinking data-source=\"model\">hidden</thinking>\nvisible",
			wantVisible: "visible",
			wantThink:   "hidden",
		},
		{
			name:        "middle think remains visible",
			input:       "visible <think>not hidden</think> tail",
			wantVisible: "visible <think>not hidden</think> tail",
			wantThink:   "",
		},
		{
			name:        "unclosed think remains visible",
			input:       "<think>not closed",
			wantVisible: "<think>not closed",
			wantThink:   "",
		},
		{
			name:        "plain thinking word remains visible",
			input:       "stream JSON uses isThinking to describe state",
			wantVisible: "stream JSON uses isThinking to describe state",
			wantThink:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visible, think := splitThinkingContent(tt.input)
			if visible != tt.wantVisible || think != tt.wantThink {
				t.Fatalf("unexpected split: visible=%q think=%q", visible, think)
			}
		})
	}
}

func TestSplitAssistantOutputThinkingContentRemovesProtocolUnsafeThinking(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantVisible string
		wantThink   string
	}{
		{
			name:        "closed leading think block",
			input:       "<think>hidden</think>visible",
			wantVisible: "visible",
			wantThink:   "hidden",
		},
		{
			name:        "unclosed leading think block",
			input:       "<thinking>tool decision",
			wantVisible: "",
			wantThink:   "tool decision",
		},
		{
			name:        "plain visible content",
			input:       "visible <think>literal</think>",
			wantVisible: "visible <think>literal</think>",
			wantThink:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visible, think := splitAssistantOutputThinkingContent(tt.input)
			if visible != tt.wantVisible || think != tt.wantThink {
				t.Fatalf("unexpected split: visible=%q think=%q", visible, think)
			}
		})
	}
}

func TestThinkingDeltaRouterParsesEachAssistantOutputStart(t *testing.T) {
	router := &thinkingDeltaRouter{}
	visible, think := router.consume("<thi")
	if visible != "" || think != "" {
		t.Fatalf("partial opening tag should be buffered, got visible=%q think=%q", visible, think)
	}
	visible, think = router.consume("nk>hid")
	if visible != "" || think != "hid" {
		t.Fatalf("completed opening tag should enter thinking immediately: visible=%q think=%q", visible, think)
	}
	visible, think = router.consume("den</think>visible")
	if visible != "visible" || think != "den" {
		t.Fatalf("unexpected completed block tail split: visible=%q think=%q", visible, think)
	}
	visible, think = router.consume(" with <think>literal</think>")
	if visible != " with <think>literal</think>" || think != "" {
		t.Fatalf("post-resolution tags should stay visible: visible=%q think=%q", visible, think)
	}

	nextRouter := &thinkingDeltaRouter{}
	visible, think = nextRouter.consume("<thinking>sec")
	if visible != "" || think != "sec" {
		t.Fatalf("new assistant output should enter thinking immediately: visible=%q think=%q", visible, think)
	}
	visible, think = nextRouter.consume("ond</thinking>answer")
	if visible != "answer" || think != "ond" {
		t.Fatalf("new assistant output should parse its own leading block: visible=%q think=%q", visible, think)
	}
}

func TestThinkingDeltaRouterHoldsPartialClosingTag(t *testing.T) {
	router := &thinkingDeltaRouter{}
	visible, think := router.consume("<think>hidden</thi")
	if visible != "" || think != "hidden" {
		t.Fatalf("partial closing tag should be buffered outside thinking text: visible=%q think=%q", visible, think)
	}
	visible, think = router.consume("nk>visible")
	if visible != "visible" || think != "" {
		t.Fatalf("completed closing tag should exit thinking: visible=%q think=%q", visible, think)
	}
}

func TestThinkingDeltaRouterKeepsInvalidLeadingTagVisible(t *testing.T) {
	router := &thinkingDeltaRouter{}
	visible, think := router.consume("<thinkingg")
	if visible != "<thinkingg" || think != "" {
		t.Fatalf("invalid leading tag should stay visible: visible=%q think=%q", visible, think)
	}
}

func TestThinkingDeltaRouterFlushesUnclosedBlockAsThinking(t *testing.T) {
	router := &thinkingDeltaRouter{}
	visible, think := router.consume("<thinking>not closed")
	if visible != "" || think != "not closed" {
		t.Fatalf("leading block should stream as thinking after opening tag, got visible=%q think=%q", visible, think)
	}
	visible, think = router.flush()
	if visible != "" || think != "" {
		t.Fatalf("flushed unclosed block should not duplicate streamed thinking, got visible=%q think=%q", visible, think)
	}
}

func TestSanitizeAssistantProtocolContentRemovesToolCallBlocks(t *testing.T) {
	input := "Let me check.\n<tool_call>\n<tool_name>find_file</tool_name>\n</tool_call>\nFinal answer."
	got := sanitizeAssistantProtocolContent(input)
	if got != "Let me check.\n\nFinal answer." {
		t.Fatalf("unexpected sanitized content: %q", got)
	}
}

func TestSanitizeAssistantProtocolContentRemovesRawToolCallJSON(t *testing.T) {
	input := `{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"web_search","arguments":"{}"}}]}`
	if got := sanitizeAssistantProtocolContent(input); got != "" {
		t.Fatalf("expected raw tool call JSON to be removed, got %q", got)
	}
}

func TestSanitizeAssistantProtocolContentRemovesMiddleThinkingBlock(t *testing.T) {
	input := "Visible <thinking>hidden</thinking> answer"
	if got := sanitizeAssistantProtocolContent(input); got != "Visible  answer" {
		t.Fatalf("unexpected sanitized content: %q", got)
	}
}

func TestAssistantProtocolDeltaRouterHidesChunkedToolCallBlock(t *testing.T) {
	router := &assistantProtocolDeltaRouter{}
	var got string
	got += router.consume("Visible <tool")
	got += router.consume("_call><tool_name>find")
	got += router.consume("_file</tool_name></tool_call> done")
	got += router.flush()
	if got != "Visible  done" {
		t.Fatalf("unexpected visible stream: %q", got)
	}
}
