package conversation

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/channel"
	appupload "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/upload"
	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/llm"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/pkg/traceid"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/security"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/net/proxy"
)

// MediaImageTaskType 表示媒体图片任务类型。
type MediaImageTaskType string

const (
	// MediaImageTaskGeneration 表示纯文本提示词生成图片任务。
	MediaImageTaskGeneration MediaImageTaskType = "image_generation"
	// MediaImageTaskEdit 表示基于输入图片的编辑任务。
	MediaImageTaskEdit MediaImageTaskType = "image_edit"
)

const maxMediaImageEditInputImages = 16

const maxMediaVideoInputImages = 8

const (
	openAIVideoCreatePath = "/videos"
	openAIVideoPollPath   = "/videos/"
	legacyVideoCreatePath = "/videos/generations"
	legacyVideoPollPath   = "/videos/generations/"
	taskVideoCreatePath   = "/video/generations"
	taskVideoPollPath     = "/video/generations/"
	seedanceCreatePath    = "/contents/generations/tasks"
	seedancePollPath      = "/contents/generations/tasks/"
	videoTaskPollAttempts = 120
	videoTaskPollInterval = 5 * time.Second
	videoCreateAttempts   = 3
	videoCreateRetryGap   = 2 * time.Second

	generatedVideoDownloadAttempts = 3
	generatedVideoDownloadRetryGap = 2 * time.Second

	defaultVolcengineArkBaseURL = "https://ark.cn-beijing.volces.com/api/v3"
)

type mediaImageCapabilities struct {
	Image struct {
		Stream *bool `json:"stream"`
	} `json:"image"`
}

// MediaImageInput 定义媒体图片任务的应用层入参。
type MediaImageInput struct {
	UserID                uint
	ConversationID        uint
	RequestID             string
	TaskType              MediaImageTaskType
	Prompt                string
	PlatformModelName     string
	Options               map[string]interface{}
	ClientRunID           string
	FileIDs               []string
	MaskFileID            string
	ParentMessagePublicID string
	SourceMessagePublicID string
	BranchReason          string
	OnEvent               func(eventType string, payload map[string]interface{}) error
}

