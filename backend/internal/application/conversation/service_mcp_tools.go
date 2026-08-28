package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	domainmcp "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/mcp"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/pkg/secretbox"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/ports/llm"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/ports/mcp"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/security"
)

type selectedToolRuntime struct {
	definitions         []llm.ToolDefinition
	nameMap             map[string]string
	mcpBindings         map[string]mcpToolCallBinding
	builtIn             map[string]string
	schemas             map[string]json.RawMessage
	attachmentProcessor *selectedAttachmentProcessor
}

// mcpToolCallBinding 绑定模型侧工具名对应的 MCP 调用配置与计量元数据。
// 服务器归属与价格在解析选中工具时快照，保证同名工具跨服务器可区分、计费按调用时价格结算。
type mcpToolCallBinding struct {
	Config       mcp.CallConfig
	ServerID     uint
	ServerName   string
	ToolName     string
	PriceNanousd int64
}

type selectedAttachmentProcessor struct {
	toolID         uint
	modelName      string
	toolName       string
	displayName    string
	argument       string
	encoding       string
	promptArgument string
}

func injectMCPToolGuidance(messages []llm.Message, runtime selectedToolRuntime, customPrompt string) []llm.Message {
	if len(runtime.definitions) == 0 {
		return messages
	}

	content := strings.TrimSpace(customPrompt)
	if content == "" {
		content = defaultMCPToolGuidancePrompt()
	}

	insertAt := 0
	for insertAt < len(messages) && messages[insertAt].Role == "system" {
		insertAt++
	}
	next := make([]llm.Message, 0, len(messages)+1)
	next = append(next, messages[:insertAt]...)
	next = append(next, llm.Message{Role: "system", Content: content})
	next = append(next, messages[insertAt:]...)
	return next
}

func defaultMCPToolGuidancePrompt() string {
	var builder strings.Builder
	builder.WriteString("# tool_use\n")
	builder.WriteString("- Tools are declared separately via the API schema; follow that schema exactly.\n")
	builder.WriteString("- Use tools only for external, realtime, private, or explicitly requested data.\n")
	builder.WriteString("- Use the fewest useful calls; each call must add new information.\n")
	builder.WriteString("- Do not repeat an identical failed call. Adjust arguments, use another tool, or answer from available evidence.\n")
	builder.WriteString("- If tools fail or lack enough data, state the gap in the final answer.\n")
	builder.WriteString("- Do not expose raw tool JSON, internal fields, or tool logs unless the user asks.\n")
	return strings.TrimSpace(builder.String())
}

