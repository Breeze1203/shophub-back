package services

import (
	"LiteAdmin/config"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

// 微信的认证端
var wechatEndpoint = oauth2.Endpoint{
	AuthURL:  "https://open.weixin.qq.com/connect/qrconnect",
	TokenURL: "https://api.weixin.qq.com/sns/oauth2/access_token",
}

// 企业微信的认证端
var enWechatEndpoint = oauth2.Endpoint{
	AuthURL:  "https://open.work.weixin.qq.com/wwopen/sso/qrConnect",
	TokenURL: "https://qyapi.weixin.qq.com/cgi-bin/gettoken",
}

type OAuthService struct {
	providers map[string]*oauth2.Config
}

type OAuthUserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar"`
	Provider string `json:"provider"`
}

func NewOAuthService(config *config.AuthConfig) *OAuthService {
	service := &OAuthService{
		providers: make(map[string]*oauth2.Config),
	}

	// Google OAuth
	if config.OAuth.Google.ClientID != "" {
		service.providers["google"] = &oauth2.Config{
			ClientID:     config.OAuth.Google.ClientID,
			ClientSecret: config.OAuth.Google.ClientSecret,
			RedirectURL:  config.OAuth.Google.RedirectURL,
			Scopes:       config.OAuth.Google.Scopes,
			Endpoint:     google.Endpoint,
		}
	}

	// GitHub OAuth
	if config.OAuth.GitHub.ClientID != "" {
		service.providers["github"] = &oauth2.Config{
			ClientID:     config.OAuth.GitHub.ClientID,
			ClientSecret: config.OAuth.GitHub.ClientSecret,
			RedirectURL:  config.OAuth.GitHub.RedirectURL,
			Scopes:       config.OAuth.GitHub.Scopes,
			Endpoint:     github.Endpoint,
		}
	}

	// Facebook OAuth
	if config.OAuth.Facebook.ClientID != "" {
		service.providers["facebook"] = &oauth2.Config{
			ClientID:     config.OAuth.Facebook.ClientID,
			ClientSecret: config.OAuth.Facebook.ClientSecret,
			RedirectURL:  config.OAuth.Facebook.RedirectURL,
			Scopes:       config.OAuth.Facebook.Scopes,
			Endpoint:     facebook.Endpoint,
		}
	}
	// WeChat OAuth
	if config.OAuth.Wechat.ClientID != "" {
		service.providers["wechat"] = &oauth2.Config{
			ClientID:     config.OAuth.Wechat.ClientID,
			ClientSecret: config.OAuth.Wechat.ClientSecret,
			RedirectURL:  config.OAuth.Wechat.RedirectURL,
			Scopes:       config.OAuth.Wechat.Scopes,
			Endpoint:     wechatEndpoint,
		}
	}
	// Enterprise WeChat OAuth
	if config.OAuth.EnterpriseWeChat.ClientID != "" {
		service.providers["en_wechat"] = &oauth2.Config{
			ClientID:     config.OAuth.EnterpriseWeChat.ClientID,
			ClientSecret: config.OAuth.EnterpriseWeChat.ClientSecret,
			RedirectURL:  config.OAuth.EnterpriseWeChat.RedirectURL,
			Scopes:       config.OAuth.EnterpriseWeChat.Scopes,
			Endpoint:     enWechatEndpoint,
		}
	}
	// Custom OAuth providers
	for name, provider := range config.OAuth.Custom {
		service.providers[name] = &oauth2.Config{
			ClientID:     provider.ClientID,
			ClientSecret: provider.ClientSecret,
			RedirectURL:  provider.RedirectURL,
			Scopes:       provider.Scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  provider.AuthURL,
				TokenURL: provider.TokenURL,
			},
		}
	}

	return service
}