// StreamMediaImage 执行图片生成任务并把结果保存为文件对象。
// 图片能力不复用聊天生成链路，只通过图片任务类型和图片协议路由。
func (s *Service) StreamMediaImage(ctx context.Context, input MediaImageInput) (*SendMessageResult, error) {
	if input.TaskType != MediaImageTaskGeneration && input.TaskType != MediaImageTaskEdit {
		return nil, ErrInvalidMediaGenerationTask
	}
	if s.routeResolver == nil || s.llmClient == nil {
		return nil, ErrModelRouteNotConfigured
	}
	ctx = context.WithoutCancel(ctx)

	// clientRunID 是媒体任务的幂等键；重复提交不能继续创建 run 和消息。
	runID := normalizeRunID(input.ClientRunID)
	if runID == "" {
		runID = "run_" + normalizePublicID(uuid.NewString())
	}
	existingRuns, err := s.repo.ListConversationRunsByRunIDs(ctx, input.UserID, input.ConversationID, []string{runID})
	if err != nil {
		return nil, err
	}
	if len(existingRuns) > 0 {
		return nil, ErrDuplicateMessageGenerationRun
	}
	startedAt := time.Now()
	conversation, err := s.repo.GetConversationByUser(ctx, input.ConversationID, input.UserID)
	if err != nil {
		return nil, ErrConversationNotFound
	}

	normalizedBranchReason := normalizeBranchReason(input.BranchReason)
	branchState, err := s.resolveMessageBranch(ctx, input.ConversationID, input.UserID, input.ParentMessagePublicID, input.SourceMessagePublicID, normalizedBranchReason)
	if err != nil {
		return nil, err
	}
	reuseUserMessage := branchState.ReuseUserMessage != nil
	if reuseUserMessage {
		input.Prompt = branchState.ReuseUserMessage.Content
		input.FileIDs = parseAttachmentSnapshotFileIDs(branchState.ReuseUserMessage.Attachments)
	}
	if strings.TrimSpace(input.Prompt) == "" {
		return nil, ErrMediaImagePromptRequired
	}
	if input.TaskType == MediaImageTaskGeneration && len(input.FileIDs) > 0 {
		return nil, ErrMediaImageGenerationRejectsInputs
	}
	if input.TaskType == MediaImageTaskEdit && len(input.FileIDs) == 0 {
		return nil, ErrMediaImageEditInputRequired
	}

	platformModelName := strings.TrimSpace(input.PlatformModelName)
	if platformModelName == "" {
		platformModelName = strings.TrimSpace(conversation.Model)
	}
	if platformModelName == "" {
		return nil, ErrModelRouteNotConfigured
	}
	taskRouteType := channel.TaskTypeImageGeneration
	endpoint := llm.EndpointImageGenerations
	if input.TaskType == MediaImageTaskEdit {
		taskRouteType = channel.TaskTypeImageEdit
		endpoint = llm.EndpointImageEdits
	}
	route, err := s.routeResolver.ResolveRoute(ctx, channel.ResolveRouteInput{
		PlatformModelName: platformModelName,
		TaskType:          taskRouteType,
		Scope:             channel.RouteScopeUser,
		UserID:            input.UserID,
		ConversationID:    input.ConversationID,
		RequestID:         strings.TrimSpace(input.RequestID),
	})
	if err != nil {
		return nil, ErrModelRouteNotConfigured
	}
	if input.TaskType == MediaImageTaskGeneration && !llm.IsImageGenerationAdapter(route.Protocol) {
		return nil, ErrMediaRouteProtocolMismatch
	}
	if input.TaskType == MediaImageTaskEdit && !llm.IsImageEditAdapter(route.Protocol) {
		return nil, ErrMediaRouteProtocolMismatch
	}
	// 图片任务会把会话当前模型更新为实际执行的图片模型；标题、标签等内部文本任务会单独回退到聊天模型。
	if strings.TrimSpace(conversation.Model) != strings.TrimSpace(route.PlatformModelName) {
		conversation.Model = strings.TrimSpace(route.PlatformModelName)
		conversation.Provider = inferProvider(conversation.Model)
		if err = s.repo.UpdateConversationModel(ctx, input.ConversationID, conversation.Model, conversation.Provider); err != nil {
			return nil, err
		}
	}
	resolvedAttachments, imageEditParts, err := s.resolveMediaImageEditInputs(ctx, input)
	if err != nil {
		return nil, err
	}
	maskPart, err := s.resolveMediaImageEditMask(ctx, input.UserID, input.MaskFileID)
	if err != nil {
		return nil, err
	}
	run := &model.Run{
		RunID:              runID,
		RequestID:          strings.TrimSpace(input.RequestID),
		UserID:             input.UserID,
		ConversationID:     input.ConversationID,
		TaskType:           string(input.TaskType),
		Endpoint:           endpoint,
		Provider:           strings.TrimSpace(conversation.Provider),
		ProviderProtocol:   route.Protocol,
		UpstreamID:         route.UpstreamID,
		UpstreamModelID:    route.UpstreamModelID,
		UpstreamName:       route.UpstreamName,
		RequestedModelName: platformModelName,
		PlatformModelName:  route.PlatformModelName,
		RoutedBindingCode:  route.BindingCode,
		ModelVendor:        route.ModelVendor,
		ModelIcon:          route.ModelIcon,
		UpstreamModelName:  route.UpstreamModel,
		Status:             "error",
		StartedAt:          startedAt,
	}
	var retErr error
	defer func() {
		endedAt := time.Now()
		run.EndedAt = &endedAt
		run.TotalLatencyMS = endedAt.Sub(startedAt).Milliseconds()
		if retErr == nil {
			run.Status = "success"
		} else {
			run.Status = "error"
			run.ErrorCode = classifyRunErrorCode(retErr)
			run.ErrorMessage = truncateError(messageErrorSummary(retErr), 255)
		}
		if err := s.repo.CreateConversationRun(context.WithoutCancel(ctx), run); err != nil && s.logger != nil {
			s.logger.Error("create_media_conversation_run_failed",
				zap.String("trace_id", traceid.FromContext(ctx)),
				zap.String("run_id", run.RunID),
				zap.Error(err),
			)
		}
	}()
	attachmentsJSON := marshalAttachmentSnapshots(resolvedAttachments)
	cancelCtx, cancel := context.WithCancel(ctx)
	ctx = cancelCtx
	s.generationStreams.register(ctx, runID, input.UserID, cancel)

	assistantMessage := &model.Message{
		ConversationID: input.ConversationID,
		UserID:         input.UserID,
		PublicID:       normalizePublicID(uuid.NewString()),
		RunID:          runID,
		Role:           "assistant",
		ContentType:    "image",
		Content:        "",
		BranchReason:   normalizedBranchReason,
		Status:         "pending",
		Attachments:    "[]",
	}
	var userMessage *model.Message
	if reuseUserMessage {
		reused := *branchState.ReuseUserMessage
		userMessage = &reused
		assistantMessage.ParentMessageID = &userMessage.ID
		assistantMessage.SourceMessageID = branchState.SourceMessageID
		if err = s.repo.CreateAssistantBranchMessage(ctx, assistantMessage); err != nil {
			retErr = err
			return nil, err
		}
		assistantMessage.ParentPublicID = userMessage.PublicID
		assistantMessage.SourcePublicID = branchState.SourcePublicID
	} else {
		userMessage = &model.Message{
			ConversationID:  input.ConversationID,
			UserID:          input.UserID,
			PublicID:        normalizePublicID(uuid.NewString()),
			ParentMessageID: branchState.ParentMessageID,
			RunID:           runID,
			Role:            "user",
			ContentType:     mediaImageUserContentType(input.TaskType),
			Content:         strings.TrimSpace(input.Prompt),
			BranchReason:    normalizedBranchReason,
			SourceMessageID: branchState.SourceMessageID,
			TokenUsage:      estimateTokens(input.Prompt),
			InputTokens:     estimateTokens(input.Prompt),
			Status:          "success",
			Attachments:     attachmentsJSON,
		}
		userAttachmentRows := make([]model.Attachment, 0, len(resolvedAttachments))
		if len(resolvedAttachments) > 0 {
			now := time.Now()
			for _, item := range resolvedAttachments {
				userAttachmentRows = append(userAttachmentRows, model.Attachment{
					ConversationID: input.ConversationID,
					UserID:         input.UserID,
					FileID:         strings.TrimSpace(item.FileID),
					Kind:           normalizeAttachmentKind(item.Kind, item.MimeType),
					FileName:       strings.TrimSpace(item.FileName),
					MimeType:       strings.TrimSpace(item.MimeType),
					FileSize:       item.FileSize,
					SHA256:         strings.TrimSpace(item.SHA256),
					StoragePath:    strings.TrimSpace(item.StoragePath),
					Status:         "active",
					MetaJSON:       strings.TrimSpace(item.MetaJSON),
					UploadedAt:     now,
				})
			}
		}

		// 媒体任务同样产生一个完整消息回合，初始本地写入必须原子提交。
		if err = s.repo.CreateMessagePairWithUserAttachments(ctx, userMessage, assistantMessage, userAttachmentRows); err != nil {
			retErr = err
			return nil, err
		}
		userMessage.ParentPublicID = branchState.ParentPublicID
		userMessage.SourcePublicID = branchState.SourcePublicID
		assistantMessage.ParentPublicID = userMessage.PublicID
		s.maybeGenerateConversationMetadataAsync(*conversation, *userMessage)
	}
	traceRecorder := newMessageTraceRecorder(s, ctx, assistantMessage, input.OnEvent)
	defer func() {
		if retErr != nil && traceRecorder != nil {
			traceRecorder.fail(retErr)
			traceRecorder.attachToMessage(assistantMessage)
		}
	}()
	emitMediaEvent(input.OnEvent, "queued", "image task queued")

	cfg := s.cfg.Snapshot()
	attributionReferer, attributionTitle := s.llmAttribution()
	routeConfig := llm.RouteConfig{
		Protocol:            route.Protocol,
		BaseURL:             route.BaseURL,
		APIKey:              route.APIKey,
		HeadersJSON:         route.HeadersJSON,
		ConnectTimeoutMS:    route.ConnectTimeoutMS,
		ReadTimeoutMS:       route.ReadTimeoutMS,
		StreamIdleTimeoutMS: route.StreamIdleTimeoutMS,
		Endpoint:            endpoint,
		UpstreamModel:       route.UpstreamModel,
		AttributionReferer:  attributionReferer,
		AttributionTitle:    attributionTitle,
	}
	filteredOptions := filterModelOptions(input.Options, route.Protocol, modelOptionPolicyConfig{
		Mode:                  cfg.ModelOptionPolicyMode,
		AllowedPathsJSON:      cfg.ModelOptionAllowedPaths,
		DeniedPathsJSON:       cfg.ModelOptionDeniedPaths,
		ModelCapabilitiesJSON: route.ModelCapabilitiesJSON,
	})

	emitMediaEvent(input.OnEvent, "running", mediaImageRunningMessage(input.TaskType))
	generateInput := llm.GenerateInput{
		RequestID:      strings.TrimSpace(input.RequestID),
		ConversationID: input.ConversationID,
		Messages: []llm.Message{{
			Role:    "user",
			Content: strings.TrimSpace(input.Prompt),
		}},
		Options: filteredOptions,
	}
	if input.TaskType == MediaImageTaskEdit {
		parts := make([]llm.ContentPart, 0, 1+len(imageEditParts))
		parts = append(parts, llm.ContentPart{
			Kind: llm.ContentPartText,
			Text: strings.TrimSpace(input.Prompt),
		})
		parts = append(parts, imageEditParts...)
		generateInput.Messages = []llm.Message{{
			Role:  "user",
			Parts: parts,
		}}
		generateInput.ImageEditMask = maskPart
	}
	var output *llm.GenerateOutput
	if mediaImageStreamEnabled(routeConfig.Protocol, routeConfig.UpstreamModel, route.ModelCapabilitiesJSON) {
		output, err = s.llmClient.GenerateStream(ctx, routeConfig, generateInput, func(event llm.GenerateStreamEvent) error {
			if event.Usage != (llm.Usage{}) && input.OnEvent != nil {
				if streamErr := input.OnEvent("usage", map[string]interface{}{
					"input_tokens":       event.Usage.InputTokens,
					"output_tokens":      event.Usage.OutputTokens,
					"cache_read_tokens":  event.Usage.CacheReadTokens,
					"cache_write_tokens": event.Usage.CacheWriteTokens,
					"reasoning_tokens":   event.Usage.ReasoningTokens,
				}); streamErr != nil {
					return streamErr
				}
			}
			if event.GeneratedImage != nil && event.GeneratedImagePartial {
				return emitMediaImageDelta(input.OnEvent, event)
			}
			return nil
		})
	} else {
		output, err = s.llmClient.Generate(ctx, routeConfig, generateInput)
	}
	if err != nil {
		s.routeResolver.MarkRouteFailure(ctx, route, err)
		retErr = wrapUpstreamRequestError(err)
		_ = s.repo.UpdateMessageState(ctx, assistantMessage.ID, "error", classifyRunErrorCode(retErr), truncateError(messageErrorSummary(retErr), 255))
		return nil, retErr
	}
	s.routeResolver.MarkRouteSuccess(ctx, route)
	if output == nil || len(output.GeneratedImages) == 0 {
		retErr = ErrUpstreamEmptyResponse
		_ = s.repo.UpdateMessageState(ctx, assistantMessage.ID, "error", classifyRunErrorCode(retErr), truncateError(messageErrorSummary(retErr), 255))
		return nil, retErr
	}

	emitMediaEvent(input.OnEvent, "saving_artifact", "saving image")
	uploaded := make([]model.FileObject, 0, len(output.GeneratedImages))
	attachmentRows := make([]model.Attachment, 0, len(output.GeneratedImages))
	now := time.Now()
	for i, image := range output.GeneratedImages {
		image.URL = buildRouteMediaProxyURL(route.BaseURL, image.URL)
		data, mimeType, readErr := s.readGeneratedImage(ctx, image)
		if readErr != nil {
			retErr = readErr
			_ = s.repo.UpdateMessageState(ctx, assistantMessage.ID, "error", classifyRunErrorCode(retErr), truncateError(messageErrorSummary(retErr), 255))
			return nil, readErr
		}
		fileName := generatedImageFileName(route.PlatformModelName, now, i, len(output.GeneratedImages), mimeType)
		uploadResult, uploadErr := s.UploadFile(ctx, appupload.UploadFileInput{
			UserID:       input.UserID,
			Purpose:      "generated_image",
			FileName:     fileName,
			MimeType:     mimeType,
			DeclaredSize: int64(len(data)),
			Reader:       bytes.NewReader(data),
		})
		if uploadErr != nil {
			retErr = uploadErr
			_ = s.repo.UpdateMessageState(ctx, assistantMessage.ID, "error", classifyRunErrorCode(retErr), truncateError(messageErrorSummary(retErr), 255))
			return nil, uploadErr
		}
		file := uploadResult.File
		uploaded = append(uploaded, file)
		attachmentRows = append(attachmentRows, model.Attachment{
			ConversationID: input.ConversationID,
			MessageID:      assistantMessage.ID,
			UserID:         input.UserID,
			FileID:         file.FileID,
			Kind:           "image",
			FileName:       file.FileName,
			MimeType:       file.DetectedMIME,
			FileSize:       file.SizeBytes,
			SHA256:         file.SHA256,
			StoragePath:    file.StoragePath,
			Status:         "active",
			UploadedAt:     now,
		})
	}
	usage := output.Usage
	if reuseUserMessage {
		assistantMessage.InputTokens = usage.InputTokens
		assistantMessage.CacheReadTokens = usage.CacheReadTokens
		assistantMessage.CacheWriteTokens = usage.CacheWriteTokens
	} else {
		userMessage.InputTokens = usage.InputTokens
		userMessage.CacheReadTokens = usage.CacheReadTokens
		userMessage.CacheWriteTokens = usage.CacheWriteTokens
		userMessage.TokenUsage = usage.InputTokens + usage.CacheReadTokens + usage.CacheWriteTokens
	}

	content := generatedImageMarkdown(uploaded)
	latencyMS := time.Since(startedAt).Milliseconds()
	// 上游与文件上传已完成后，数据库侧的附件、用量和完成态仍需保持原子一致。
	if reuseUserMessage {
		if err = s.repo.CompleteAssistantMessageWithGeneratedAttachments(ctx,
			assistantMessage.ID,
			repository.AssistantMessageCompletionUpdate{
				ContentType:      "image",
				Content:          content,
				InputTokens:      usage.InputTokens,
				OutputTokens:     usage.OutputTokens,
				CacheReadTokens:  usage.CacheReadTokens,
				CacheWriteTokens: usage.CacheWriteTokens,
				ReasoningTokens:  usage.ReasoningTokens,
				LatencyMS:        latencyMS,
				Status:           "success",
			},
			attachmentRows,
		); err != nil {
			retErr = err
			return nil, err
		}
	} else {
		if err = s.repo.CompleteAssistantMessageWithAttachments(ctx,
			userMessage.ID,
			repository.MessageUsageUpdate{
				InputTokens:      usage.InputTokens,
				CacheReadTokens:  usage.CacheReadTokens,
				CacheWriteTokens: usage.CacheWriteTokens,
			},
			assistantMessage.ID,
			repository.AssistantMessageCompletionUpdate{
				ContentType:     "image",
				Content:         content,
				OutputTokens:    usage.OutputTokens,
				ReasoningTokens: usage.ReasoningTokens,
				LatencyMS:       latencyMS,
				Status:          "success",
			},
			attachmentRows,
		); err != nil {
			retErr = err
			return nil, err
		}
	}
	assistantMessage.Content = content
	assistantMessage.OutputTokens = usage.OutputTokens
	assistantMessage.ReasoningTokens = usage.ReasoningTokens
	assistantMessage.TokenUsage = assistantMessage.OutputTokens + assistantMessage.ReasoningTokens
	if reuseUserMessage {
		assistantMessage.TokenUsage += usage.InputTokens + usage.CacheReadTokens + usage.CacheWriteTokens
	}
	assistantMessage.LatencyMS = latencyMS
	assistantMessage.Status = "success"
	assistantMessage.Attachments = string(marshalAttachmentSnapshots(attachmentsFromFiles(uploaded)))
	run.InputTokens = usage.InputTokens
	run.OutputTokens = usage.OutputTokens
	run.CacheReadTokens = usage.CacheReadTokens
	run.CacheWriteTokens = usage.CacheWriteTokens
	run.ReasoningTokens = usage.ReasoningTokens

	return &SendMessageResult{
		UserMessage:         *userMessage,
		AssistantMessage:    *assistantMessage,
		MetadataRefreshHint: conversationMetadataRefreshHint(*conversation, *userMessage),
		UpstreamID:          route.UpstreamID,
		UpstreamName:        route.UpstreamName,
		PlatformModelName:   route.PlatformModelName,
		RoutedBindingCode:   route.BindingCode,
		UpstreamModelName:   route.UpstreamModel,
		UpstreamProtocol:    route.Protocol,
		EffectiveOptions:    filteredOptions,
		UsageSpeed:          usage.Speed,
		UsageServiceTier:    usage.ServiceTier,
		RawUsageJSON:        usage.RawUsageJSON,
		CacheWrite5mTokens:  usage.CacheWrite5mTokens,
		CacheWrite1hTokens:  usage.CacheWrite1hTokens,
		LatencyMS:           latencyMS,
		StartedAt:           startedAt,
	}, nil
}

