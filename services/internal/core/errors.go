package core

import (
	"context"
	"errors"
	"os/exec"
	"strings"
)

type ErrorCode string

const (
	CodeYouTubeBotChallenge ErrorCode = "YOUTUBE_BOT_CHALLENGE"
	CodeYouTubePOToken      ErrorCode = "YOUTUBE_PO_TOKEN_REQUIRED"
	CodeYouTubeRateLimited  ErrorCode = "YOUTUBE_RATE_LIMITED"
	CodeYouTubeUnavailable  ErrorCode = "YOUTUBE_UNAVAILABLE"
	CodeUpstreamTimeout     ErrorCode = "UPSTREAM_TIMEOUT"
	CodeExtractorFailed     ErrorCode = "EXTRACTOR_FAILED"
)

type RuntimeError struct {
	Code      ErrorCode
	Message   string
	Retryable bool
	cause     error
}

func (e *RuntimeError) Error() string { return e.Message }
func (e *RuntimeError) Unwrap() error { return e.cause }

func RuntimeErrorDetails(err error) (ErrorCode, string, bool) {
	var runtimeErr *RuntimeError
	if errors.As(err, &runtimeErr) {
		return runtimeErr.Code, runtimeErr.Message, runtimeErr.Retryable
	}
	return CodeExtractorFailed, "Não foi possível concluir o processamento do conteúdo.", false
}

func classifyYTDLPError(err error, output string) *RuntimeError {
	if errors.Is(err, context.Canceled) {
		return &RuntimeError{Code: CodeUpstreamTimeout, Message: "A solicitação foi cancelada.", cause: err}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &RuntimeError{Code: CodeUpstreamTimeout, Message: "O serviço de origem demorou demais para responder.", Retryable: true, cause: err}
	}
	var exitErr *exec.ExitError
	if output == "" && errors.As(err, &exitErr) {
		output = string(exitErr.Stderr)
	}
	normalized := strings.ToLower(output)
	switch {
	case containsAny(normalized, "sign in to confirm you’re not a bot", "sign in to confirm you're not a bot", "confirm you're not a bot", "bot challenge"):
		return &RuntimeError{Code: CodeYouTubeBotChallenge, Message: "O YouTube solicitou uma verificação antibot. Tente mais tarde ou use o processamento local.", cause: err}
	case strings.Contains(normalized, "po token") && containsAny(normalized, "required", "missing", "not provided", "without"):
		return &RuntimeError{Code: CodeYouTubePOToken, Message: "O YouTube exigiu uma prova de origem que este servidor não possui.", cause: err}
	case containsAny(normalized, "http error 429", "too many requests", "status code 429"):
		return &RuntimeError{Code: CodeYouTubeRateLimited, Message: "O servidor online foi temporariamente limitado pelo YouTube. Tente mais tarde ou use o processamento local.", Retryable: true, cause: err}
	case containsAny(normalized, "timed out", "timeout", "deadline exceeded"):
		return &RuntimeError{Code: CodeUpstreamTimeout, Message: "O serviço de origem demorou demais para responder.", Retryable: true, cause: err}
	case containsAny(normalized, "video unavailable", "this video is unavailable", "private video", "members-only", "not available in your country"):
		return &RuntimeError{Code: CodeYouTubeUnavailable, Message: "Este conteúdo não está disponível para processamento.", cause: err}
	default:
		return &RuntimeError{Code: CodeExtractorFailed, Message: "O extrator não conseguiu processar este conteúdo.", cause: err}
	}
}

func containsAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}
