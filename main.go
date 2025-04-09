package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	ImageDir string
	Password string
	Port     string
	Title    string
	Icon     string
	Dynamic  string
}

var config = Config{
	ImageDir: "./images",
	Password: "",
	Port:     "8008",
	Title:    "在线图集",
	Icon:     "https://i.obai.cc/favicon.ico",
	Dynamic:  "false",
}

var categoryCache []Category

type Category struct {
	Name        string
	EncodedName string
	CoverImage  string
}

var imageExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
}

func scanCategories(imageDir string) []Category {
	categories, err := os.ReadDir(imageDir)
	if err != nil {
		log.Fatalf("无法读取目录 %s: %v", imageDir, err)
	}

	var categoryList []Category
	for _, category := range categories {
		if category.IsDir() {
			dirPath := filepath.Join(imageDir, category.Name())
			entries, err := os.ReadDir(dirPath)
			if err != nil {
				log.Printf("无法读取目录 %s: %v", dirPath, err)
				continue
			}

			var coverImage string
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				ext := strings.ToLower(filepath.Ext(entry.Name()))
				if imageExtensions[ext] {
					coverImage = entry.Name()
					break
				}
			}
			if coverImage != "" {
				categoryList = append(categoryList, Category{
					Name:        category.Name(),
					EncodedName: url.PathEscape(category.Name()),
					CoverImage:  coverImage,
				})
			}
		}
	}
	return categoryList
}

func main() {
	envVars := map[string]*string{
		"SITE_DIR":      &config.ImageDir,
		"SITE_PASSWORD": &config.Password,
		"SITE_PORT":     &config.Port,
		"SITE_TITLE":    &config.Title,
		"SITE_ICON":     &config.Icon,
		"SITE_DYNAMIC":  &config.Dynamic,
	}

	for env, conf := range envVars {
		if val := os.Getenv(env); val != "" {
			*conf = val
		}
	}

	categoryCache = scanCategories(config.ImageDir)

	// 路由设置
	http.HandleFunc("/login", loginHandler)
	if config.Dynamic == "true" {
		http.Handle("/api/index/", AuthMiddleware(http.HandlerFunc(indexJson)))
		http.Handle("/api/category/", AuthMiddleware(http.HandlerFunc(categoryJson)))
	}
	http.Handle("/", AuthMiddleware(http.HandlerFunc(indexHandler)))
	http.Handle("/category/", AuthMiddleware(http.HandlerFunc(categoryHandler)))
	http.Handle("/images/", AuthMiddleware(http.StripPrefix("/images/", http.FileServer(http.Dir(config.ImageDir)))))
	log.Println("服务器启动在 :", config.Port)
	log.Fatal(http.ListenAndServe(":"+config.Port, nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if config.Dynamic == "true" {
		tmpl := template.Must(template.New("index").Parse(indexDynamicTemplate))
		tmpl.Execute(w, config)
	} else {
		type Tmp struct {
			Category []Category
			Config   Config
		}
		var tmp = Tmp{
			Category: categoryCache, // 使用缓存数据
			Config:   config,
		}
		tmpl := template.Must(template.New("index").Parse(indexTemplate))
		tmpl.Execute(w, tmp)
	}

}

func categoryHandler(w http.ResponseWriter, r *http.Request) {
	// category := r.URL.Path[len("/category/"):]
	// category := filepath.FromSlash(r.URL.Path[len("/category/"):])
	encodedCategory := filepath.FromSlash(r.URL.Path[len("/category/"):])
	category, _ := url.PathUnescape(encodedCategory)
	imagePath := filepath.Join(config.ImageDir, category)
	cleanImageDir := filepath.Clean(config.ImageDir)
	absImageDir, _ := filepath.Abs(cleanImageDir)
	absPath, _ := filepath.Abs(imagePath)
	if !strings.HasPrefix(absPath, absImageDir) {
		http.Error(w, "无效路径", http.StatusBadRequest)
		return
	}
	entries, err := os.ReadDir(imagePath)
	if err != nil {
		http.Error(w, "无法读取图片目录", http.StatusInternalServerError)
		return
	}

	type Image struct {
		Name string
		Type string
	}

	var imageList []Image
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if imageExtensions[ext] {
			imageList = append(imageList, Image{
				Name: entry.Name(),
				// 添加类型字段供模板使用（可选）
				Type: strings.TrimPrefix(ext, "."),
			})
		}
	}

	data := struct {
		Category string
		Images   []Image
		Config   Config
	}{
		Category: category,
		Images:   imageList,
		Config:   config,
	}

	if config.Dynamic == "true" {
		tmpl := template.Must(template.New("category").Parse(categoryDynamicTemplate))
		tmpl.Execute(w, data)
	} else {
		tmpl := template.Must(template.New("category").Parse(categoryTemplate))
		tmpl.Execute(w, data)

	}
}

func indexJson(w http.ResponseWriter, r *http.Request) {
	// 获取分页参数
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}

	// 使用缓存的分类信息
	totalCategories := len(categoryCache)
	totalPages := (totalCategories + limit - 1) / limit
	start := (page - 1) * limit
	end := start + limit
	if start > totalCategories {
		start = totalCategories
	}
	if end > totalCategories {
		end = totalCategories
	}
	currentCategories := categoryCache[start:end]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"categories": currentCategories,
		"page":       page,
		"limit":      limit,
		"total":      totalCategories,
		"pages":      totalPages,
	})
}

func categoryJson(w http.ResponseWriter, r *http.Request) {
	encodedCategory := filepath.FromSlash(r.URL.Path[len("/api/category/"):])
	category, _ := url.PathUnescape(encodedCategory)
	imagePath := filepath.Join(config.ImageDir, category)
	cleanImageDir := filepath.Clean(config.ImageDir)
	absImageDir, _ := filepath.Abs(cleanImageDir)
	absPath, _ := filepath.Abs(imagePath)
	if !strings.HasPrefix(absPath, absImageDir) {
		http.Error(w, "无效路径", http.StatusBadRequest)
		return
	}

	// 获取分页参数
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}

	entries, err := os.ReadDir(imagePath)
	if err != nil {
		http.Error(w, "无法读取图片目录", http.StatusInternalServerError)
		return
	}

	type Image struct {
		Name string
		Type string
	}

	var imageList []Image
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if imageExtensions[ext] {
			imageList = append(imageList, Image{
				Name: entry.Name(),
				Type: strings.TrimPrefix(ext, "."),
			})
		}
	}

	totalImages := len(imageList)
	totalPages := (totalImages + limit - 1) / limit
	start := (page - 1) * limit
	end := start + limit
	if start > totalImages {
		start = totalImages
	}
	if end > totalImages {
		end = totalImages
	}
	currentImages := imageList[start:end]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"category": category,
		"images":   currentImages,
		"page":     page,
		"limit":    limit,
		"total":    totalImages,
		"pages":    totalPages,
	})
}

// 认证中间件
func AuthMiddleware(next http.Handler) http.Handler {
	if config.Password != "" {
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
	return cookie != nil && cookie.Value == "authenticated"
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
				Secure:   false,                // 开发环境可设为false，生产环境必须设为true
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
	tmpl.Execute(w, nil)
}