func mediaImageUserContentType(taskType MediaImageTaskType) string {
	if taskType == MediaImageTaskEdit {
		return "mixed"
	}
	return "text"
}

func mediaImageRunningMessage(taskType MediaImageTaskType) string {
	if taskType == MediaImageTaskEdit {
		return "editing image"
	}
	return "generating image"
}

// resolveMediaImageEditInputs 读取图片编辑输入图，确保只有图片文件进入图片编辑协议。
func (s *Service) resolveMediaImageEditInputs(ctx context.Context, input MediaImageInput) ([]AttachmentInput, []llm.ContentPart, error) {
	if input.TaskType != MediaImageTaskEdit {
		return nil, nil, nil
	}
	attachments, err := s.resolveAttachments(ctx, input.UserID, input.FileIDs)
	if err != nil {
		return nil, nil, err
	}
	if len(attachments) == 0 || len(attachments) > maxMediaImageEditInputImages {
		if len(attachments) == 0 {
			return nil, nil, ErrMediaImageEditInputRequired
		}
		return nil, nil, ErrMediaImageEditTooManyInputs
	}
	parts := make([]llm.ContentPart, 0, len(attachments))
	for _, attachment := range attachments {
		if normalizeAttachmentKind(attachment.Kind, attachment.MimeType) != "image" {
			return nil, nil, ErrMediaImageEditInputInvalid
		}
		part, readErr := s.readMediaImageEditFile(ctx, input.UserID, attachment.FileID)
		if readErr != nil {
			return nil, nil, readErr
		}
		part.FileName = mediaImageEditInputFileName(attachment.FileName, part.MimeType)
		parts = append(parts, part)
	}
	return attachments, parts, nil
}

func (s *Service) resolveMediaVideoInputs(ctx context.Context, input MediaVideoInput) ([]AttachmentInput, []llm.ContentPart, error) {
	if len(input.FileIDs) == 0 {
		return nil, nil, nil
	}
	attachments, err := s.resolveAttachments(ctx, input.UserID, input.FileIDs)
	if err != nil {
		return nil, nil, err
	}
	if len(attachments) > maxMediaVideoInputImages {
		return nil, nil, ErrMediaImageEditTooManyInputs
	}
	parts := make([]llm.ContentPart, 0, len(attachments))
	for _, attachment := range attachments {
		if normalizeAttachmentKind(attachment.Kind, attachment.MimeType) != "image" {
			return nil, nil, ErrMediaImageEditInputInvalid
		}
		part, readErr := s.readMediaImageEditFile(ctx, input.UserID, attachment.FileID)
		if readErr != nil {
			return nil, nil, readErr
		}
		part.FileName = mediaImageEditInputFileName(attachment.FileName, part.MimeType)
		parts = append(parts, part)
	}
	return attachments, parts, nil
}

func (s *Service) resolveMediaImageEditMask(ctx context.Context, userID uint, fileID string) (*llm.ContentPart, error) {
	if strings.TrimSpace(fileID) == "" {
		return nil, nil
	}
	part, err := s.readMediaImageEditFile(ctx, userID, fileID)
	if err != nil {
		return nil, err
	}
	return &part, nil
}

func (s *Service) readMediaImageEditFile(ctx context.Context, userID uint, fileID string) (llm.ContentPart, error) {
	content, err := s.OpenFileContent(ctx, userID, strings.TrimSpace(fileID))
	if err != nil {
		return llm.ContentPart{}, err
	}
	defer content.Reader.Close() //nolint:errcheck

	limit := s.cfg.Snapshot().MaxUploadFileBytes
	if limit <= 0 {
		limit = 20 * 1024 * 1024
	}
	data, err := io.ReadAll(io.LimitReader(content.Reader, limit+1))
	if err != nil {
		return llm.ContentPart{}, err
	}
	if int64(len(data)) > limit {
		return llm.ContentPart{}, ErrFileTooLarge
	}
	mimeType := strings.TrimSpace(content.ContentType)
	if mimeType == "" {
		mimeType = strings.TrimSpace(content.File.DetectedMIME)
	}
	data, mimeType, err = normalizeMediaImageEditInput(data, mimeType)
	if err != nil {
		return llm.ContentPart{}, ErrMediaImageEditInputInvalid
	}
	return llm.ContentPart{
		Kind:     llm.ContentPartImage,
		MimeType: mimeType,
		Data:     data,
		FileName: mediaImageEditInputFileName(content.File.FileName, mimeType),
	}, nil
}

// emitMediaEvent 输出媒体任务状态事件；失败不影响主流程。
func emitMediaEvent(onEvent func(string, map[string]interface{}) error, status string, message string) {
	if onEvent == nil {
		return
	}
	_ = onEvent("media_status", map[string]interface{}{
		"status":  status,
		"message": message,
	})
}

func emitMediaImageDelta(onEvent func(string, map[string]interface{}) error, event llm.GenerateStreamEvent) error {
	if onEvent == nil || event.GeneratedImage == nil {
		return nil
	}
	image := event.GeneratedImage
	if strings.TrimSpace(image.B64JSON) == "" {
		return nil
	}
	return onEvent("media_image_delta", map[string]interface{}{
		"index":          event.GeneratedImageIndex,
		"b64_json":       image.B64JSON,
		"mime_type":      strings.TrimSpace(image.MIMEType),
		"revised_prompt": strings.TrimSpace(image.RevisedPrompt),
	})
}

func mediaImageStreamEnabled(protocol string, upstreamModel string, capabilitiesJSON string) bool {
	return llm.SupportsImageGenerationStream(protocol, upstreamModel) && !mediaImageStreamExplicitlyDisabled(capabilitiesJSON)
}

func mediaImageStreamExplicitlyDisabled(capabilitiesJSON string) bool {
	raw := strings.TrimSpace(capabilitiesJSON)
	if raw == "" {
		return false
	}
	var caps mediaImageCapabilities
	if err := json.Unmarshal([]byte(raw), &caps); err != nil {
		return false
	}
	return caps.Image.Stream != nil && !*caps.Image.Stream
}

