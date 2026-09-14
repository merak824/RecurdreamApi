package repository

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestOpsErrorLogInsertDoesNotWriteLegacyRequestReplayFields(t *testing.T) {
	legacyColumns := []string{
		"request_body",
		"request_headers",
		"request_body_truncated",
		"request_body_bytes",
		"is_retryable",
		"retry_count",
		"resolved_retry_id",
	}

	insertSQL := strings.ToLower(insertOpsErrorLogSQL)
	for _, column := range legacyColumns {
		if strings.Contains(insertSQL, column) {
			t.Fatalf("ops error log insert still writes legacy replay column %q", column)
		}
	}

	inputType := reflect.TypeOf(service.OpsInsertErrorLogInput{})
	legacyFields := []string{
		"RequestBodyJSON",
		"RequestBodyTruncated",
		"RequestBodyBytes",
		"RequestHeadersJSON",
		"IsRetryable",
		"RetryCount",
		"ResolvedRetryID",
	}
	for _, field := range legacyFields {
		if _, ok := inputType.FieldByName(field); ok {
			t.Fatalf("OpsInsertErrorLogInput still carries legacy replay field %q", field)
		}
	}
}