func (s *OAuthService) GetAuthURL(provider, state string) (string, error) {
	if provider == "en_wechat" {
		return s.GetEnWeChatAuthURL(state)
	}
	if provider == "wechat" {
		return s.GetWeChatAuthURL(state)
	}
	cfg, exists := s.providers[provider]
	if !exists {
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
	return cfg.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

func (s *OAuthService) GetWeChatAuthURL(state string) (string, error) {
	// 个人微信可以直接用 oauth2.Config（标准 OAuth2）
	cfg, ok := s.providers["wechat"]
	if !ok {
		return "", fmt.Errorf("wechat (personal) config not found")
	}
	// 舔加 #wechat_redirect
	authURL := cfg.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		// 公众号网页授权必须加这个参数，否则会 40163 code been used
		oauth2.SetAuthURLParam("response_type", "code"),
	)
	// 公众号网页授权必须在 URL 后面强制加 #wechat_redirect
	if !strings.Contains(authURL, "#wechat_redirect") {
		if strings.Contains(authURL, "?") {
			authURL += "&"
		} else {
			authURL += "?"
		}
		authURL += "#wechat_redirect"
	}
	return authURL, nil
}

// 获取企业微信认证的url
func (s *OAuthService) GetEnWeChatAuthURL(state string) (string, error) {
	cfg, ok := s.providers["en_wechat"]
	if !ok {
		return "", fmt.Errorf("wechat config not found")
	}
	// 手动拼接企业微信授权地址
	u, _ := url.Parse(cfg.Endpoint.AuthURL)
	q := u.Query()
	q.Set("appid", cfg.ClientID) // CorpID
	q.Set("agentid", "1000053")  // AgentID
	q.Set("redirect_uri", cfg.RedirectURL)
	q.Set("state", state)
	q.Set("response_type", "code")
	q.Set("scope", "snsapi_userinfo")
	q.Set("href", "")
	q.Set("#wechat_redirect", "")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (s *OAuthService) ExchangeCode(provider, code string) (*oauth2.Token, error) {
	// 非oauth2标准特殊处理
	if provider == "en_wechat" {
		return s.handleWeChatEnterpriseCallback(code)
	}
	if provider == "wechat" {
		return s.handleWeChatPersonalCallback(code)
	}
	// 其他 provider 走标准 oauth2
	cfg, exists := s.providers[provider]
	if !exists {
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
	return cfg.Exchange(context.Background(), code)
}

// handleWeChatPersonalCallback 个人微信公众号网页授权
func (s *OAuthService) handleWeChatPersonalCallback(code string) (*oauth2.Token, error) {
	cfg, ok := s.providers["wechat"]
	if !ok {
		return nil, fmt.Errorf("wechat (personal) config not found")
	}
	token, err := cfg.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("wechat exchange code failed: %w", err)
	}
	// 微信返回的 token 里会自带 openid，我们把它显式放进 Extra，方便后面统一取
	if openid, ok := token.Extra("openid").(string); ok && openid != "" {
		extra := map[string]interface{}{
			"openid":          openid,
			"scope":           token.Extra("scope"),
			"unionid":         token.Extra("unionid"), // 有可能有
			"provider":        "wechat",               // 标记是个人微信
			"personal_wechat": true,                   // 方便判断
		}
		token = token.WithExtra(extra)
	}
	return token, nil
}

// 获取企业微信的token
func (s *OAuthService) handleWeChatEnterpriseCallback(code string) (*oauth2.Token, error) {
	cfg, ok := s.providers["en_wechat"]
	if !ok {
		return nil, fmt.Errorf("enterprise wechat config not found")
	}
	accessToken, err := s.getWeChatEnterpriseAccessToken(cfg)
	if err != nil {
		return nil, fmt.Errorf("get enterprise access_token failed: %w", err)
	}
	// 用 code 换 userid
	userInfoURL := fmt.Sprintf(
		"https://qyapi.weixin.qq.com/cgi-bin/user/getuserinfo?access_token=%s&code=%s&agentid=%s",
		accessToken,
		code,
		1000053,
	)
	resp, err := http.Get(userInfoURL)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing resp body: %v", err)
		}
	}(resp.Body)
	body, _ := io.ReadAll(resp.Body)
	type Resp struct {
		ErrCode  int    `json:"errcode"`
		ErrMsg   string `json:"errmsg"`
		UserId   string `json:"UserId"` // 企业内唯一
		OpenId   string `json:"OpenId"` // 可能为空
		DeviceId string `json:"DeviceId"`
	}

	var r Resp
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("parse getuserinfo response failed: %w", err)
	}
	if r.ErrCode != 0 {
		return nil, fmt.Errorf("enterprise wechat getuserinfo error %d: %s", r.ErrCode, r.ErrMsg)
	}
	if r.UserId == "" {
		return nil, fmt.Errorf("enterprise wechat returned empty UserId")
	}
	// oauth2.Token
	token := &oauth2.Token{
		AccessToken: r.UserId,
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(2 * time.Hour),
	}
	extra := map[string]interface{}{
		"enterprise_access_token": accessToken,
		"agent_id":                1000053,
		"userid":                  r.UserId,
	}
	return token.WithExtra(extra), nil
}