// readGeneratedImage 读取上游图片结果，并统一校验为可保存的图片字节。
// 上游临时 URL 只用于服务端下载，最终不会直接写入消息内容，避免长期依赖外部地址。
func (s *Service) readGeneratedImage(ctx context.Context, image llm.GeneratedImage) ([]byte, string, error) {
	mimeType := strings.TrimSpace(image.MIMEType)
	if mimeType == "" {
		mimeType = "image/png"
	}
	if b64 := strings.TrimSpace(image.B64JSON); b64 != "" {
		data, err := base64.StdEncoding.DecodeString(stripBase64DataURLPrefix(b64))
		if err != nil {
			return nil, mimeType, err
		}
		return validateGeneratedImageBytes(data, mimeType)
	}
	url := strings.TrimSpace(image.URL)
	if url == "" {
		return nil, mimeType, ErrUpstreamEmptyResponse
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, mimeType, err
	}
	cfg := s.cfg.Snapshot()
	client := security.NewOutboundHTTPClient(cfg.Env, cfg.SSRFProtectionEnabled, 60*time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return nil, mimeType, err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, mimeType, fmt.Errorf("download generated image failed: HTTP %d", resp.StatusCode)
	}
	if contentType := strings.TrimSpace(resp.Header.Get("Content-Type")); strings.HasPrefix(strings.ToLower(contentType), "image/") {
		mimeType = strings.Split(contentType, ";")[0]
	}
	limit := s.cfg.Snapshot().MaxUploadFileBytes
	if limit <= 0 {
		limit = 20 * 1024 * 1024
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, mimeType, err
	}
	if int64(len(data)) > limit {
		return nil, mimeType, ErrFileTooLarge
	}
	return validateGeneratedImageBytes(data, mimeType)
}

// stripBase64DataURLPrefix 兼容 data URL 和纯 base64 两种上游返回格式。
func stripBase64DataURLPrefix(value string) string {
	normalized := strings.TrimSpace(value)
	if !strings.HasPrefix(strings.ToLower(normalized), "data:") {
		return normalized
	}
	if index := strings.Index(normalized, ","); index >= 0 {
		return strings.TrimSpace(normalized[index+1:])
	}
	return normalized
}

// validateGeneratedImageBytes 使用文件头重新识别 MIME，防止把非图片响应保存成图片文件。
func validateGeneratedImageBytes(data []byte, declaredMIME string) ([]byte, string, error) {
	detected := detectGeneratedImageMIME(data)
	if detected == "" {
		return nil, strings.TrimSpace(declaredMIME), fmt.Errorf("generated image content is not a supported image")
	}
	return data, detected, nil
}

// detectGeneratedImageMIME 识别当前支持落库的图片格式。
func detectGeneratedImageMIME(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	if len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		return "image/png"
	}
	if len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff {
		return "image/jpeg"
	}
	if len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")) {
		return "image/webp"
	}
	if len(data) >= 6 && (bytes.Equal(data[:6], []byte("GIF87a")) || bytes.Equal(data[:6], []byte("GIF89a"))) {
		return "image/gif"
	}
	return ""
}

// imageFileExtension 根据最终识别出的 MIME 决定生成文件扩展名。
func imageFileExtension(mimeType string) string {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".png"
	}
}

// generatedImageFileName 使用模型名和生成时间构造稳定可读的文件名。
func generatedImageFileName(modelName string, capturedAt time.Time, index int, total int, mimeType string) string {
	base := sanitizeGeneratedImageFileBase(modelName)
	timestamp := fmt.Sprintf("%s-%03d", capturedAt.Format("20060102-150405"), capturedAt.Nanosecond()/int(time.Millisecond))
	if total > 1 {
		return fmt.Sprintf("%s-%s-%02d%s", base, timestamp, index+1, imageFileExtension(mimeType))
	}
	return fmt.Sprintf("%s-%s%s", base, timestamp, imageFileExtension(mimeType))
}

