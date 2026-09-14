package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupUpstreamBillingProbeRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := NewAccountHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	handler.SetUpstreamBillingProbeService(service.NewUpstreamBillingProbeService(nil, nil, nil))

	router := gin.New()
	router.GET("/admin/accounts/upstream-billing-probe/settings", handler.GetUpstreamBillingProbeSettings)
	router.POST("/admin/accounts/upstream-billing-probe/batch", handler.ProbeUpstreamBillingBatch)
	router.PUT("/admin/accounts/:id/upstream-billing-probe", handler.SetUpstreamBillingProbeEnabled)
	return router
}

func TestAccountHandlerGetUpstreamBillingProbeSettingsReturnsDefaults(t *testing.T) {
	router := setupUpstreamBillingProbeRouter()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/admin/accounts/upstream-billing-probe/settings", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Data service.UpstreamBillingProbeSettings `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Data.Enabled)
	require.Equal(t, 30, response.Data.IntervalMinutes)
}

func TestAccountHandlerProbeUpstreamBillingBatchValidatesIDs(t *testing.T) {
	router := setupUpstreamBillingProbeRouter()

	for _, body := range []string{`{"account_ids":[]}`, `{"account_ids":[0]}`} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/admin/accounts/upstream-billing-probe/batch", bytes.NewBufferString(body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
	}
}

func TestAccountHandlerSetUpstreamBillingProbeEnabledRejectsInvalidID(t *testing.T) {
	router := setupUpstreamBillingProbeRouter()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/admin/accounts/not-an-id/upstream-billing-probe", bytes.NewBufferString(`{"enabled":true}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestAccountHandlerSetUpstreamBillingProbeEnabledRequiresValue(t *testing.T) {
	router := setupUpstreamBillingProbeRouter()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/admin/accounts/1/upstream-billing-probe", bytes.NewBufferString(`{}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

type createAccountWithProbeAdminService struct {
	*stubAdminService
	createCalls atomic.Int32
}

func (s *createAccountWithProbeAdminService) CreateAccount(_ context.Context, input *service.CreateAccountInput) (*service.Account, error) {
	s.createCalls.Add(1)
	extra := map[string]any{}
	if input.ProbeEnabled != nil {
		extra[service.UpstreamBillingProbeEnabledExtraKey] = *input.ProbeEnabled
	}
	return &service.Account{
		ID:       300,
		Name:     input.Name,
		Platform: input.Platform,
		Type:     input.Type,
		Status:   service.StatusActive,
		Extra:    extra,
	}, nil
}

type immediateBillingProbeStub struct {
	calls atomic.Int32
	ids   chan int64
}

func (s *immediateBillingProbeStub) ProbeAccount(_ context.Context, accountID int64) (*service.UpstreamBillingProbeSnapshot, error) {
	s.calls.Add(1)
	s.ids <- accountID
	return &service.UpstreamBillingProbeSnapshot{Status: service.UpstreamBillingProbeStatusOK}, nil
}

func TestCreateSchedulesEnabledUpstreamBillingProbeOnceAcrossIdempotentReplay(t *testing.T) {
	previousCoordinator := service.DefaultIdempotencyCoordinator()
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(newMemoryIdempotencyRepoStub(), service.DefaultIdempotencyConfig()))
	t.Cleanup(func() { service.SetDefaultIdempotencyCoordinator(previousCoordinator) })

	adminService := &createAccountWithProbeAdminService{stubAdminService: newStubAdminService()}
	probe := &immediateBillingProbeStub{ids: make(chan int64, 2)}
	handler := NewAccountHandler(adminService, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	handler.upstreamBillingProbeAccount = probe.ProbeAccount

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/accounts", handler.Create)
	body := []byte(`{"name":"relay","platform":"openai","type":"apikey","credentials":{"api_key":"test","base_url":"https://relay.example"},"upstream_billing_probe_enabled":true}`)
	request := func() *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "create-relay-once")
		router.ServeHTTP(recorder, req)
		return recorder
	}

	first := request()
	require.Equal(t, http.StatusOK, first.Code)
	select {
	case accountID := <-probe.ids:
		require.Equal(t, int64(300), accountID)
	case <-time.After(time.Second):
		t.Fatal("immediate upstream billing probe was not scheduled")
	}

	second := request()
	require.Equal(t, http.StatusOK, second.Code)
	require.Equal(t, "true", second.Header().Get("X-Idempotency-Replayed"))
	select {
	case accountID := <-probe.ids:
		t.Fatalf("idempotent replay scheduled another probe for account %d", accountID)
	case <-time.After(100 * time.Millisecond):
	}
	require.Equal(t, int32(1), adminService.createCalls.Load())
	require.Equal(t, int32(1), probe.calls.Load())
}
