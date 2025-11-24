package services

import (
	"LiteAdmin/config"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"net/http"
)

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
	config, exists := s.providers[provider]
	if !exists {
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
	return config.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

func (s *OAuthService) ExchangeCode(provider, code string) (*oauth2.Token, error) {
	config, exists := s.providers[provider]
	if !exists {
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
	return config.Exchange(context.Background(), code)
}

func (s *OAuthService) GetUserInfo(provider string, token *oauth2.Token) (*OAuthUserInfo, error) {
	switch provider {
	case "google":
		return s.getGoogleUserInfo(token)
	case "github":
		return s.getGitHubUserInfo(token)
	case "facebook":
		return s.getFacebookUserInfo(token)
	default:
		return s.getCustomUserInfo(provider, token)
	}
}

func (s *OAuthService) getGoogleUserInfo(token *oauth2.Token) (*OAuthUserInfo, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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
	defer resp.Body.Close()
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
	defer resp.Body.Close()

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