// sanitizeGeneratedImageFileBase 清理模型名，确保生成文件名不含路径分隔符或不可控字符。
func sanitizeGeneratedImageFileBase(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "image"
	}
	var builder strings.Builder
	builder.Grow(len(normalized))
	lastDash := false
	for _, item := range normalized {
		allowed := (item >= 'a' && item <= 'z') ||
			(item >= 'A' && item <= 'Z') ||
			(item >= '0' && item <= '9') ||
			item == '.' ||
			item == '_' ||
			item == '-'
		if allowed {
			builder.WriteRune(item)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	result := strings.Trim(builder.String(), ".-_")
	if result == "" {
		return "image"
	}
	if len(result) > 80 {
		result = strings.Trim(result[:80], ".-_")
	}
	if result == "" {
		return "image"
	}
	return result
}

// generatedImageMarkdown 将已保存的文件对象转换为受保护文件接口的 markdown 引用。
func generatedImageMarkdown(files []model.FileObject) string {
	blocks := make([]string, 0, len(files))
	for i, file := range files {
		alt := "Generated image"
		if len(files) > 1 {
			alt = fmt.Sprintf("Generated image %d", i+1)
		}
		blocks = append(blocks, fmt.Sprintf("![%s](/api/v1/files/%s/content)", alt, file.FileID))
	}
	return strings.Join(blocks, "\n\n")
}

// attachmentsFromFiles 生成消息附件快照，供流式完成事件立即返回给前端。
func attachmentsFromFiles(files []model.FileObject) []AttachmentInput {
	items := make([]AttachmentInput, 0, len(files))
	for _, file := range files {
		items = append(items, AttachmentInput{
			FileObjID:        file.ID,
			FileID:           file.FileID,
			Kind:             "image",
			FileName:         file.FileName,
			MimeType:         file.MimeType,
			DetectedMIME:     file.DetectedMIME,
			FileCategory:     file.FileCategory,
			FileSize:         file.SizeBytes,
			SHA256:           file.SHA256,
			StoragePath:      file.StoragePath,
			ProcessingStatus: file.ProcessingStatus,
			ProcessingReady:  file.ProcessingReady,
		})
	}
	return items
}

// MediaVideoInput 定义视频生成任务的应用层入参。
type MediaVideoInput struct {
	UserID                uint
	ConversationID        uint
	RequestID             string
	Prompt                string
	PlatformModelName     string
	Options               map[string]interface{}
	ClientRunID           string
	FileIDs               []string
	ParentMessagePublicID string
	SourceMessagePublicID string
	BranchReason          string
	OnEvent               func(eventType string, payload map[string]interface{}) error
}

// StreamMediaVideo 执行视频生成任务：通过路由解析上游，创建异步任务并轮询直到完成，保存视频文件。
func (s *Service) StreamMediaVideo(ctx context.Context, input MediaVideoInput) (*SendMessageResult, error) {
	if strings.TrimSpace(input.Prompt) == "" {
		return nil, ErrMediaImagePromptRequired
	}
	if s.routeResolver == nil || s.llmClient == nil {
		return nil, ErrModelRouteNotConfigured
	}
	ctx = context.WithoutCancel(ctx)

	// clientRunID 是幂等键
	runID := normalizeRunID(input.ClientRunID)
	if runID == "" {
		runID = "run_" + normalizePublicID(uuid.NewString())
	}
	existingRuns, err := s.repo.ListConversationRunsByRunIDs(ctx, input.UserID, input.ConversationID, []string{runID})
	if err != nil {
		return nil, err
	}
	if len(existingRuns) > 0 {
		return nil, ErrDuplicateMessageGenerationRun
	}
	cancelCtx, cancel := context.WithCancel(ctx)
	ctx = cancelCtx
	s.generationStreams.register(ctx, runID, input.UserID, cancel)

	startedAt := time.Now()
	conversation, err := s.repo.GetConversationByUser(ctx, input.ConversationID, input.UserID)
	if err != nil {
		return nil, ErrConversationNotFound
	}

	platformModelName := strings.TrimSpace(input.PlatformModelName)
	if platformModelName == "" {
		platformModelName = strings.TrimSpace(conversation.Model)
	}
	if platformModelName == "" {
		return nil, ErrModelRouteNotConfigured
	}

	route, err := s.routeResolver.ResolveRoute(ctx, channel.ResolveRouteInput{
		PlatformModelName: platformModelName,
		TaskType:          channel.TaskTypeVideoGeneration,
		Scope:             channel.RouteScopeUser,
		UserID:            input.UserID,
		ConversationID:    input.ConversationID,
		RequestID:         strings.TrimSpace(input.RequestID),
	})
	if err != nil {
		return nil, ErrModelRouteNotConfigured
	}

	// 更新会话模型
	if strings.TrimSpace(conversation.Model) != strings.TrimSpace(route.PlatformModelName) {
		conversation.Model = strings.TrimSpace(route.PlatformModelName)
		conversation.Provider = inferProvider(conversation.Model)
		if err = s.repo.UpdateConversationModel(ctx, input.ConversationID, conversation.Model, conversation.Provider); err != nil {
			return nil, err
		}
	}

	normalizedBranchReason := normalizeBranchReason(input.BranchReason)
	branchState, err := s.resolveMessageBranch(ctx, input.ConversationID, input.UserID, input.ParentMessagePublicID, input.SourceMessagePublicID, normalizedBranchReason)
	if err != nil {
		return nil, err
	}

	resolvedAttachments, videoInputImages, err := s.resolveMediaVideoInputs(ctx, input)
	if err != nil {
		return nil, err
	}
	attachmentsJSON := marshalAttachmentSnapshots(resolvedAttachments)

	run := &model.Run{
		RunID:              runID,
		RequestID:          strings.TrimSpace(input.RequestID),
		UserID:             input.UserID,
		ConversationID:     input.ConversationID,
		TaskType:           "video_generation",
		Endpoint:           llm.EndpointVideoGenerations,
		Provider:           strings.TrimSpace(conversation.Provider),
		ProviderProtocol:   route.Protocol,
		UpstreamID:         route.UpstreamID,
		UpstreamModelID:    route.UpstreamModelID,
		UpstreamName:       route.UpstreamName,
		RequestedModelName: platformModelName,
		PlatformModelName:  route.PlatformModelName,
		RoutedBindingCode:  route.BindingCode,
		ModelVendor:        route.ModelVendor,
		ModelIcon:          route.ModelIcon,
		UpstreamModelName:  route.UpstreamModel,
		Status:             "error",
		StartedAt:          startedAt,
	}
	var retErr error
	defer func() {
		endedAt := time.Now()
		run.EndedAt = &endedAt
		run.TotalLatencyMS = endedAt.Sub(startedAt).Milliseconds()
		if retErr == nil {
			run.Status = "success"
		} else {
			run.Status = "error"
			run.ErrorCode = classifyRunErrorCode(retErr)
			run.ErrorMessage = truncateError(messageErrorSummary(retErr), 255)
		}
		if err := s.repo.CreateConversationRun(context.WithoutCancel(ctx), run); err != nil && s.logger != nil {
			s.logger.Error("create_video_conversation_run_failed",
				zap.String("trace_id", traceid.FromContext(ctx)),
				zap.String("run_id", run.RunID),
				zap.Error(err),
			)
		}
	}()

	userMessage := &model.Message{
		ConversationID:  input.ConversationID,
		UserID:          input.UserID,
		PublicID:        normalizePublicID(uuid.NewString()),
		ParentMessageID: branchState.ParentMessageID,
		RunID:           runID,
		Role:            "user",
		ContentType:     "text",
		Content:         strings.TrimSpace(input.Prompt),
		BranchReason:    normalizedBranchReason,
		SourceMessageID: branchState.SourceMessageID,
		TokenUsage:      estimateTokens(input.Prompt),
		InputTokens:     estimateTokens(input.Prompt),
		Status:          "success",
		Attachments:     attachmentsJSON,
	}

	assistantMessage := &model.Message{
		ConversationID: input.ConversationID,
		UserID:         input.UserID,
		PublicID:       normalizePublicID(uuid.NewString()),
		RunID:          runID,
		Role:           "assistant",
		ContentType:    "video",
		Content:        "",
		BranchReason:   normalizedBranchReason,
		Status:         "pending",
		Attachments:    "[]",
	}

	userAttachmentRows := make([]model.Attachment, 0, len(resolvedAttachments))
	if len(resolvedAttachments) > 0 {
		now := time.Now()
		for _, item := range resolvedAttachments {
			userAttachmentRows = append(userAttachmentRows, model.Attachment{
				ConversationID: input.ConversationID,
				UserID:         input.UserID,
				FileID:         strings.TrimSpace(item.FileID),
				Kind:           normalizeAttachmentKind(item.Kind, item.MimeType),
				FileName:       strings.TrimSpace(item.FileName),
				MimeType:       strings.TrimSpace(item.MimeType),
				FileSize:       item.FileSize,
				SHA256:         strings.TrimSpace(item.SHA256),
				StoragePath:    strings.TrimSpace(item.StoragePath),
				Status:         "active",
				MetaJSON:       strings.TrimSpace(item.MetaJSON),
				UploadedAt:     now,
			})
		}
	}

	if err = s.repo.CreateMessagePairWithUserAttachments(ctx, userMessage, assistantMessage, userAttachmentRows); err != nil {
		retErr = err
		return nil, err
	}
	userMessage.ParentPublicID = branchState.ParentPublicID
	userMessage.SourcePublicID = branchState.SourcePublicID
	assistantMessage.ParentPublicID = userMessage.PublicID
	traceRecorder := newMessageTraceRecorder(s, ctx, assistantMessage, input.OnEvent)
	defer func() {
		if retErr != nil && traceRecorder != nil {
			traceRecorder.fail(retErr)
			traceRecorder.attachToMessage(assistantMessage)
		}
	}()
	emitMediaEvent(input.OnEvent, "queued", "video task queued")

	emitMediaEvent(input.OnEvent, "running", "generating video")
	cfg := s.cfg.Snapshot()
	httpClient := newVideoGenerationHTTPClient(cfg.Env, cfg.SSRFProtectionEnabled, 300*time.Second)
	videoBaseURL := videoGenerationBaseURL(route)

	createResult, createPath, err := createVideoGenerationTask(ctx, httpClient, route, videoBaseURL, input, videoInputImages)
	if err != nil {
		retErr = err
		s.routeResolver.MarkRouteFailure(ctx, route, err)
		_ = s.repo.UpdateMessageState(ctx, assistantMessage.ID, "error", classifyRunErrorCode(retErr), truncateError(messageErrorSummary(retErr), 255))
		return nil, retErr
	}
	if strings.TrimSpace(createResult.TaskID) == "" {
		retErr = fmt.Errorf("视频任务创建失败：未返回任务ID")
		_ = s.repo.UpdateMessageState(ctx, assistantMessage.ID, "error", classifyRunErrorCode(retErr), truncateError(messageErrorSummary(retErr), 255))
		return nil, retErr
	}

	s.routeResolver.MarkRouteSuccess(ctx, route)

	// 轮询任务状态
	taskID := createResult.TaskID
	var videoURL string
	for i := 0; i < videoTaskPollAttempts; i++ {
		select {
		case <-ctx.Done():
			retErr = ctx.Err()
			_ = s.repo.UpdateMessageState(ctx, assistantMessage.ID, "error", classifyRunErrorCode(retErr), "generation cancelled")
			return nil, retErr
		case <-time.After(videoTaskPollInterval):
		}

		pollResult, err := pollVideoGenerationTask(ctx, httpClient, route, videoBaseURL, createPath, taskID)
		if err != nil {
			continue
		}

		if input.OnEvent != nil {
			_ = input.OnEvent("media_status", map[string]interface{}{
				"status":  pollResult.Status,
				"message": fmt.Sprintf("视频生成中... (%d/%d)", i+1, videoTaskPollAttempts),
			})
		}

		if isVideoTaskSuccessStatus(pollResult.Status) {
			videoURL = strings.TrimSpace(pollResult.VideoURL)
			break
		}
		if isVideoTaskFailedStatus(pollResult.Status) {
			errMsg := firstNonEmptyString(pollResult.ErrorMessage, "视频生成失败")
			retErr = fmt.Errorf("%s", errMsg)
			_ = s.repo.UpdateMessageState(ctx, assistantMessage.ID, "error", classifyRunErrorCode(retErr), truncateError(errMsg, 255))
			return nil, retErr
		}
	}

	if videoURL == "" {
		retErr = fmt.Errorf("视频生成超时")
		_ = s.repo.UpdateMessageState(ctx, assistantMessage.ID, "error", classifyRunErrorCode(retErr), "generation timeout")
		return nil, retErr
	}

	// 下载视频并保存为文件
	emitMediaEvent(input.OnEvent, "saving_artifact", "saving video")
	videoData, mimeType, downloadErr := s.downloadGeneratedVideo(ctx, videoURL)
	if downloadErr != nil {
		retErr = downloadErr
		_ = s.repo.UpdateMessageState(ctx, assistantMessage.ID, "error", classifyRunErrorCode(retErr), truncateError(messageErrorSummary(retErr), 255))
		return nil, retErr
	}

	fileName := generatedVideoFileName(route.PlatformModelName, time.Now(), mimeType)
	uploadResult, uploadErr := s.UploadFile(ctx, appupload.UploadFileInput{
		UserID:       input.UserID,
		Purpose:      "generated_video",
		FileName:     fileName,
		MimeType:     mimeType,
		DeclaredSize: int64(len(videoData)),
		Reader:       bytes.NewReader(videoData),
	})
	if uploadErr != nil {
		retErr = uploadErr
		_ = s.repo.UpdateMessageState(ctx, assistantMessage.ID, "error", classifyRunErrorCode(retErr), truncateError(messageErrorSummary(retErr), 255))
		return nil, retErr
	}

	file := uploadResult.File
	now := time.Now()
	attachmentRow := model.Attachment{
		ConversationID: input.ConversationID,
		MessageID:      assistantMessage.ID,
		UserID:         input.UserID,
		FileID:         file.FileID,
		Kind:           "video",
		FileName:       file.FileName,
		MimeType:       file.DetectedMIME,
		FileSize:       file.SizeBytes,
		SHA256:         file.SHA256,
		StoragePath:    file.StoragePath,
		Status:         "active",
		UploadedAt:     now,
	}

	content := ""
	latencyMS := time.Since(startedAt).Milliseconds()
	// 假设视频生成按次计费，这里可以根据实际情况调整 token 计算
	outputTokens := estimateTokens(input.Prompt)
	if err = s.repo.CompleteAssistantMessageWithAttachments(ctx,
		userMessage.ID,
		repository.MessageUsageUpdate{
			InputTokens:      userMessage.InputTokens,
			CacheReadTokens:  0,
			CacheWriteTokens: 0,
		},
		assistantMessage.ID,
		repository.AssistantMessageCompletionUpdate{
			ContentType:     "video",
			Content:         content,
			OutputTokens:    outputTokens,
			ReasoningTokens: 0,
			LatencyMS:       latencyMS,
			Status:          "success",
		},
		[]model.Attachment{attachmentRow},
	); err != nil {
		retErr = err
		return nil, err
	}

	assistantMessage.Content = content
	assistantMessage.OutputTokens = outputTokens
	assistantMessage.TokenUsage = outputTokens
	assistantMessage.LatencyMS = latencyMS
	assistantMessage.Status = "success"
	assistantMessage.Attachments = string(marshalAttachmentSnapshots(attachmentsFromFiles([]model.FileObject{file})))
	run.InputTokens = userMessage.InputTokens
	run.OutputTokens = outputTokens

	s.maybeGenerateConversationMetadataAsync(*conversation, *userMessage)

	return &SendMessageResult{
		UserMessage:       *userMessage,
		AssistantMessage:  *assistantMessage,
		UpstreamID:        route.UpstreamID,
		UpstreamName:      route.UpstreamName,
		PlatformModelName: route.PlatformModelName,
		RoutedBindingCode: route.BindingCode,
		UpstreamModelName: route.UpstreamModel,
		UpstreamProtocol:  route.Protocol,
		EffectiveOptions:  input.Options,
		LatencyMS:         latencyMS,
	}, nil
}

type videoTaskCreateResult struct {
	TaskID string
	Status string
}

type videoTaskPollResult struct {
	Status       string
	VideoURL     string
	ErrorMessage string
}

func createVideoGenerationTask(ctx context.Context, httpClient *http.Client, route *channel.ResolvedRoute, baseURL string, input MediaVideoInput, inputImages []llm.ContentPart) (videoTaskCreateResult, string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(route.BaseURL)
	}
	var lastErr error
	for _, path := range videoCreatePathsForRoute(baseURL) {
		createBody := buildVideoGenerationCreateBody(route.UpstreamModel, input.Prompt, input.Options, inputImages)
		if path == seedanceCreatePath {
			createBody = buildSeedanceVideoGenerationCreateBody(route.UpstreamModel, input.Prompt, input.Options, inputImages)
		}
		createJSON, _ := json.Marshal(createBody)
		createURL := buildVideoEndpointURL(baseURL, path)
		var createResp *http.Response
		var err error
		for attempt := 1; attempt <= videoCreateAttempts; attempt++ {
			createReq, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, createURL, bytes.NewReader(createJSON))
			if reqErr != nil {
				return videoTaskCreateResult{}, path, fmt.Errorf("创建视频请求失败: %w", reqErr)
			}
			createReq.Header.Set("Content-Type", "application/json")
			createReq.Header.Set("Accept", "application/json")
			if apiKey := strings.TrimSpace(route.APIKey); apiKey != "" {
				createReq.Header.Set("Authorization", "Bearer "+apiKey)
			}
			setVideoRouteHeaders(createReq, route.HeadersJSON)

			createResp, err = httpClient.Do(createReq)
			if err == nil {
				break
			}
			lastErr = wrapUpstreamRequestError(err)
			if attempt >= videoCreateAttempts || !shouldRetryVideoCreateError(err) {
				return videoTaskCreateResult{}, path, lastErr
			}
			if waitErr := waitVideoCreateRetry(ctx, attempt); waitErr != nil {
				return videoTaskCreateResult{}, path, waitErr
			}
		}
		body, readErr := io.ReadAll(createResp.Body)
		_ = createResp.Body.Close()
		if readErr != nil {
			return videoTaskCreateResult{}, path, readErr
		}
		if createResp.StatusCode < 200 || createResp.StatusCode >= 300 {
			lastErr = fmt.Errorf("视频生成失败(HTTP %d): %s", createResp.StatusCode, strings.TrimSpace(string(body)))
			if path != seedanceCreatePath && shouldTryNextVideoEndpoint(createResp.StatusCode) {
				continue
			}
			return videoTaskCreateResult{}, path, lastErr
		}
		result, err := parseVideoTaskCreateResult(body)
		if err != nil {
			return videoTaskCreateResult{}, path, err
		}
		if strings.TrimSpace(result.TaskID) == "" {
			return videoTaskCreateResult{}, path, fmt.Errorf("视频任务创建失败：未返回任务ID")
		}
		return result, path, nil
	}
	if lastErr != nil {
		return videoTaskCreateResult{}, "", lastErr
	}
	return videoTaskCreateResult{}, "", fmt.Errorf("视频生成失败：没有可用视频端点")
}

