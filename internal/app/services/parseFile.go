package parseFile

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type Param struct {
	Value string
	Type  string
}

var cachedHTML string
var cacheLoaded bool

// парсер начался
func RunParser(entityName string) []Param {

	// URL страницы
	url := os.Getenv("URL_ODATA")

	// Получаем содержимое страницы
	htmlContent, err := fetchPage(url)
	if err != nil {
		fmt.Println("Ошибка при получении страницы:", err)
	}

	// Извлекаем шаблоны из HTML
	templates := extractTemplates(htmlContent)

	var params []Param
	for _, template := range templates {
		if strings.Contains(template, entityName) {
			fmt.Println("Найден шаблон:", template)

			rePattern := fmt.Sprintf(`%s/(\{[^}]+\})`, regexp.QuoteMeta(entityName))
			re := regexp.MustCompile(rePattern)
			match := re.FindStringSubmatch(template)
			if len(match) > 1 {
				fmt.Println("Извлеченный параметр:", match[1])
			}

			params = extractParams(template)
			for _, param := range params {
				fmt.Printf("Параметр: Value=%s, Type=%s\n", param.Value, param.Type)
			}
		}
	}

	return params
}

// Функция для получения содержимого страницы по URL
func fetchPage(url string) (string, error) {
	if cacheLoaded {
		return cachedHTML, nil
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	cachedHTML = string(body)
	cacheLoaded = true
	return cachedHTML, nil
}

// Функция для извлечения шаблонов из HTML
func extractTemplates(htmlContent string) []string {
	var templates []string

	re := regexp.MustCompile(`<td><a href="([^"]+)">([^<]+)</a></td>`)
	matches := re.FindAllStringSubmatch(htmlContent, -1)

	for _, match := range matches {
		if len(match) > 2 {
			templates = append(templates, match[2])
		}
	}

	return templates
}

// Функция для извлечения параметров из шаблона
func extractParams(template string) []Param {
	var params []Param

	re := regexp.MustCompile(`\{([^:]+):?([^}]*)\}`)
	matches := re.FindAllStringSubmatch(template, -1)

	for _, match := range matches {
		param := Param{
			Value: match[1],
		}
		if len(match) > 2 && match[2] != "" {
			param.Type = match[2]
		}
		params = append(params, param)
	}

	return params
}