func summarizeToolInputSchema(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(raw, &schema); err != nil {
		return ""
	}
	properties, _ := schema["properties"].(map[string]interface{})
	if len(properties) == 0 {
		return "无需参数"
	}
	required := map[string]struct{}{}
	if items, ok := schema["required"].([]interface{}); ok {
		for _, item := range items {
			if name, ok := item.(string); ok && strings.TrimSpace(name) != "" {
				required[strings.TrimSpace(name)] = struct{}{}
			}
		}
	}
	names := make([]string, 0, len(properties))
	for name := range properties {
		if strings.TrimSpace(name) != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, name := range names {
		prop, _ := properties[name].(map[string]interface{})
		fieldType := schemaFieldType(prop)
		label := name
		if fieldType != "" {
			label = fmt.Sprintf("%s:%s", name, fieldType)
		}
		if _, ok := required[name]; ok {
			label += " 必填"
		}
		parts = append(parts, label)
	}
	if len(parts) > 6 {
		parts = append(parts[:6], fmt.Sprintf("等 %d 个字段", len(parts)))
	}
	return "参数 " + strings.Join(parts, "，")
}

func schemaFieldType(prop map[string]interface{}) string {
	if len(prop) == 0 {
		return ""
	}
	if value, ok := prop["type"].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	if items, ok := prop["type"].([]interface{}); ok && len(items) > 0 {
		types := make([]string, 0, len(items))
		for _, item := range items {
			if value, ok := item.(string); ok && strings.TrimSpace(value) != "" {
				types = append(types, strings.TrimSpace(value))
			}
		}
		if len(types) > 0 {
			return strings.Join(types, "|")
		}
	}
	if _, ok := prop["enum"].([]interface{}); ok {
		return "enum"
	}
	return ""
}

func (s *Service) resolveSelectedToolRuntime(
	ctx context.Context,
	userID uint,
	toolIDs []uint,
	confirmedToolIDs []uint,
	webSearchEnabled bool,
	codeSandboxEnabled bool,
) (selectedToolRuntime, error) {
	cfg := s.cfg.Snapshot()
	result := selectedToolRuntime{
		definitions: make([]llm.ToolDefinition, 0, len(toolIDs)+2),
		nameMap:     map[string]string{},
		mcpBindings: map[string]mcpToolCallBinding{},
		builtIn:     map[string]string{},
		schemas:     map[string]json.RawMessage{},
	}
	usedNames := map[string]int{}
	if webSearchEnabled {
		s.addBuiltInToolDefinition(&result, usedNames, "web_search")
	}
	if codeSandboxEnabled && cfg.CodeSandboxEnabled {
		s.addBuiltInToolDefinition(&result, usedNames, "run_python")
	}
	if s.mcpRepo == nil || len(toolIDs) == 0 || !cfg.MCPEnable {
		if len(result.definitions) > 0 {
			return result, nil
		}
		if len(toolIDs) > 0 && s.mcpRepo == nil {
			return selectedToolRuntime{}, fmt.Errorf("resolve selected MCP tools: repository unavailable")
		}
		return selectedToolRuntime{}, nil
	}
	tools, err := s.mcpRepo.ListToolsByIDsForUser(ctx, uniqueToolIDs(toolIDs), userID)
	if err != nil {
		return selectedToolRuntime{}, fmt.Errorf("resolve selected MCP tools: %w", err)
	}
	if len(tools) == 0 {
		if len(result.definitions) > 0 {
			return result, nil
		}
		return selectedToolRuntime{}, nil
	}

	confirmedSet := uintSet(confirmedToolIDs)
	serverCache := map[uint]*domainmcp.Server{}
	for _, tool := range tools {
		if tool.Status != "active" {
			continue
		}
		if tool.RequiresConfirm {
			if _, ok := confirmedSet[tool.ID]; !ok {
				continue
			}
		}
		isAttachmentProcessor := strings.EqualFold(strings.TrimSpace(tool.AttachmentInputMode), domainmcp.AttachmentInputModeImage)
		server, ok := serverCache[tool.ServerID]
		if !ok {
			server, err = s.mcpRepo.GetServer(ctx, tool.ServerID)
			if err != nil {
				return selectedToolRuntime{}, fmt.Errorf("resolve MCP server %d: %w", tool.ServerID, err)
			}
			if server == nil || server.Status != "active" {
				if isAttachmentProcessor {
					return selectedToolRuntime{}, fmt.Errorf("%w: processor server is unavailable", ErrImageAttachmentProcessingFailed)
				}
				continue
			}
			if validateErr := security.ValidateTrustedOutboundHTTPURL(server.BaseURL); validateErr != nil {
				if isAttachmentProcessor {
					return selectedToolRuntime{}, fmt.Errorf("%w: processor server URL is not allowed", ErrImageAttachmentProcessingFailed)
				}
				continue
			}
			serverCache[tool.ServerID] = server
		}
		modelName := uniqueModelToolName(llm.NormalizeToolName(tool.Name), usedNames)
		if modelName == "" {
			continue
		}
		schema := json.RawMessage(strings.TrimSpace(tool.InputSchemaJSON))
		if len(schema) == 0 {
			schema = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		token, err := secretbox.DecryptString(cfg.DataEncryptionKey, server.AuthTokenEnc)
		if err != nil {
			if isAttachmentProcessor {
				return selectedToolRuntime{}, fmt.Errorf("%w: processor credentials are unavailable", ErrImageAttachmentProcessingFailed)
			}
			continue
		}
		headers := parseMCPHeaders(server.HeadersJSON)
		result.definitions = append(result.definitions, llm.ToolDefinition{
			Name:        modelName,
			Description: strings.TrimSpace(tool.Description),
			InputSchema: schema,
		})
		result.nameMap[modelName] = tool.Name
		result.schemas[modelName] = schema
		result.mcpBindings[modelName] = mcpToolCallBinding{
			Config: mcp.CallConfig{
				BaseURL:   server.BaseURL,
				AuthToken: token,
				TimeoutMS: resolveMCPToolTimeoutMS(server.TimeoutSeconds, cfg.MCPToolTimeoutSeconds),
				Headers:   headers,
			},
			ServerID:     server.ID,
			ServerName:   server.Name,
			ToolName:     tool.Name,
			PriceNanousd: tool.PriceNanousd,
		}
		if isAttachmentProcessor {
			if bindErr := result.bindAttachmentProcessor(selectedAttachmentProcessor{
				toolID:         tool.ID,
				modelName:      modelName,
				toolName:       tool.Name,
				displayName:    firstNonEmptyString(tool.DisplayName, tool.Name),
				argument:       strings.TrimSpace(tool.AttachmentArgument),
				encoding:       strings.TrimSpace(tool.AttachmentEncoding),
				promptArgument: strings.TrimSpace(tool.AttachmentPromptArgument),
			}); bindErr != nil {
				return selectedToolRuntime{}, bindErr
			}
		}
	}
	return result, nil
}

func (r *selectedToolRuntime) bindAttachmentProcessor(processor selectedAttachmentProcessor) error {
	if r.attachmentProcessor != nil {
		return ErrMultipleImageAttachmentProcessors
	}
	r.attachmentProcessor = &processor
	return nil
}

func (r selectedToolRuntime) withoutAttachmentProcessor() selectedToolRuntime {
	processor := r.attachmentProcessor
	if processor == nil {
		return r
	}
	definitions := make([]llm.ToolDefinition, 0, len(r.definitions))
	for _, definition := range r.definitions {
		if definition.Name != processor.modelName {
			definitions = append(definitions, definition)
		}
	}
	r.definitions = definitions
	delete(r.nameMap, processor.modelName)
	delete(r.mcpBindings, processor.modelName)
	delete(r.schemas, processor.modelName)
	r.attachmentProcessor = nil
	return r
}

func (r selectedToolRuntime) withoutDefinitions() selectedToolRuntime {
	r.definitions = nil
	r.nameMap = nil
	r.mcpBindings = nil
	r.builtIn = nil
	r.schemas = nil
	r.attachmentProcessor = nil
	return r
}

func (s *Service) addBuiltInToolDefinition(runtime *selectedToolRuntime, usedNames map[string]int, kind string) {
	if runtime == nil {
		return
	}
	var name string
	var description string
	var schema json.RawMessage
	switch kind {
	case "web_search":
		if !webSearchAvailable(s.cfg.Snapshot()) {
			return
		}
		name = uniqueModelToolName("web_search", usedNames)
		description = "Search the web for current public information. Return concise results with source URLs."
		schema = json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"Search query"},"max_results":{"type":"integer","minimum":1,"maximum":10}},"required":["query"]}`)
	case "run_python":
		name = uniqueModelToolName("run_python", usedNames)
		description = "Run a short Python 3 snippet in an isolated, time-limited process and return stdout, stderr, and exit code."
		schema = json.RawMessage(`{"type":"object","properties":{"code":{"type":"string","description":"Python code to run"}},"required":["code"]}`)
	default:
		return
	}
	if name == "" {
		return
	}
	runtime.definitions = append(runtime.definitions, llm.ToolDefinition{
		Name:        name,
		Description: description,
		InputSchema: schema,
	})
	runtime.nameMap[name] = kind
	runtime.schemas[name] = schema
	runtime.builtIn[name] = kind
}

func uniqueToolIDs(items []uint) []uint {
	seen := make(map[uint]struct{}, len(items))
	result := make([]uint, 0, len(items))
	for _, item := range items {
		if item == 0 {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func uintSet(items []uint) map[uint]struct{} {
	result := make(map[uint]struct{}, len(items))
	for _, item := range items {
		if item != 0 {
			result[item] = struct{}{}
		}
	}
	return result
}

func resolveMCPToolTimeoutMS(serverSeconds int, fallbackSeconds int) int {
	if serverSeconds > 0 {
		return serverSeconds * 1000
	}
	if fallbackSeconds > 0 {
		return fallbackSeconds * 1000
	}
	return 60000
}

func uniqueModelToolName(base string, used map[string]int) string {
	value := strings.TrimSpace(base)
	if value == "" {
		return ""
	}
	count := used[value]
	used[value] = count + 1
	if count == 0 {
		return value
	}
	suffix := "_" + strconv.Itoa(count+1)
	if len(value)+len(suffix) > 64 {
		value = value[:64-len(suffix)]
	}
	return value + suffix
}

func parseMCPHeaders(raw string) map[string]string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return map[string]string{}
	}
	payload := map[string]string{}
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return map[string]string{}
	}
	result := make(map[string]string, len(payload))
	for key, item := range payload {
		headerKey := strings.TrimSpace(key)
		if headerKey == "" {
			continue
		}
		result[headerKey] = strings.TrimSpace(item)
	}
	return result
}