func pollVideoGenerationTask(ctx context.Context, httpClient *http.Client, route *channel.ResolvedRoute, baseURL string, createPath string, taskID string) (videoTaskPollResult, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(route.BaseURL)
	}
	pollPath := videoPollPathForCreatePath(createPath)
	pollURL := buildVideoEndpointURL(baseURL, pollPath+taskID)
	pollReq, err := http.NewRequestWithContext(ctx, http.MethodGet, pollURL, nil)
	if err != nil {
		return videoTaskPollResult{}, err
	}
	pollReq.Header.Set("Accept", "application/json")
	if apiKey := strings.TrimSpace(route.APIKey); apiKey != "" {
		pollReq.Header.Set("Authorization", "Bearer "+apiKey)
	}
	setVideoRouteHeaders(pollReq, route.HeadersJSON)
	pollResp, err := httpClient.Do(pollReq)
	if err != nil {
		return videoTaskPollResult{}, err
	}
	body, readErr := io.ReadAll(pollResp.Body)
	_ = pollResp.Body.Close()
	if readErr != nil {
		return videoTaskPollResult{}, readErr
	}
	if pollResp.StatusCode < 200 || pollResp.StatusCode >= 300 {
		return videoTaskPollResult{}, fmt.Errorf("视频任务查询失败(HTTP %d): %s", pollResp.StatusCode, strings.TrimSpace(string(body)))
	}
	return parseVideoTaskPollResult(body)
}

func buildVideoGenerationCreateBody(modelName string, prompt string, options map[string]interface{}, inputImages []llm.ContentPart) map[string]interface{} {
	createBody := map[string]interface{}{
		"model":      strings.TrimSpace(modelName),
		"prompt":     strings.TrimSpace(prompt),
		"duration":   5,
		"seconds":    "5",
		"resolution": "720p",
		"ratio":      "16:9",
		"size":       "1280x720",
	}
	_, hasDurationOption := options["duration"]
	_, hasSecondsOption := options["seconds"]
	for key, value := range options {
		if strings.TrimSpace(key) == "" {
			continue
		}
		createBody[key] = value
	}
	syncVideoDurationAliases(createBody, hasDurationOption, hasSecondsOption)
	metadata := map[string]interface{}{}
	if existing, ok := createBody["metadata"].(map[string]interface{}); ok {
		for key, value := range existing {
			metadata[key] = value
		}
	}
	for _, key := range []string{"resolution", "ratio", "watermark", "seed", "frames", "camera_fixed", "service_tier", "generate_audio", "return_last_frame"} {
		if value, ok := createBody[key]; ok {
			metadata[key] = value
		}
	}
	if len(metadata) > 0 {
		createBody["metadata"] = metadata
	}
	if len(inputImages) > 0 {
		images := make([]string, 0, len(inputImages))
		for _, image := range inputImages {
			if len(image.Data) == 0 {
				continue
			}
			mimeType := strings.TrimSpace(image.MimeType)
			if mimeType == "" {
				mimeType = "image/png"
			}
			images = append(images, "data:"+mimeType+";base64,"+base64.StdEncoding.EncodeToString(image.Data))
		}
		if len(images) > 0 {
			createBody["images"] = images
		}
	}
	return createBody
}

func buildSeedanceVideoGenerationCreateBody(modelName string, prompt string, options map[string]interface{}, inputImages []llm.ContentPart) map[string]interface{} {
	content := []map[string]interface{}{
		{
			"type": "text",
			"text": strings.TrimSpace(prompt),
		},
	}
	for index, image := range inputImages {
		if len(image.Data) == 0 {
			continue
		}
		mimeType := strings.TrimSpace(image.MimeType)
		if mimeType == "" {
			mimeType = "image/png"
		}
		role := "reference_image"
		if index == 0 {
			role = "first_frame"
		}
		content = append(content, map[string]interface{}{
			"type": "image_url",
			"image_url": map[string]interface{}{
				"url": "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(image.Data),
			},
			"role": role,
		})
	}

	createBody := map[string]interface{}{
		"model":      strings.TrimSpace(modelName),
		"content":    content,
		"duration":   int64(5),
		"ratio":      "16:9",
		"resolution": "720p",
	}
	if len(options) == 0 {
		return createBody
	}
	hasDurationOption := false
	if _, ok := options["duration"]; ok {
		hasDurationOption = true
	}
	for key, value := range options {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			continue
		}
		switch normalizedKey {
		case "duration":
			setSeedanceVideoOption(createBody, "duration", value)
		case "seconds":
			if !hasDurationOption {
				setSeedanceVideoOption(createBody, "duration", value)
			}
		case "ratio", "resolution", "frames", "seed", "generate_audio", "camera_fixed", "watermark", "draft", "service_tier", "execution_expires_after":
			setSeedanceVideoOption(createBody, normalizedKey, value)
		case "size":
			applySeedanceSizeOption(createBody, value)
		}
	}
	return createBody
}

