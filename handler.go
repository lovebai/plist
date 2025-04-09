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
	"time"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		log.Printf("Method: %s, URL: %s, Duration: %s\n", r.Method, r.URL.Path, duration)
	})
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