func (s *OAuthService) getWeChatEnterpriseAccessToken(cfg *oauth2.Config) (string, error) {
	auth_url := fmt.Sprintf(
		"https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s",
		cfg.ClientID,
		cfg.ClientSecret,
	)
	resp, err := http.Get(auth_url)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing resp body: %v", err)
		}
	}(resp.Body)
	body, _ := io.ReadAll(resp.Body)
	type TokenResp struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
	}
	var tr TokenResp
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", err
	}
	if tr.ErrCode != 0 {
		return "", fmt.Errorf("gettoken error %d: %s", tr.ErrCode, tr.ErrMsg)
	}
	return tr.AccessToken, nil
}

// 获取用户信息
func (s *OAuthService) GetUserInfo(provider string, token *oauth2.Token) (*OAuthUserInfo, error) {
	switch provider {
	case "google":
		return s.getGoogleUserInfo(token)
	case "github":
		return s.getGitHubUserInfo(token)
	case "facebook":
		return s.getFacebookUserInfo(token)
	case "wechat":
		return s.getEnWeChatUserInfo(token)
	case "en_wechat":
		return s.getEnWeChatUserInfo(token)
	default:
		return s.getCustomUserInfo(provider, token)
	}
}

func (s *OAuthService) getGoogleUserInfo(token *oauth2.Token) (*OAuthUserInfo, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing resp body: %v", err)
		}
	}(resp.Body)

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &OAuthUserInfo{
		ID:       data["id"].(string),
		Email:    data["email"].(string),
		Name:     data["name"].(string),
		Avatar:   data["picture"].(string),
		Provider: "google",
	}, nil
}

func (s *OAuthService) getGitHubUserInfo(token *oauth2.Token) (*OAuthUserInfo, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing resp body: %v", err)
		}
	}(resp.Body)
	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	email := ""
	if data["email"] != nil {
		email = data["email"].(string)
	}

	return &OAuthUserInfo{
		ID:       fmt.Sprintf("%v", data["id"]),
		Email:    email,
		Name:     data["login"].(string),
		Avatar:   data["avatar_url"].(string),
		Provider: "github",
	}, nil
}

func (s *OAuthService) getFacebookUserInfo(token *oauth2.Token) (*OAuthUserInfo, error) {
	resp, err := http.Get("https://graph.facebook.com/me?fields=id,name,email,picture&access_token=" + token.AccessToken)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing resp body: %v", err)
		}
	}(resp.Body)
	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	picture := data["picture"].(map[string]interface{})["data"].(map[string]interface{})["url"].(string)

	return &OAuthUserInfo{
		ID:       data["id"].(string),
		Email:    data["email"].(string),
		Name:     data["name"].(string),
		Avatar:   picture,
		Provider: "facebook",
	}, nil
}

func (s *OAuthService) getCustomUserInfo(provider string, token *oauth2.Token) (*OAuthUserInfo, error) {
	// For custom providers, you'll need to specify the user info endpoint
	// This is a generic implementation
	return nil, fmt.Errorf("custom provider user info not implemented")
}

func (s *OAuthService) GetAvailableProviders() []string {
	providers := make([]string, 0, len(s.providers))
	for name := range s.providers {
		providers = append(providers, name)
	}
	return providers
}

