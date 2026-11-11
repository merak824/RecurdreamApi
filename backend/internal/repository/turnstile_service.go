package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

type turnstileVerifier struct {
	httpClient *http.Client
	verifyURL  string
}

func NewTurnstileVerifier() service.TurnstileVerifier {
	sharedClient, err := httpclient.GetClient(httpclient.Options{
		Timeout:            10 * time.Second,
		ValidateResolvedIP: true,
	})
	if err != nil {
		sharedClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &turnstileVerifier{
		httpClient: sharedClient,
		verifyURL:  turnstileVerifyURL,
	}
}

func (v *turnstileVerifier) VerifyToken(ctx context.Context, secretKey, token, remoteIP string) (*service.TurnstileVerifyResponse, error) {
	// 说明：不向 Cloudflare 传 remoteip。该参数要求"解验证码的 IP"与"提交登录的 IP"
	// 严格一致，在客户端使用代理/VPN（出口按连接轮换）或站点套 CDN 时必然误伤，
	// 会导致合法用户登录被 CF 以 invalid-input-secret/invalid-input-response 拒绝。
	// token 本身一次性且绑定 sitekey，去掉 remoteip 不降低校验强度。
	formData := url.Values{}
	formData.Set("secret", secretKey)
	formData.Set("response", token)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.verifyURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result service.TurnstileVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}
