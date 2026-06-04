package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/adeithe/go-twitch/api"
)

const (
	oauthValidateURL = "https://id.twitch.tv/oauth2/validate"
	helixBaseURL     = "https://api.twitch.tv/helix"
)

type OAuthValidation struct {
	ClientID  string   `json:"client_id"`
	Login     string   `json:"login"`
	Scopes    []string `json:"scopes"`
	UserID    string   `json:"user_id"`
	ExpiresIn int      `json:"expires_in"`
}

type resolvedAccount struct {
	Account
	Validation OAuthValidation
}

func validateAccount(ctx context.Context, acc Account) (resolvedAccount, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, oauthValidateURL, nil)
	if err != nil {
		return resolvedAccount{}, err
	}
	req.Header.Set("Authorization", "OAuth "+sanitizeToken(acc.Token))

	httpClient, err := accountHTTPClient(acc)
	if err != nil {
		return resolvedAccount{}, err
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return resolvedAccount{}, err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		var body map[string]any
		_ = json.NewDecoder(res.Body).Decode(&body)
		if msg, ok := body["message"].(string); ok && msg != "" {
			return resolvedAccount{}, fmt.Errorf("oauth validate failed for %s: %s", acc.Name, msg)
		}
		return resolvedAccount{}, fmt.Errorf("oauth validate failed for %s: %s", acc.Name, res.Status)
	}

	var validation OAuthValidation
	if err := json.NewDecoder(res.Body).Decode(&validation); err != nil {
		return resolvedAccount{}, err
	}

	acc.Token = sanitizeToken(acc.Token)
	if acc.Login == "" {
		acc.Login = normalizeLogin(validation.Login)
	}
	if acc.UserID == "" {
		acc.UserID = validation.UserID
	}
	if acc.ClientID == "" {
		acc.ClientID = validation.ClientID
	}
	if acc.Login == "" || acc.UserID == "" || acc.ClientID == "" {
		return resolvedAccount{}, fmt.Errorf("oauth validate returned incomplete account data for %s", acc.Name)
	}

	return resolvedAccount{Account: acc, Validation: validation}, nil
}

func newAPIClient(acc resolvedAccount) *api.Client {
	httpClient, err := accountHTTPClient(acc.Account)
	if err != nil {
		return api.New(acc.ClientID, api.WithDefaultBearerToken(acc.Token))
	}
	return api.New(acc.ClientID, api.WithDefaultBearerToken(acc.Token), api.WithHTTPClient(httpClient))
}

func sendChatMessage(ctx context.Context, acc resolvedAccount, channel string, message string) error {
	channel = normalizeLogin(channel)
	if channel == "" {
		return errors.New("channel is required")
	}
	if strings.TrimSpace(message) == "" {
		return errors.New("message is required")
	}

	conn, err := newTwitchIRCConn(acc.Account)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.Connect(ctx); err != nil {
		return err
	}
	if err := conn.Join(channel); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(750 * time.Millisecond):
	}

	return conn.Say(channel, message)
}

func resolveUserID(ctx context.Context, acc resolvedAccount, login string) (string, error) {
	login = normalizeLogin(login)
	if login == "" {
		return "", errors.New("login is required")
	}
	if strings.EqualFold(login, acc.Login) {
		return acc.UserID, nil
	}

	params := url.Values{}
	params.Set("login", login)
	var decoded helixData[api.User]
	statusCode, err := helixGET(ctx, acc, "/users", params, &decoded)
	if err != nil {
		return "", err
	}
	if statusCode == http.StatusNotFound || len(decoded.Data) == 0 {
		return "", fmt.Errorf("twitch user %q not found", login)
	}
	return decoded.Data[0].UserID, nil
}

func listEmotes(ctx context.Context, acc resolvedAccount, kind, channel string) ([]api.Emote, error) {
	params := url.Values{}
	path := ""
	switch kind {
	case "global":
		path = "/chat/emotes/global"
	case "channel":
		broadcasterID, err := resolveUserID(ctx, acc, channel)
		if err != nil {
			return []api.Emote{}, nil
		}
		path = "/chat/emotes"
		params.Set("broadcaster_id", broadcasterID)
	case "user":
		path = "/chat/emotes/user"
		params.Set("user_id", acc.UserID)
		if strings.TrimSpace(channel) != "" {
			broadcasterID, err := resolveUserID(ctx, acc, channel)
			if err != nil {
				return []api.Emote{}, nil
			}
			params.Set("broadcaster_id", broadcasterID)
		}
	default:
		return nil, fmt.Errorf("unknown emote kind %q", kind)
	}

	var decoded helixData[api.Emote]
	statusCode, err := helixGET(ctx, acc, path, params, &decoded)
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusNotFound {
		return []api.Emote{}, nil
	}
	return decoded.Data, nil
}

type helixData[T any] struct {
	Data []T `json:"data"`
}

func listCustomRewards(ctx context.Context, acc resolvedAccount, broadcasterLogin string, onlyManageable bool) ([]api.CustomReward, error) {
	broadcasterLogin = normalizeLogin(broadcasterLogin)
	if broadcasterLogin == "" {
		broadcasterLogin = acc.Login
	}
	broadcasterID, err := resolveUserID(ctx, acc, broadcasterLogin)
	if err != nil {
		return []api.CustomReward{}, nil
	}

	params := url.Values{}
	params.Set("broadcaster_id", broadcasterID)
	if onlyManageable {
		params.Set("only_manageable_rewards", "true")
	}

	var decoded helixData[api.CustomReward]
	statusCode, err := helixGET(ctx, acc, "/channel_points/custom_rewards", params, &decoded)
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusNotFound {
		return []api.CustomReward{}, nil
	}
	return decoded.Data, nil
}

func helixGET(ctx context.Context, acc resolvedAccount, path string, params url.Values, out any) (int, error) {
	reqURL := helixBaseURL + "/" + strings.TrimPrefix(path, "/")
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Client-ID", acc.ClientID)
	req.Header.Set("Authorization", "Bearer "+acc.Token)

	httpClient, err := accountHTTPClient(acc.Account)
	if err != nil {
		return 0, err
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode == http.StatusNotFound {
		return res.StatusCode, nil
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		var body map[string]any
		_ = json.NewDecoder(res.Body).Decode(&body)
		if msg, ok := body["message"].(string); ok && msg != "" {
			return res.StatusCode, fmt.Errorf("twitch api request failed: %s", msg)
		}
		return res.StatusCode, fmt.Errorf("twitch api request failed: %s", res.Status)
	}

	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		return res.StatusCode, err
	}
	return res.StatusCode, nil
}

func hasScope(scopes []string, scope string) bool {
	for _, item := range scopes {
		if item == scope {
			return true
		}
	}
	return false
}
