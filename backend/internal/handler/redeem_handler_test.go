package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRedeemHandlerGetBalanceHistoryUsesAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = client.Close() })
	mock.ExpectQuery(`(?s)WITH balance_history AS .*WHERE ledger.user_id = \$1`).
		WithArgs(int64(77), "", 10, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "type", "amount", "occurred_at", "reference", "description", "total_count",
		}))

	redeemService := service.NewRedeemService(nil, nil, nil, nil, nil, client, nil)
	handler := NewRedeemHandler(redeemService)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/balance-history?page=1&page_size=10", nil)
	c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 77})

	handler.GetBalanceHistory(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRedeemHandlerGetBalanceHistoryReturnsNormalizedPageSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = client.Close() })
	mock.ExpectQuery(`(?s)WITH balance_history AS .*WHERE ledger.user_id = \$1`).
		WithArgs(int64(77), "", 100, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "type", "amount", "occurred_at", "reference", "description", "total_count",
		}))

	redeemService := service.NewRedeemService(nil, nil, nil, nil, nil, client, nil)
	handler := NewRedeemHandler(redeemService)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/balance-history?page=1&page_size=1000", nil)
	c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 77})

	handler.GetBalanceHistory(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var body struct {
		Data struct {
			PageSize int `json:"page_size"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, 100, body.Data.PageSize)
	require.NoError(t, mock.ExpectationsWereMet())
}
