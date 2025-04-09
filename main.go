package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

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
	initConfig()

	categoryCache = scanCategories(config.ImageDir)

	// 路由设置
	http.HandleFunc("/login", loginHandler)

	if config.Linuxdo != "false" {
		http.HandleFunc("/oauth2/linxdo", initiateAuthHandler)
		http.HandleFunc("/oauth2/callback", callbackHandler)
	}

	if config.Dynamic == "true" {
		http.Handle("/api/index/", AuthMiddleware(http.HandlerFunc(indexJson)))
		http.Handle("/api/category/", AuthMiddleware(http.HandlerFunc(categoryJson)))
	}

	http.Handle("/", AuthMiddleware(http.HandlerFunc(indexHandler)))
	http.Handle("/category/", AuthMiddleware(http.HandlerFunc(categoryHandler)))
	http.Handle("/images/", AuthMiddleware(http.StripPrefix("/images/", http.FileServer(http.Dir(config.ImageDir)))))

	log.Println("服务器启动在 :", config.Port)
	if err := http.ListenAndServe(":"+config.Port, loggingMiddleware(http.DefaultServeMux)); err != nil {
		log.Fatal(err)
	}
}
