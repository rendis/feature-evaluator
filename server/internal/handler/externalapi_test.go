package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	domainexternalapi "github.com/rendis/feature-evaluator/internal/domain/externalapi"
	"github.com/rendis/feature-evaluator/internal/dto"
	"github.com/rendis/feature-evaluator/internal/external"
)

func TestExternalAPIHandlerExpressionProfile(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	handler := NewExternalAPIHandler(nil, nil)
	router := gin.New()
	router.GET("/external-apis/expression-profile", handler.ExpressionProfile)

	req := httptest.NewRequest(http.MethodGet, "/external-apis/expression-profile", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response dto.ExternalAPIExpressionProfileResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(response.Keywords) == 0 {
		t.Fatal("expected profile keywords")
	}
	if len(response.Symbols) == 0 {
		t.Fatal("expected profile symbols")
	}
	if len(response.Actions) == 0 {
		t.Fatal("expected profile actions")
	}

	foundLen := false
	foundLegacyLength := false
	foundVars := false
	for _, symbol := range response.Symbols {
		if symbol.Path == "vars" {
			foundVars = true
			break
		}
	}
	for _, action := range response.Actions {
		if strings.Contains(action.Template, "len({{path}})") {
			foundLen = true
		}
		if strings.Contains(action.Template, ".length") {
			foundLegacyLength = true
		}
	}

	if !foundLen {
		t.Fatal("expected len({{path}}) array action in profile")
	}
	if foundLegacyLength {
		t.Fatal("did not expect legacy .length action in profile")
	}
	if foundVars {
		t.Fatal("did not expect vars root symbol in backend profile")
	}
}

func TestExternalAPIHandlerValidateExpression(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	handler := NewExternalAPIHandler(nil, nil)
	router := gin.New()
	router.POST("/external-apis/expression/validate", handler.ValidateExpression)

	tests := []struct {
		name       string
		body       string
		wantValid  bool
		wantErrSub string
	}{
		{
			name:      "empty expression is allowed while editing",
			body:      `{"expression":""}`,
			wantValid: true,
		},
		{
			name:      "array len expression is valid",
			body:      `{"expression":"len(response.body.results) > 0"}`,
			wantValid: true,
		},
		{
			name:      "array index expression is valid",
			body:      `{"expression":"response.body.results[0].id == 1"}`,
			wantValid: true,
		},
		{
			name:      "vars expression is valid",
			body:      `{"expression":"vars.campus_code == \"north\" and response.body.allowed == true"}`,
			wantValid: true,
		},
		{
			name:       "invalid malformed expression returns error",
			body:       `{"expression":"response.body.results =="}`,
			wantValid:  false,
			wantErrSub: "invalid response condition",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodPost,
				"/external-apis/expression/validate",
				strings.NewReader(tt.body),
			)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
			}

			var response dto.ValidateExpressionResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if response.Valid != tt.wantValid {
				t.Fatalf("valid = %v, want %v", response.Valid, tt.wantValid)
			}
			if tt.wantErrSub != "" {
				if response.Error == nil || !strings.Contains(*response.Error, tt.wantErrSub) {
					t.Fatalf("error = %v, want substring %q", response.Error, tt.wantErrSub)
				}
			}
		})
	}
}

func TestExternalAPIHandlerTestIncludesEvaluationDetails(t *testing.T) { //nolint:funlen // test setup
	t.Parallel()

	gin.SetMode(gin.TestMode)

	svc := domainexternalapi.NewService(externalAPIRepositoryStub{}, externalAPISecretCipherStub{}, nil)
	executor := external.NewCaller(nil, nil, nil)
	executor.SetHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				Body:    io.NopCloser(strings.NewReader(`{"approved":true}`)),
				Request: req,
			}, nil
		}),
	})
	handler := NewExternalAPIHandler(svc, executor)
	router := gin.New()
	router.POST("/external-apis/test", handler.Test)

	req := httptest.NewRequest(
		http.MethodPost,
		"/external-apis/test",
		strings.NewReader(`{
			"key":"eligibility_check",
			"name":"Eligibility Check",
			"request":{
				"method":"POST",
				"urlTemplate":"https://example.com/check",
				"bodyTemplate":{"user_id":"{{user_id}}"}
			},
			"params":[
				{"name":"user_id","type":"string","required":true,"locations":["body"]}
			],
			"responseValidation":{
				"mode":"both",
				"http":{"mode":"any_2xx"},
				"body":{"expression":"response.body.approved == true"}
			},
			"paramValues":{"user_id":"123"}
		}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response dto.ExternalAPITestResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if response.Details == nil || response.Details.Evaluations == nil {
		t.Fatalf("details.evaluations = nil, want payload: %#v", response.Details)
	}
	if response.Details.Evaluations.Final.Mode != domainexternalapi.ValidationModeBoth {
		t.Fatalf(
			"details.evaluations.final.mode = %q, want %q",
			response.Details.Evaluations.Final.Mode,
			domainexternalapi.ValidationModeBoth,
		)
	}
	if !response.Details.Evaluations.HTTP.Applied || !response.Details.Evaluations.Expression.Applied {
		t.Fatalf("details.evaluations = %#v, want both checks applied", response.Details.Evaluations)
	}
	if response.Details.Evaluations.Expression.ResolvedExpression != "true == true" {
		t.Fatalf(
			"details.evaluations.expression.resolvedExpression = %q, want %q",
			response.Details.Evaluations.Expression.ResolvedExpression,
			"true == true",
		)
	}
	if response.Details.Request == nil || response.Details.Request.Method != http.MethodPost {
		t.Fatalf("details.request.method = %#v, want %q", response.Details.Request, http.MethodPost)
	}
}

type externalAPIRepositoryStub struct{}

func (externalAPIRepositoryStub) Create(_ context.Context, _ *domainexternalapi.ExternalAPI) error {
	return nil
}

func (externalAPIRepositoryStub) GetByKey(
	_ context.Context,
	_ string,
) (*domainexternalapi.ExternalAPI, error) {
	return nil, nil
}

func (externalAPIRepositoryStub) Update(
	_ context.Context,
	_ string,
	_ *domainexternalapi.ExternalAPI,
) error {
	return nil
}

func (externalAPIRepositoryStub) Delete(_ context.Context, _ string) error {
	return nil
}

func (externalAPIRepositoryStub) List(_ context.Context) ([]domainexternalapi.ExternalAPI, error) {
	return nil, nil
}

func (externalAPIRepositoryStub) CountRuleUsages(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

type externalAPISecretCipherStub struct{}

func (externalAPISecretCipherStub) EncryptMap(payload map[string]string, _ string) (string, error) {
	if len(payload) == 0 {
		return "", nil
	}
	return "ciphertext", nil
}

func (externalAPISecretCipherStub) DecryptMap(_ string, _ string) (map[string]string, error) {
	return map[string]string{}, nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
