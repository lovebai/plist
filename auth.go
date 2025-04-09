package main

import (
	"html/template"
	"log"
	"net/http"
	"strings"
)

// 认证中间件
func AuthMiddleware(next http.Handler) http.Handler {
	if config.Secure != "false" {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("auth")
			// log.Printf("请求路径: %s, Cookie状态: %+v, 错误信息: %v", r.URL.Path, cookie, err)

			if err != nil || !verifyCookie(cookie) {
				// log.Printf("验证失败，跳转登录页面")
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	return next
}

// 验证cookie有效性
func verifyCookie(cookie *http.Cookie) bool {
	return cookie != nil && strings.Contains(cookie.Value, "authenticated")
	// return cookie != nil && cookie.Value == "authenticated"
}

// 登录处理器
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// 验证密码
		log.Printf("输入密码：%s，正确密码：%s", r.FormValue("password"), config.Password)
		if r.FormValue("password") == config.Password {
			// 设置认证cookie（1小时有效期）
			http.SetCookie(w, &http.Cookie{
				Name:     "auth",
				Value:    "authenticated",
				MaxAge:   3600, // 使用秒数设置有效期（1小时）
				HttpOnly: true,
				Path:     "/",
				Secure:   true,                 // 开发环境可设为false，生产环境必须设为true
				SameSite: http.SameSiteLaxMode, // 添加SameSite属性
			})
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		http.Error(w, "密码错误", http.StatusUnauthorized)
		return
	}

	// 显示登录表单
	tmpl := template.Must(template.New("login").Parse(loginTemplate))
	tmpl.Execute(w, config)
}
