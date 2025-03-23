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
	ImageDir: "./",
	Password: "",
	Port:     "8008",
	Title:    "在线图集",
	Icon:     "https://i.obai.cc/favicon.ico",
	Dynamic:  "false",
}

var imageExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
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

	log.Printf("配置信息: %s", config)

	// 路由设置
	http.HandleFunc("/login", loginHandler)
	if config.Dynamic == "true" {
		http.Handle("/api/index/", AuthMiddleware(http.HandlerFunc(indexJson)))
		http.Handle("/api/category/", AuthMiddleware(http.HandlerFunc(categoryJson)))
	}
	http.Handle("/", AuthMiddleware(http.HandlerFunc(indexHandler)))
	http.Handle("/category/", AuthMiddleware(http.HandlerFunc(categoryHandler)))
	http.Handle("/images/", AuthMiddleware(http.StripPrefix("/images/", http.FileServer(http.Dir(config.ImageDir)))))
	http.Handle("/js/", AuthMiddleware(http.StripPrefix("/js/", http.FileServer(http.Dir("./js")))))
	log.Println("服务器启动在 :", config.Port)
	log.Fatal(http.ListenAndServe(":"+config.Port, nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if config.Dynamic == "true" {
		tmpl := template.Must(template.New("index").Parse(indexDynamicTemplate))
		tmpl.Execute(w, config)
	} else {
		categories, err := os.ReadDir(config.ImageDir)
		if err != nil {
			http.Error(w, "无法读取目录", http.StatusInternalServerError)
			return
		}

		type Category struct {
			Name        string
			EncodedName string
			CoverImage  string
		}

		type Tmp struct {
			Category []Category
			Config   Config
		}

		var categoryList []Category
		for _, category := range categories {
			if category.IsDir() {
				dirPath := filepath.Join(config.ImageDir, category.Name())
				entries, err := os.ReadDir(dirPath)
				if err != nil {
					log.Printf("无法读取目录 %s: %v", dirPath, err)
					continue
				}

				var coverImage string
				// 遍历目录项查找第一个有效图片
				for _, entry := range entries {
					if entry.IsDir() {
						continue
					}
					ext := strings.ToLower(filepath.Ext(entry.Name()))
					if imageExtensions[ext] {
						coverImage = entry.Name()
						break // 找到第一个有效图片即停止
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
		var tmp = Tmp{
			Category: categoryList,
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
		page = 1 // 默认第一页
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20 // 默认每页20个分类
	}

	// 读取分类目录
	categories, err := os.ReadDir(config.ImageDir)
	if err != nil {
		http.Error(w, "无法读取目录", http.StatusInternalServerError)
		return
	}
	type Category struct {
		Name        string
		EncodedName string
		CoverImage  string
	}

	var categoryList []Category
	for _, category := range categories {
		if category.IsDir() {
			dirPath := filepath.Join(config.ImageDir, category.Name())
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

	// 计算分页信息
	totalCategories := len(categoryList)
	totalPages := (totalCategories + limit - 1) / limit
	start := (page - 1) * limit
	end := start + limit
	if start > totalCategories {
		start = totalCategories
	}
	if end > totalCategories {
		end = totalCategories
	}
	currentCategories := categoryList[start:end]

	// 返回 JSON 数据
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
			log.Printf("请求路径: %s, Cookie状态: %+v, 错误信息: %v", r.URL.Path, cookie, err)

			if err != nil || !verifyCookie(cookie) {
				log.Printf("验证失败，跳转登录页面")
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

// 动态加载主页面模板
const indexDynamicTemplate = `<!DOCTYPE html>
<html lang="zh">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="description" content="{{.Title}}，图片分类网站，展示各类图片集合。">
    <meta name="keywords" content="图片, 分类, 相册">
    <title>{{.Title}}</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css">
    <link rel="shortcut icon" type="image/x-icon" href="{{.Icon}}" />
    <style>
        .category-card { text-align: center; margin-bottom: 20px; }
        .category-card img { width: 100%; height: auto; border-radius: 8px; }
        .category-card p { margin-top: 10px; font-size: 1.1em; }
        #loading { text-align: center; padding: 20px; display: none; }
		img.lazy {
			background-image: url(data:image/gif;base64,R0lGODlhEgASAIABAKa4zP///yH/C05FVFNDQVBFMi4wAwEAAAAh+QQJAwABACwAAAAAEgASAEACJwyOoYa3D6N8rVqgLp5M2+x9XcWBTTmGTqqa6qqxFInWUMzhk76TBQAh+QQJAwABACwAAAAAEgASAEACKQyOoYa3D6NUrdHqGJ44d3B9m1ZNZGZ+YXmKnsuq44qaNqSmnZ3rllIAACH5BAkDAAEALAAAAAASABIAQAIpDI6hhrcPo2zt0cRuvG5xoHxfyE2UZJWeKrLtmZ3aWqG2OaOjvfPwUgAAIfkECQMAAQAsAAAAABIAEgBAAigMjqGGtw8jbC3SxO67bnLFhQD4bZRkap4qli37qWSF1utZh7a+41ABACH5BAkDAAEALAAAAAASABIAQAIqDI6hhrcP42pNMgoUdpfanXVgJSaaZ53Yt6kj+a6lI7tcioN5m+o7KSkAACH5BAkDAAEALAAAAAASABIAQAIoDI6hhrcPI2tOKpom3vZyvVEeBgLdKHYhGjZsW63kMp/Sqn4WnrtnAQAh+QQJAwABACwAAAAAEgASAEACKAyOocvtCCN0TB5lM6Ar92hYmChxX2l6qRhqYAui8GTOm8rhlL6/ZgEAIfkECQMAAQAsAAAAABIAEgBAAigMjqHL7QgjdEyeJY2leHOdgZF4KdYJfGTynaq7XmGctuicwZy+j2oBACH5BAkDAAEALAAAAAASABIAQAInDI6hy+0II3RMHrosUFpjbmUROJFdiXmfmoafMZoodUpyLU5sO1MFACH5BAkDAAEALAAAAAASABIAQAImDI6hy+2GDozyKZrspBf7an1aFy2fuJ1Z6I2oho2yGqc0SYN1rRUAIfkECQMAAQAsAAAAABIAEgBAAiYMjqHL7W+QVLJaAOnVd+eeccliRaXZVSH4ee0Uxg+bevUJnuIRFAAh+QQJAwABACwAAAAAEgASAEACKoyBacvtnyI4EtH6QrV6X5l9UYgt2DZ1JRqqIOm1ZUszrIuOeM6x8x4oAAAh+QQJAwABACwAAAAAEgASAEACKIwNqcftryJAMrFqG55hX/wcnlN9UQeipZiGo9vCZ0hD6TbiN7hSZwEAIfkECQMAAQAsAAAAABIAEgBAAiiMH6CL7Z+WNHK2yg5WdLsNQB12VQgJjmZJiqnriZEl1y94423aqlwBACH5BAkDAAEALAAAAAASABIAQAIrjH+gi+2+IjCSvaoo1vUFPHnfxlllBp5mk4qt98KSSKvZCHZ4HtmTrgoUAAAh+QQFAwABACwAAAAAEgASAEACKIyPAcvpr5g0csJYc8P1cgtpwDceGblQmiey69W6oOfEon2f6KirUwEAIfkECQMAAQAsAAAPAAgAAwBAAgSMj6lXACH5BAkDAAEALAAAAAASABIAQAIYjI+JwK0Po5y02glUvrz7bzXiBpbLaD4FACH5BAkDAAEALAAAAAASABIAQAImjI8By8qfojQPTldzw/VymB3aCIidN6KaGl7kSnWpC6ftt00zDRUAIfkECQMAAQAsAAAAABIAEgBAAiaMjwHLyp+iNA9WcO6aVHOneWBYZeUXouJEiu1lWit7jhCX4rMEFwAh+QQJAwABACwAAAAAEgASAEACJ4yPAcvKn6I0r1pA78zWQX51XrWBSzl+Uxia7Jm+mEujW3trubg3BQAh+QQFAwABACwAAAAAEgASAEACJwyOoYa3D6N8rVqgLp5M2+x9XcWBTTmGTqqa6qqxFInWUMzhk76TBQA7);
			background-repeat: no-repeat;
			background-position: 50%;
			background-size: auto;
			background-color: #ECEFF1;
		}
		img.loaded {
            background-image: none;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1 class="my-4 text-center">{{.Title}}</h1>
        <div class="row" id="category-container">
        </div>
        <div id="loading">加载中...</div>
    </div>
    <script src="https://code.jquery.com/jquery-3.6.0.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/js/bootstrap.bundle.min.js"></script>
    <script>
		let page = 1;
        const limit = 20;
        let loading = false;
        let hasMore = true;
		
		const lazyImageObserver = new IntersectionObserver((entries, observer) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    const img = entry.target;
                    if (img.dataset.src) {
                        img.src = img.dataset.src;
                        img.onload = () => {
                            img.classList.add('loaded');
                            img.classList.remove('lazy');
                        };
                    }
                    observer.unobserve(img);
                }
            });
        }, {
            rootMargin: '0px 0px 200px 0px',
            threshold: 0.01
        });


        function loadCategories() {
            if (loading || !hasMore) return;
            loading = true;
            $('#loading').show();

            $.ajax({
                url: '/api/index?page='+page+'&limit='+limit,
                method: 'GET',
                success: function(data) {
                    const categories = data.categories;
                    if (categories.length === 0) {
                        hasMore = false;
                        $('#loading').text('没有更多分类');
                        return;
                    }

                    categories.forEach(category => {
                        const html = 
                            '<div class="col-md-3 col-sm-6">' +
                                '<div class="category-card">' +
                                    '<a href="/category/' + category.EncodedName + '" style="text-decoration: none;">' +
                                        '<img data-src="/images/' + category.EncodedName + '/' + category.CoverImage 
										+ '" src="data:image/gif;base64,R0lGODlhAQABAAAAACH5BAEKAAEALAAAAAABAAEAAAICTAEAOw==" class="img-fluid lazy" alt="' +
										 category.Name + '">' +
                                        '<p>' + category.Name + '</p>' +
                                    '</a>' +
                                '</div>' +
                            '</div>';
						const $newItems = $(html);
						$('#category-container').append($newItems);
						
						$newItems.find('img.lazy').each(function() {
							lazyImageObserver.observe(this);
						});
                    });

                    page++;
                    loading = false;
                    $('#loading').hide();
                },
                error: function() {
                    loading = false;
                    $('#loading').text('加载失败，请重试');
                }
            });
        }

        $(document).ready(function() {
            loadCategories(); // 初始加载第一页

            $(window).scroll(function() {
                if ($(window).scrollTop() + $(window).height() >= $(document).height() - 100) {
                    loadCategories();
                }
            });
        });
	</script>
</body>
</html>`

// 动态加载分类页面模板
const categoryDynamicTemplate = `<!DOCTYPE html>
<html lang="zh">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="description" content="{{.Category}} 的图片集合， {{.Config.Title}}">
    <meta name="keywords" content="{{.Category}}, 图片, 相册">
    <title>{{.Category}} - {{.Config.Title}} - 图片合集</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css">
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/gh/fancyapps/fancybox@3.5.7/dist/jquery.fancybox.min.css">
    <link rel="shortcut icon" type="image/x-icon" href="{{.Config.Icon}}" />
    <style>
        .image-card { margin-bottom: 20px; }
        .image-card img { width: 100%; height: auto; border-radius: 8px; }
        #back-buttons {position: fixed;bottom: 20px;right: 20px;display: flex;flex-direction: column;gap: 10px;z-index: 1000;}
        #back-buttons button {padding: 5px 10px;border: none;color: white;border-radius: 5px;cursor: pointer;font-size: 14px;transition: all 0.3s;}
        #back-buttons button:hover {background-color: #bdc5ca;}
        #loading {text-align: center; padding: 20px; display: none;}
		img.lazy {
			background-image: url(data:image/gif;base64,R0lGODlhEgASAIABAKa4zP///yH/C05FVFNDQVBFMi4wAwEAAAAh+QQJAwABACwAAAAAEgASAEACJwyOoYa3D6N8rVqgLp5M2+x9XcWBTTmGTqqa6qqxFInWUMzhk76TBQAh+QQJAwABACwAAAAAEgASAEACKQyOoYa3D6NUrdHqGJ44d3B9m1ZNZGZ+YXmKnsuq44qaNqSmnZ3rllIAACH5BAkDAAEALAAAAAASABIAQAIpDI6hhrcPo2zt0cRuvG5xoHxfyE2UZJWeKrLtmZ3aWqG2OaOjvfPwUgAAIfkECQMAAQAsAAAAABIAEgBAAigMjqGGtw8jbC3SxO67bnLFhQD4bZRkap4qli37qWSF1utZh7a+41ABACH5BAkDAAEALAAAAAASABIAQAIqDI6hhrcP42pNMgoUdpfanXVgJSaaZ53Yt6kj+a6lI7tcioN5m+o7KSkAACH5BAkDAAEALAAAAAASABIAQAIoDI6hhrcPI2tOKpom3vZyvVEeBgLdKHYhGjZsW63kMp/Sqn4WnrtnAQAh+QQJAwABACwAAAAAEgASAEACKAyOocvtCCN0TB5lM6Ar92hYmChxX2l6qRhqYAui8GTOm8rhlL6/ZgEAIfkECQMAAQAsAAAAABIAEgBAAigMjqHL7QgjdEyeJY2leHOdgZF4KdYJfGTynaq7XmGctuicwZy+j2oBACH5BAkDAAEALAAAAAASABIAQAInDI6hy+0II3RMHrosUFpjbmUROJFdiXmfmoafMZoodUpyLU5sO1MFACH5BAkDAAEALAAAAAASABIAQAImDI6hy+2GDozyKZrspBf7an1aFy2fuJ1Z6I2oho2yGqc0SYN1rRUAIfkECQMAAQAsAAAAABIAEgBAAiYMjqHL7W+QVLJaAOnVd+eeccliRaXZVSH4ee0Uxg+bevUJnuIRFAAh+QQJAwABACwAAAAAEgASAEACKoyBacvtnyI4EtH6QrV6X5l9UYgt2DZ1JRqqIOm1ZUszrIuOeM6x8x4oAAAh+QQJAwABACwAAAAAEgASAEACKIwNqcftryJAMrFqG55hX/wcnlN9UQeipZiGo9vCZ0hD6TbiN7hSZwEAIfkECQMAAQAsAAAAABIAEgBAAiiMH6CL7Z+WNHK2yg5WdLsNQB12VQgJjmZJiqnriZEl1y94423aqlwBACH5BAkDAAEALAAAAAASABIAQAIrjH+gi+2+IjCSvaoo1vUFPHnfxlllBp5mk4qt98KSSKvZCHZ4HtmTrgoUAAAh+QQFAwABACwAAAAAEgASAEACKIyPAcvpr5g0csJYc8P1cgtpwDceGblQmiey69W6oOfEon2f6KirUwEAIfkECQMAAQAsAAAPAAgAAwBAAgSMj6lXACH5BAkDAAEALAAAAAASABIAQAIYjI+JwK0Po5y02glUvrz7bzXiBpbLaD4FACH5BAkDAAEALAAAAAASABIAQAImjI8By8qfojQPTldzw/VymB3aCIidN6KaGl7kSnWpC6ftt00zDRUAIfkECQMAAQAsAAAAABIAEgBAAiaMjwHLyp+iNA9WcO6aVHOneWBYZeUXouJEiu1lWit7jhCX4rMEFwAh+QQJAwABACwAAAAAEgASAEACJ4yPAcvKn6I0r1pA78zWQX51XrWBSzl+Uxia7Jm+mEujW3trubg3BQAh+QQFAwABACwAAAAAEgASAEACJwyOoYa3D6N8rVqgLp5M2+x9XcWBTTmGTqqa6qqxFInWUMzhk76TBQA7);
			background-repeat: no-repeat;
			background-position: 50%;
			background-size: auto;
			background-color: #ECEFF1;
		}
		img.loaded {
            background-image: none;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1 class="my-4 text-center">{{.Category}}</h1>
        <div class="row" id="image-container">
            <!-- 图片将动态加载到这里 -->
        </div>
        <div id="loading">加载中...</div>
    </div>
    <div id="back-buttons">
        <button id="back-btn" onclick="history.back()">⬅</button>
        <button id="top-btn" onclick="scrollToTop()">🔝</button>
    </div>

    <script src="https://code.jquery.com/jquery-3.6.0.min.js"></script>
    <script src="https://cdn.jsdelivr.net/gh/fancyapps/fancybox@3.5.7/dist/jquery.fancybox.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/js/bootstrap.bundle.min.js"></script>
	<script>
	    let page = 1;
        const limit = 20;
        let loading = false;
        let hasMore = true;

		const lazyImageObserver = new IntersectionObserver((entries, observer) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    const img = entry.target;
                    if (img.dataset.src) {
                        img.src = img.dataset.src;
                        img.onload = () => {
                            img.classList.add('loaded');
                            img.classList.remove('lazy');
                        };
                    }
                    observer.unobserve(img);
                }
            });
        }, {
            rootMargin: '0px 0px 200px 0px',
            threshold: 0.01
        });

        function loadImages(category) {            
            if (loading || !hasMore) return;
            loading = true;
            $('#loading').show();

            $.ajax({
                url: '/api/category/'+ category +'?page='+page+'&limit='+limit,
                method: 'GET',
                success: function(data) {
                    const images = data.images;
                    if (images.length === 0) {
                        hasMore = false;
                        $('#loading').text('没有更多图片');
                        return;
                    }

                    images.forEach(image => {
                        const html = 
                            '<div class="col-md-3 col-sm-6">' +
                                '<div class="image-card">' +
                                    '<a href="/images/' + category + '/' + image.Name + '" data-fancybox="' + category + '">' +
                                        '<img data-src="/images/' + category + '/' + image.Name + '" alt="' + image.Name +
										'src="data:image/gif;base64,R0lGODlhAQABAAAAACH5BAEKAAEALAAAAAABAAEAAAICTAEAOw=="' +
										 '" class="img-fluid lazy" ' + (image.Type === 'gif' ? 'data-type="image/gif"' : '') + '>' +
                                    '</a>' +
                                '</div>' +
                            '</div>';
						const $newItems = $(html);
						$('#image-container').append($newItems);
						
						$newItems.find('img.lazy').each(function() {
							lazyImageObserver.observe(this);
						});
                    });
                    page++;
                    loading = false;
                    $('#loading').hide();
                },
                error: function() {
                    loading = false;
                    $('#loading').text('加载失败，请重试');
                }
            });
        }
        function scrollToTop() {
            window.scrollTo({ top: 0, behavior: 'smooth' });
        }
			
        $(document).ready(function() {

			const category = window.location.pathname.split('/').pop();
            loadImages(category); // 初始加载第一页

            $(window).scroll(function() {
                if ($(window).scrollTop() + $(window).height() >= $(document).height() - 100) {
                    loadImages(category);
                }
            });

            $('[data-fancybox]').fancybox();
        });
	</script>
</body>
</html>`

// 主页面模板
const indexTemplate = `<!DOCTYPE html>
<html lang="zh">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="description" content="{{.Config.Title}}，图片分类网站，展示各类图片集合。">
    <meta name="keywords" content="图片, 分类, 相册">
    <title>{{.Config.Title}}</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css">
	<link rel="shortcut icon" type="image/x-icon" href="{{.Config.Icon}}" />
	<style>
        .category-card { text-align: center; margin-bottom: 20px; }
        .category-card img { width: 100%; height: auto; border-radius: 8px; }
        .category-card p { margin-top: 10px; font-size: 1.1em; }
		img.lazy {
            background: #ECEFF1 url('data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="40" stroke="%23ccc" fill="none" stroke-width="6"><animate attributeName="stroke-dashoffset" values="0;300" dur="1.5s" repeatCount="indefinite"/><animate attributeName="stroke-dasharray" values="60 200;160 40;60 200" dur="1.5s" repeatCount="indefinite"/></circle></svg>') no-repeat center/50px;
            min-height: 200px;
            transition: opacity 0.3s;
        }
        img.loaded { opacity: 1; }
        img.error { background-image: url('data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="%23ff4444" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>');}
    </style>
</head>
<body>
    <div class="container">
        <h1 class="my-4 text-center">{{.Config.Title}}</h1>
        <div class="row">
			{{range .Category}}
				<div class="col-md-3 col-sm-6">
					<div class="category-card">
						<a href="/category/{{.EncodedName}}" style="text-decoration: none;">
							<img data-src="/images/{{.EncodedName}}/{{.CoverImage}}"
							 src="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 1 1'%3E%3C/svg%3E"
							 class="img-fluid lazy" loading="lazy" alt="{{.Name}}">
							<p>{{.Name}}</p>
						</a>
					</div>
				</div>
			{{end}}
        </div>
    </div>
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/js/bootstrap.bundle.min.js"></script>
	<script>
	(function() {
        'use strict';
        
        const config = {
            rootMargin: '0px 0px 400px 0px',
            threshold: 0.001
        };
        
        let observer;
        let isPageHidden = false;

        function init() {
            if ('IntersectionObserver' in window) {
                setupObserver();
            } else {
                setupFallback();
            }
            setupVisibilityListener();
        }

        function setupObserver() {
            observer = new IntersectionObserver(handleIntersect, config);
            document.querySelectorAll('img.lazy').forEach(img => {
                observer.observe(img);
            });
        }

        function handleIntersect(entries) {
            if (isPageHidden) return;
            
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    const img = entry.target;
                    loadImage(img);
                    observer.unobserve(img);
                }
            });
        }

        function loadImage(img) {
            if (!img.dataset.src) return;
            
            img.decoding = 'async';
            img.src = img.dataset.src;
            img.removeAttribute('data-src');

            img.onload = () => {
                img.classList.add('loaded');
                img.classList.remove('lazy');
            };
            
            img.onerror = () => {
                img.classList.add('error');
                img.src = '';
            };
        }

        function setupFallback() {
            console.log('IntersectionObserver not supported, using fallback');
        }

        function setupVisibilityListener() {
            document.addEventListener('visibilitychange', () => {
                isPageHidden = document.hidden;
            });
        }

        document.addEventListener('DOMContentLoaded', init);
        window.addEventListener('beforeunload', () => {
            if (observer) observer.disconnect();
        });
    })();
	</script>
</body>
</html>`

// 分类页面模板
const categoryTemplate = `<!DOCTYPE html>
<html lang="zh">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="description" content="{{.Category}} 的图片集合， {{.Config.Title}}">
    <meta name="keywords" content="{{.Category}}, 图片, 相册">
    <title>{{.Category}} - {{.Config.Title}} - 图片合集</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css">
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/gh/fancyapps/fancybox@3.5.7/dist/jquery.fancybox.min.css">
	<link rel="shortcut icon" type="image/x-icon" href="{{.Config.Icon}}" />
	<style>
        .image-card { margin-bottom: 20px; }
        .image-card img { width: 100%; height: auto; border-radius: 8px; }
		#back-buttons {position: fixed;bottom: 20px;right: 20px;display: flex;flex-direction: column;gap: 10px;z-index: 1000;}
		#back-buttons button {padding: 5px 10px;border: none;color: white;border-radius: 5px;cursor: pointer;font-size: 14px;transition: all 0.3s;}
		#back-buttons button:hover {background-color: #bdc5ca;}
		img.lazy {
            background: #ECEFF1 url('data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="40" stroke="%23ccc" fill="none" stroke-width="6"><animate attributeName="stroke-dashoffset" values="0;300" dur="1.5s" repeatCount="indefinite"/><animate attributeName="stroke-dasharray" values="60 200;160 40;60 200" dur="1.5s" repeatCount="indefinite"/></circle></svg>') no-repeat center/50px;
            min-height: 200px;
            transition: opacity 0.3s;
        }
        img.loaded { opacity: 1; }
        img.error { background-image: url('data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="%23ff4444" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>');}
    </style>
</head>
<body>
    <div class="container">
        <h1 class="my-4 text-center">{{.Category}}</h1>
        <div class="row">
            {{range .Images}}
                <div class="col-md-3 col-sm-6">
                    <div class="image-card">
                        <a href="/images/{{$.Category}}/{{.Name}}" data-fancybox="{{$.Category}}">
                            <img data-src="/images/{{$.Category}}/{{.Name}}" alt="{{.Name}}" class="img-fluid lazy" loading="lazy" 
							src="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 1 1'%3E%3C/svg%3E"
							{{if eq .Type "gif"}}data-type="image/gif"{{end}}>
                        </a>
                    </div>
                </div>
            {{end}}
        </div>
    </div>
	<div id="back-buttons">
    <button id="back-btn" onclick="back()">⬅</button>
    <button id="top-btn" onclick="scrollToTop()">🔝</button>
	</div>

    <script src="https://code.jquery.com/jquery-3.6.0.min.js"></script>
    <script src="https://cdn.jsdelivr.net/gh/fancyapps/fancybox@3.5.7/dist/jquery.fancybox.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/js/bootstrap.bundle.min.js"></script>
    <script>
	(function() {
        'use strict';
        
        const config = {
            rootMargin: '0px 0px 400px 0px',
            threshold: 0.001
        };
        
        let observer;
        let isPageHidden = false;

        function init() {
            if ('IntersectionObserver' in window) {
                setupObserver();
            } else {
                setupFallback();
            }
            setupVisibilityListener();
        }

        function setupObserver() {
            observer = new IntersectionObserver(handleIntersect, config);
            document.querySelectorAll('img.lazy').forEach(img => {
                observer.observe(img);
            });
        }

        function handleIntersect(entries) {
            if (isPageHidden) return;
            
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    const img = entry.target;
                    loadImage(img);
                    observer.unobserve(img);
                }
            });
        }

        function loadImage(img) {
            if (!img.dataset.src) return;
            
            img.decoding = 'async';
            img.src = img.dataset.src;
            img.removeAttribute('data-src');

            img.onload = () => {
                img.classList.add('loaded');
                img.classList.remove('lazy');
            };
            
            img.onerror = () => {
                img.classList.add('error');
                img.src = '';
            };
        }

        function setupFallback() {
            console.log('IntersectionObserver not supported, using fallback');
        }

        function setupVisibilityListener() {
            document.addEventListener('visibilitychange', () => {
                isPageHidden = document.hidden;
            });
        }

        document.addEventListener('DOMContentLoaded', init);
        window.addEventListener('beforeunload', () => {
            if (observer) observer.disconnect();
        });
    })();

        $(document).ready(function() {
            $('[data-fancybox]').fancybox();
			$('img[data-type="image/gif"]').each(function() {
				const img = new Image();
				img.src = $(this).attr('data-src');
				img.onload = function() {
					$(this).attr('src', img.src).addClass('loaded');
				}.bind(this);
			});
        });
		function scrollToTop() {
		    window.scrollTo({ top: 0, behavior: 'smooth' });
		}
		function back() {
			history.back();
		}

    </script>
</body>
</html>`

const loginTemplate = `<!DOCTYPE html>
<html lang="zh">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>登录</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css">
</head>
<body class="bg-light">
    <div class="container mt-5">
        <div class="row justify-content-center">
            <div class="col-md-4">
                <div class="card shadow">
                    <div class="card-body">
                        <h3 class="card-title mb-4">请输入访问密码</h3>
                        <form method="POST">
                            <div class="mb-3">
                                <input type="password" 
                                       name="password" 
                                       class="form-control"
                                       placeholder="密码"
                                       required>
                            </div>
                            <button type="submit" 
                                    class="btn btn-primary w-100">
                                登录
                            </button>
                        </form>
                    </div>
                </div>
            </div>
        </div>
    </div>
</body>
</html>`
