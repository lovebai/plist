package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/gorilla/sessions"
)

const (
	AuthorizationEndpoint = "https://connect.linux.do/oauth2/authorize"
	TokenEndpoint         = "https://connect.linux.do/oauth2/token"
	UserEndpoint          = "https://connect.linux.do/api/user"
)

var store = sessions.NewCookieStore([]byte("secret-key"))

type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
}

// 生成随机密钥
func generateRandomKey(length int) []byte {
	key := make([]byte, length)
	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}
	return key
}

// 发起授权
func initiateAuthHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")

	// 生成随机的 state
	stateBytes := generateRandomKey(16)
	state := hex.EncodeToString(stateBytes)
	session.Values["oauth_state"] = state
	session.Save(r, w)

	// 构造授权 URL
	authURL := fmt.Sprintf("%s?client_id=%s&response_type=code&redirect_uri=%s&state=%s",
		AuthorizationEndpoint,
		config.ClientId,
		config.Adderss+"/oauth2/callback",
		state,
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// 处理回调
func callbackHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")

	// 获取查询参数
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	// 验证 state
	storedState := session.Values["oauth_state"]
	if storedState == nil || state != storedState.(string) {
		http.Error(w, "State value does not match", http.StatusUnauthorized)
		return
	}

	// 创建 HTTP 客户端
	client := resty.New()

	// 请求 access token
	resp, err := client.R().
		SetBasicAuth(config.ClientId, config.ClientSecret).
		SetHeader("Accept", "application/json").
		SetFormData(map[string]string{
			"grant_type":   "authorization_code",
			"code":         code,
			"redirect_uri": config.Adderss + "/oauth2/callback",
		}).
		Post(TokenEndpoint)

	if err != nil || resp.StatusCode() != http.StatusOK {
		http.Error(w, "Failed to fetch access token", http.StatusInternalServerError)
		return
	}

	// 解析 token 响应
	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(resp.Body(), &tokenResp); err != nil {
		http.Error(w, "Failed to parse token response", http.StatusInternalServerError)
		return
	}

	// 获取用户信息
	userResp, err := client.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", tokenResp.AccessToken)).
		Get(UserEndpoint)

	if err != nil || userResp.StatusCode() != http.StatusOK {
		http.Error(w, "Failed to fetch user info", http.StatusInternalServerError)
		return
	}

	// 解析用户信息
	var user User
	if err := json.Unmarshal(userResp.Body(), &user); err != nil {
		http.Error(w, "Failed to parse user response", http.StatusInternalServerError)
		return
	}

	session.Values["username"] = user.Username
	session.Values["avatar"] = user.AvatarURL
	session.Save(r, w)

	http.SetCookie(w, &http.Cookie{
		Name:     "auth",
		Value:    "linuxdo-authenticated:" + user.Username,
		MaxAge:   3600, // 使用秒数设置有效期（1小时）
		HttpOnly: true,
		Path:     "/",
		Secure:   true,                 // 开发环境可设为false，生产环境必须设为true
		SameSite: http.SameSiteLaxMode, // 添加SameSite属性
	})
	http.Redirect(w, r, "/", http.StatusFound)
}
