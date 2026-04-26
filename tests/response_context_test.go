package tests

import (
	"net/http/httptest"
	"testing"

	"ad-platform/pkg/response"

	"github.com/gin-gonic/gin"
)

func TestErrorWritesResponseCodeToContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	response.Error(c, response.CodeForbidden, response.MsgInsufficientPermissions)

	gotCode, ok := c.Get(response.ContextResponseCodeKey)
	if !ok {
		t.Fatalf("expected response code in context")
	}
	if gotCode != response.CodeForbidden {
		t.Fatalf("unexpected response code in context: %v", gotCode)
	}
	if _, ok := c.Get(response.ContextResponseMessageKey); !ok {
		t.Fatalf("expected response message in context")
	}
	if gotCategory, ok := c.Get(response.ContextErrorCategoryKey); !ok || gotCategory != "permission" {
		t.Fatalf("expected error category permission, got: %v", gotCategory)
	}
	if gotReason, ok := c.Get(response.ContextErrorReasonKey); !ok || gotReason != response.MsgInsufficientPermissions {
		t.Fatalf("expected error reason in context")
	}
}

func TestSuccessWritesResponseCodeToContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	response.Success(c, response.MsgSuccess)

	gotCode, ok := c.Get(response.ContextResponseCodeKey)
	if !ok {
		t.Fatalf("expected response code in context")
	}
	if gotCode != response.CodeSuccess {
		t.Fatalf("unexpected response code in context: %v", gotCode)
	}
	if _, ok := c.Get(response.ContextErrorCategoryKey); ok {
		t.Fatalf("did not expect error category for success response")
	}
}

func TestErrorUsesPresetReasonAndCategory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	response.SetErrorReason(c, response.ReasonRateLimited)
	response.SetErrorCategory(c, "throttle")
	response.Error(c, response.CodeTooManyRequests, response.MsgTooManyRequestsRetryLater)

	gotReason, ok := c.Get(response.ContextErrorReasonKey)
	if !ok || gotReason != response.ReasonRateLimited {
		t.Fatalf("expected preset reason %q, got %v", response.ReasonRateLimited, gotReason)
	}
	gotCategory, ok := c.Get(response.ContextErrorCategoryKey)
	if !ok || gotCategory != "throttle" {
		t.Fatalf("expected preset category throttle, got %v", gotCategory)
	}
}