// getEnWeChatUserInfo 获取企业微信用户详细信息 支持：企业微信（en_wechat） 和 个人微信公众号（wechat）两种
func (s *OAuthService) getEnWeChatUserInfo(token *oauth2.Token) (*OAuthUserInfo, error) {
	// 先判断是哪种微信（从 token.Extra 里拿标记最稳）
	providerInterface := token.Extra("provider")
	if providerInterface == nil {
		// 兼容旧数据：如果没有 provider 字段，看有没有 enterprise_access_token
		if token.Extra("enterprise_access_token") != nil {
			return s.getEnterpriseWeChatUserInfo(token)
		}
		// 默认走个人微信公众号流程
		return s.getPersonalWeChatUserInfo(token)
	}
	provider := providerInterface.(string)
	if provider == "en_wechat" {
		return s.getEnterpriseWeChatUserInfo(token)
	}
	// 默认走个人微信公众号
	return s.getPersonalWeChatUserInfo(token)
}

// 企业微信专用获取用户信息
func (s *OAuthService) getEnterpriseWeChatUserInfo(token *oauth2.Token) (*OAuthUserInfo, error) {
	entAccessToken, ok1 := token.Extra("enterprise_access_token").(string)
	agentID, ok2 := token.Extra("agent_id").(string)
	userID := token.AccessToken

	if !ok1 || entAccessToken == "" {
		return nil, fmt.Errorf("enterprise_access_token not found in token")
	}
	if !ok2 || agentID == "" {
		return nil, fmt.Errorf("agent_id not found in token")
	}
	if userID == "" {
		return nil, fmt.Errorf("userid not found in token")
	}
	// 调用企业微信获取用户详情接口
	user_url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/user/get?access_token=%s&userid=%s", entAccessToken, userID)

	resp, err := http.Get(user_url)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing resp body: %v", err)
		}
	}(resp.Body)
	body, _ := io.ReadAll(resp.Body)

	type UserResp struct {
		ErrCode    int    `json:"errcode"`
		ErrMsg     string `json:"errmsg"`
		UserId     string `json:"userid"`
		Name       string `json:"name"`
		Mobile     string `json:"mobile"`
		Email      string `json:"email"`
		Avatar     string `json:"avatar"`
		Department []int  `json:"department,omitempty"`
		Position   string `json:"position"`
		Gender     int    `json:"gender"`
	}

	var u UserResp
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, fmt.Errorf("parse enterprise wechat userinfo failed: %w", err)
	}

	if u.ErrCode != 0 {
		return nil, fmt.Errorf("enterprise wechat user/get error %d: %s", u.ErrCode, u.ErrMsg)
	}

	return &OAuthUserInfo{
		ID:       u.UserId,
		Name:     u.Name,
		Email:    u.Email,
		Avatar:   u.Avatar,
		Provider: "en_wechat", // 标记是企业微信
	}, nil
}

// 个人微信公众号
func (s *OAuthService) getPersonalWeChatUserInfo(token *oauth2.Token) (*OAuthUserInfo, error) {
	openid, ok := token.Extra("openid").(string)
	if !ok || openid == "" {
		// 有些老的 oauth2 实现会把 openid 放 AccessToken，兼容一下
		if openid == "" {
			openid = token.AccessToken
		}
		if openid == "" {
			return nil, fmt.Errorf("wechat openid not found in token")
		}
	}
	ueer_info := "https://api.weixin.qq.com/sns/userinfo"
	req, _ := http.NewRequest("GET", ueer_info, nil)
	q := req.URL.Query()
	q.Add("access_token", token.AccessToken)
	q.Add("openid", openid)
	q.Add("lang", "zh_CN")
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing resp body: %v", err)
		}
	}(resp.Body)
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wechat userinfo error: %s", string(body))
	}

	var data struct {
		OpenID     string `json:"openid"`
		Nickname   string `json:"nickname"`
		HeadImgURL string `json:"headimgurl"`
		UnionID    string `json:"unionid,omitempty"`
		ErrCode    int    `json:"errcode,omitempty"`
		ErrMsg     string `json:"errmsg,omitempty"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	if data.ErrCode != 0 {
		return nil, fmt.Errorf("wechat api error %d: %s", data.ErrCode, data.ErrMsg)
	}

	return &OAuthUserInfo{
		ID:       data.OpenID,
		Name:     data.Nickname,
		Avatar:   data.HeadImgURL,
		Provider: "wechat",
	}, nil
}