func setSeedanceVideoOption(createBody map[string]interface{}, key string, value interface{}) {
	if createBody == nil {
		return
	}
	if value == nil {
		return
	}
	if text, ok := value.(string); ok {
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}
		createBody[key] = normalizeSeedanceNumericOption(key, text)
		return
	}
	createBody[key] = normalizeSeedanceNumericOption(key, value)
}

func normalizeSeedanceNumericOption(key string, value interface{}) interface{} {
	switch strings.TrimSpace(key) {
	case "duration", "frames", "seed":
		switch item := value.(type) {
		case string:
			if parsed, err := strconv.ParseInt(strings.TrimSpace(item), 10, 64); err == nil {
				return parsed
			}
		case json.Number:
			if parsed, err := item.Int64(); err == nil {
				return parsed
			}
		case float64:
			if item == float64(int64(item)) {
				return int64(item)
			}
		case float32:
			if item == float32(int64(item)) {
				return int64(item)
			}
		}
	}
	return value
}

func applySeedanceSizeOption(createBody map[string]interface{}, value interface{}) {
	if createBody == nil || value == nil {
		return
	}
	size := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(fmt.Sprint(value)), " ", ""))
	switch size {
	case "1280x720", "1920x1080":
		createBody["ratio"] = "16:9"
	case "720x1280", "1080x1920":
		createBody["ratio"] = "9:16"
	case "1024x1024", "1080x1080":
		createBody["ratio"] = "1:1"
	}
	if strings.Contains(size, "1080") || strings.Contains(size, "1920") {
		createBody["resolution"] = "1080p"
		return
	}
	if strings.Contains(size, "720") || strings.Contains(size, "1280") {
		createBody["resolution"] = "720p"
	}
}

func syncVideoDurationAliases(createBody map[string]interface{}, hasDurationOption bool, hasSecondsOption bool) {
	if createBody == nil {
		return
	}
	if hasSecondsOption && !hasDurationOption {
		createBody["duration"] = createBody["seconds"]
		return
	}
	if hasDurationOption && !hasSecondsOption {
		createBody["seconds"] = fmt.Sprint(createBody["duration"])
	}
}

func setVideoRouteHeaders(req *http.Request, headersJSON string) {
	if req == nil {
		return
	}
	raw := strings.TrimSpace(headersJSON)
	if raw == "" {
		return
	}
	var headers map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &headers); err != nil {
		return
	}
	for key, value := range headers {
		if value == nil {
			continue
		}
		name := strings.TrimSpace(key)
		if name == "" {
			continue
		}
		text := strings.TrimSpace(fmt.Sprint(value))
		if text == "" {
			continue
		}
		req.Header.Set(name, text)
	}
}

func buildRouteMediaProxyURL(baseURL string, mediaURL string) string {
	raw := strings.TrimSpace(mediaURL)
	if raw == "" || strings.HasPrefix(strings.ToLower(raw), "data:") {
		return raw
	}
	targetURL, err := url.Parse(raw)
	if err != nil || targetURL.Scheme == "" || targetURL.Host == "" {
		return raw
	}
	if targetURL.Scheme != "http" && targetURL.Scheme != "https" {
		return raw
	}
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return raw
	}
	if !routeBaseSupportsMediaProxy(base) {
		return raw
	}
	if strings.EqualFold(targetURL.Host, base.Host) && strings.HasPrefix(strings.TrimRight(targetURL.EscapedPath(), "/"), "/media") {
		return raw
	}
	base.Path = strings.TrimRight(base.Path, "/")
	if videoBaseEndsWithVersionSegment(base.Path) {
		base.Path = strings.TrimRight(base.Path[:strings.LastIndex(base.Path, "/")], "/")
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/media"
	values := url.Values{}
	values.Set("url", raw)
	base.RawQuery = values.Encode()
	base.Fragment = ""
	return base.String()
}

func routeBaseSupportsMediaProxy(base *url.URL) bool {
	if base == nil {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(base.Hostname()))
	return host == "volcengine-proxy.3677635979.workers.dev"
}

func videoGenerationBaseURL(route *channel.ResolvedRoute) string {
	if route == nil {
		return ""
	}
	baseURL := strings.TrimSpace(route.BaseURL)
	if routeUsesVolcengineWorker(baseURL) && routeIsSeedanceVideo(route) {
		return defaultVolcengineArkBaseURL
	}
	return baseURL
}

func routeUsesVolcengineWorker(baseURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	return host == "volcengine-proxy.3677635979.workers.dev"
}

func routeIsSeedanceVideo(route *channel.ResolvedRoute) bool {
	if route == nil {
		return false
	}
	for _, value := range []string{
		route.PlatformModelName,
		route.UpstreamModel,
		route.BindingCode,
		route.ModelVendor,
		route.UpstreamName,
	} {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if strings.Contains(normalized, "seedance") || strings.Contains(normalized, "doubao-seedance") {
			return true
		}
	}
	return false
}

func routeBaseSupportsSeedanceNative(baseURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "ark.cn-beijing.volces.com" {
		return true
	}
	return strings.Contains(host, "volces.com") && strings.Contains(strings.ToLower(parsed.Path), "/api/v3")
}

func videoCreatePathsForRoute(baseURL string) []string {
	if routeBaseSupportsSeedanceNative(baseURL) {
		return []string{seedanceCreatePath}
	}
	if createPath, ok := videoCreatePathFromEndpointBase(baseURL); ok {
		result := []string{createPath}
		for _, candidate := range []string{openAIVideoCreatePath, taskVideoCreatePath, legacyVideoCreatePath, seedanceCreatePath} {
			if candidate != createPath {
				result = append(result, candidate)
			}
		}
		return result
	}
	return []string{openAIVideoCreatePath, taskVideoCreatePath, legacyVideoCreatePath}
}

func videoPollPathForCreatePath(createPath string) string {
	switch strings.TrimSpace(createPath) {
	case taskVideoCreatePath:
		return taskVideoPollPath
	case legacyVideoCreatePath:
		return legacyVideoPollPath
	case seedanceCreatePath:
		return seedancePollPath
	default:
		return openAIVideoPollPath
	}
}

func buildVideoEndpointURL(baseURL string, endpointPath string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	path := "/" + strings.TrimLeft(strings.TrimSpace(endpointPath), "/")
	if base == "" {
		return path
	}
	if createPath, ok := videoCreatePathFromEndpointBase(base); ok {
		if strings.EqualFold(path, createPath) {
			return base
		}
		pollPath := videoPollPathForCreatePath(createPath)
		if strings.HasPrefix(strings.ToLower(path), strings.ToLower(pollPath)) {
			return base + strings.TrimPrefix(path, createPath)
		}
		base = strings.TrimRight(base[:len(base)-len(createPath)], "/")
		if base == "" {
			return path
		}
	}
	if videoBaseEndsWithVersionSegment(base) && strings.HasPrefix(path, "/v1/") {
		path = strings.TrimPrefix(path, "/v1")
	}
	if !videoBaseEndsWithVersionSegment(base) {
		path = "/v1" + path
	}
	return base + path
}

func videoCreatePathFromEndpointBase(baseURL string) (string, bool) {
	normalized := strings.TrimRight(strings.ToLower(strings.TrimSpace(baseURL)), "/")
	for _, createPath := range []string{seedanceCreatePath, taskVideoCreatePath, legacyVideoCreatePath, openAIVideoCreatePath} {
		if strings.HasSuffix(normalized, strings.ToLower(createPath)) {
			return createPath, true
		}
	}
	return "", false
}

func videoBaseEndsWithVersionSegment(baseURL string) bool {
	value := strings.TrimRight(strings.ToLower(strings.TrimSpace(baseURL)), "/")
	if value == "" {
		return false
	}
	index := strings.LastIndex(value, "/")
	if index >= 0 {
		value = value[index+1:]
	}
	if len(value) < 2 || value[0] != 'v' {
		return false
	}
	for index := 1; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}

func shouldTryNextVideoEndpoint(statusCode int) bool {
	return statusCode == http.StatusNotFound || statusCode == http.StatusMethodNotAllowed || statusCode == http.StatusBadRequest
}

func shouldRetryVideoCreateError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "socks connect") ||
		strings.Contains(text, "connection reset") ||
		strings.Contains(text, "connection refused") ||
		strings.Contains(text, "i/o timeout") ||
		strings.Contains(text, "timeout") ||
		strings.Contains(text, "temporary") ||
		strings.Contains(text, "unexpected eof")
}

func waitVideoCreateRetry(ctx context.Context, attempt int) error {
	if attempt < 1 {
		attempt = 1
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(attempt) * videoCreateRetryGap):
		return nil
	}
}

func parseVideoTaskCreateResult(body []byte) (videoTaskCreateResult, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return videoTaskCreateResult{}, fmt.Errorf("解析视频任务响应失败: %w", err)
	}
	taskID := firstNonEmptyString(
		jsonPathString(payload, "id"),
		jsonPathString(payload, "task_id"),
		jsonPathString(payload, "taskID"),
		jsonPathString(payload, "task.id"),
		jsonPathString(payload, "task.task_id"),
		jsonPathString(payload, "data.id"),
		jsonPathString(payload, "data.task_id"),
		jsonPathString(payload, "data.taskID"),
		jsonPathString(payload, "data.task.id"),
		jsonPathString(payload, "data.task.task_id"),
	)
	return videoTaskCreateResult{
		TaskID: taskID,
		Status: firstNonEmptyString(jsonPathString(payload, "status"), jsonPathString(payload, "data.status")),
	}, nil
}

func parseVideoTaskPollResult(body []byte) (videoTaskPollResult, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return videoTaskPollResult{}, fmt.Errorf("解析视频任务响应失败: %w", err)
	}
	status := firstNonEmptyString(
		jsonPathString(payload, "status"),
		jsonPathString(payload, "data.status"),
		jsonPathString(payload, "state"),
		jsonPathString(payload, "data.state"),
	)
	videoURL := firstNonEmptyString(
		jsonPathString(payload, "video.url"),
		jsonPathString(payload, "video.video_url"),
		jsonPathString(payload, "data.video.url"),
		jsonPathString(payload, "data.video.video_url"),
		jsonPathString(payload, "video_url"),
		jsonPathString(payload, "data.video_url"),
		jsonPathString(payload, "url"),
		jsonPathString(payload, "data.url"),
		jsonPathString(payload, "metadata.url"),
		jsonPathString(payload, "metadata.video_url"),
		jsonPathString(payload, "data.metadata.url"),
		jsonPathString(payload, "data.metadata.video_url"),
		jsonPathString(payload, "content.video_url"),
		jsonPathString(payload, "content.url"),
		jsonPathString(payload, "data.content.video_url"),
		jsonPathString(payload, "data.content.url"),
		jsonPathString(payload, "output.video_url"),
		jsonPathString(payload, "output.url"),
		jsonPathString(payload, "data.output.video_url"),
		jsonPathString(payload, "data.output.url"),
	)
	errorMessage := firstNonEmptyString(
		jsonPathString(payload, "error.message"),
		jsonPathString(payload, "data.error.message"),
		jsonPathString(payload, "error"),
		jsonPathString(payload, "message"),
		jsonPathString(payload, "data.message"),
	)
	return videoTaskPollResult{Status: status, VideoURL: videoURL, ErrorMessage: errorMessage}, nil
}

func jsonPathString(payload map[string]interface{}, path string) string {
	var current interface{} = payload
	for _, part := range strings.Split(path, ".") {
		item, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current, ok = item[part]
		if !ok {
			return ""
		}
	}
	switch value := current.(type) {
	case string:
		return strings.TrimSpace(value)
	case fmt.Stringer:
		return strings.TrimSpace(value.String())
	case float64, float32, int, int64, int32, uint, uint64, uint32, json.Number:
		return strings.TrimSpace(fmt.Sprint(value))
	default:
		return ""
	}
}

func isVideoTaskSuccessStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "succeeded", "success", "completed", "complete", "done":
		return true
	default:
		return false
	}
}

func isVideoTaskFailedStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "failure", "error", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

// downloadGeneratedVideo 下载生成的视频文件并返回字节数据。
func (s *Service) downloadGeneratedVideo(ctx context.Context, url string) ([]byte, string, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, "", ErrUpstreamEmptyResponse
	}
	cfg := s.cfg.Snapshot()
	client := newVideoGenerationHTTPClient(cfg.Env, cfg.SSRFProtectionEnabled, 300*time.Second)
	limit := s.cfg.Snapshot().MaxUploadFileBytes * 10 // 视频文件可能较大
	if limit <= 0 {
		limit = 200 * 1024 * 1024
	}

	var lastErr error
	var lastMimeType string
	for attempt := 1; attempt <= generatedVideoDownloadAttempts; attempt++ {
		data, mimeType, err := downloadGeneratedVideoOnce(ctx, client, url, limit)
		if err == nil {
			return data, mimeType, nil
		}
		lastErr = err
		lastMimeType = mimeType
		if !shouldRetryGeneratedVideoDownload(err) || attempt == generatedVideoDownloadAttempts {
			break
		}
		if s.logger != nil {
			s.logger.Warn("generated_video_download_retry",
				zap.Int("attempt", attempt),
				zap.Int("max_attempts", generatedVideoDownloadAttempts),
				zap.String("url_host", generatedVideoURLHost(url)),
				zap.Error(err),
			)
		}
		select {
		case <-ctx.Done():
			return nil, lastMimeType, ctx.Err()
		case <-time.After(time.Duration(attempt) * generatedVideoDownloadRetryGap):
		}
	}
	if lastErr == nil {
		lastErr = ErrUpstreamEmptyResponse
	}
	return nil, lastMimeType, lastErr
}

func newVideoGenerationHTTPClient(env string, ssrfProtectionEnabled bool, timeout time.Duration) *http.Client {
	transport := security.NewOutboundHTTPTransport(env, ssrfProtectionEnabled, 30*time.Second)
	if proxyURL := videoGenerationSocks5ProxyURL(); proxyURL != "" {
		if dialContext, err := socks5DialContext(proxyURL); err == nil {
			transport.Proxy = nil
			transport.DialContext = dialContext
		}
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

func videoGenerationSocks5ProxyURL() string {
	for _, key := range []string{"MEDIA_VIDEO_SOCKS5_PROXY", "VIDEO_GENERATION_SOCKS5_PROXY", "WARP_SOCKS5_PROXY"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func socks5DialContext(rawProxyURL string) (func(context.Context, string, string) (net.Conn, error), error) {
	proxyURL, err := url.Parse(strings.TrimSpace(rawProxyURL))
	if err != nil {
		return nil, err
	}
	scheme := strings.ToLower(strings.TrimSpace(proxyURL.Scheme))
	if scheme != "socks5" && scheme != "socks5h" {
		return nil, fmt.Errorf("unsupported SOCKS5 proxy scheme %q", proxyURL.Scheme)
	}
	address := strings.TrimSpace(proxyURL.Host)
	if address == "" {
		return nil, fmt.Errorf("empty SOCKS5 proxy host")
	}
	var auth *proxy.Auth
	if proxyURL.User != nil {
		auth = &proxy.Auth{User: proxyURL.User.Username()}
		auth.Password, _ = proxyURL.User.Password()
	}
	dialer, err := proxy.SOCKS5("tcp", address, auth, proxy.Direct)
	if err != nil {
		return nil, err
	}
	contextDialer, ok := dialer.(proxy.ContextDialer)
	if ok {
		return contextDialer.DialContext, nil
	}
	return func(ctx context.Context, network string, address string) (net.Conn, error) {
		result := make(chan struct {
			conn net.Conn
			err  error
		}, 1)
		go func() {
			conn, err := dialer.Dial(network, address)
			result <- struct {
				conn net.Conn
				err  error
			}{conn: conn, err: err}
		}()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case item := <-result:
			return item.conn, item.err
		}
	}, nil
}

func downloadGeneratedVideoOnce(ctx context.Context, client *http.Client, rawURL string, limit int64) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	mimeType := generatedVideoMimeType(resp.Header.Get("Content-Type"))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, mimeType, generatedVideoDownloadHTTPError{StatusCode: resp.StatusCode}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, mimeType, err
	}
	if int64(len(data)) > limit {
		return nil, mimeType, ErrFileTooLarge
	}
	return data, mimeType, nil
}

func generatedVideoMimeType(contentType string) string {
	mimeType := "video/mp4"
	if contentType = strings.TrimSpace(contentType); strings.HasPrefix(strings.ToLower(contentType), "video/") {
		mimeType = strings.Split(contentType, ";")[0]
	}
	return mimeType
}

type generatedVideoDownloadHTTPError struct {
	StatusCode int
}

func (e generatedVideoDownloadHTTPError) Error() string {
	return fmt.Sprintf("download generated video failed: HTTP %d", e.StatusCode)
}

func shouldRetryGeneratedVideoDownload(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrFileTooLarge) {
		return false
	}
	var httpErr generatedVideoDownloadHTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode == http.StatusTooManyRequests || httpErr.StatusCode >= 500
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "stream error") ||
		strings.Contains(text, "internal_error") ||
		strings.Contains(text, "connection reset") ||
		strings.Contains(text, "unexpected eof") ||
		strings.Contains(text, "reading body")
}

func generatedVideoURLHost(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}

// generatedVideoFileName 构造视频文件名。
func generatedVideoFileName(modelName string, capturedAt time.Time, mimeType string) string {
	base := sanitizeGeneratedImageFileBase(modelName)
	timestamp := fmt.Sprintf("%s-%03d", capturedAt.Format("20060102-150405"), capturedAt.Nanosecond()/int(time.Millisecond))
	ext := ".mp4"
	if strings.Contains(strings.ToLower(mimeType), "quicktime") || strings.Contains(strings.ToLower(mimeType), "mov") {
		ext = ".mov"
	}
	return fmt.Sprintf("%s-video-%s%s", base, timestamp, ext)
}
