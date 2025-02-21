package app

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	parseFile "odata/internal/app/services"
	"os"
	"strings"
)

type Collection struct {
	Info Info   `json:"info"`
	Item []Item `json:"item"`
}

type Info struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
}

type Item struct {
	Name    string  `json:"name"`
	Request Request `json:"request,omitempty"`
	Item    []Item  `json:"item,omitempty"`
}

type Request struct {
	Method string   `json:"method"`
	Header []Header `json:"header"`
	URL    URL      `json:"url"`
	Body   Body     `json:"body,omitempty"`
}

type Header struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

type URL struct {
	Raw   string   `json:"raw"`
	Host  []string `json:"host"`
	Path  []string `json:"path"`
	Query []Query  `json:"query"`
}

type Query struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
	Disabled    bool   `json:"disabled"`
}

type Body struct {
	Mode    string `json:"mode"`
	Raw     string `json:"raw,omitempty"`
	Options struct {
		Raw struct {
			Language string `json:"language"`
		} `json:"raw"`
	} `json:"options,omitempty"`
}

type Metadata struct {
	XMLName      xml.Name      `xml:"Edmx"`
	DataServices []DataService `xml:"DataServices>Schema"`
}

type DataService struct {
	XMLName     xml.Name     `xml:"Schema"`
	EntityTypes []EntityType `xml:"EntityType"`
	EntitySets  []EntitySet  `xml:"EntityContainer>EntitySet"`
}

type EntityType struct {
	Name       string     `xml:"Name,attr"`
	Properties []Property `xml:"Property"`
	Key        Key        `xml:"Key"`
}

type EntitySet struct {
	Name       string `xml:"Name,attr"`
	EntityType string `xml:"EntityType,attr"`
}

type Property struct {
	Name string `xml:"Name,attr"`
	Type string `xml:"Type,attr"`
}

type Key struct {
	PropertyRef []PropertyRef `xml:"PropertyRef"`
}

type PropertyRef struct {
	Name string `xml:"Name,attr"`
}

func generateExampleValue(propertyType string) string {
	switch propertyType {
	case "Guid":
		return "00000000-0000-0000-0000-000000000000"
	case "int":
		return "1"
	case "long":
		return "example"
	case "decimal":
		return "1.1"
	default:
		return "XZ"
	}
}

func generateRequestBody(properties []Property) string {
	body := map[string]interface{}{}
	for _, prop := range properties {
		body[prop.Name] = generateExampleValue(prop.Type)
	}
	jsonBody, _ := json.MarshalIndent(body, "", "  ")
	return string(jsonBody)
}

func Process() {
	// URL страницы
	metadataURL := os.Getenv("URL_META_DATA")

	resp, err := http.Get(metadataURL)
	if err != nil {
		fmt.Println("Error fetching metadata:", err)
		return
	}
	defer resp.Body.Close()

	metadata, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading metadata:", err)
		return
	}

	var metadataXML Metadata
	err = xml.Unmarshal(metadata, &metadataXML)
	if err != nil {
		fmt.Println("Error unmarshalling metadata:", err)
		return
	}

	collection := Collection{
		Info: Info{
			Name:   "Sherp OData Service",
			Schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		Item: []Item{},
	}

	// Создаем папки для каждой сущности
	entityFolders := map[string]*Item{}

	for _, dataService := range metadataXML.DataServices {
		for _, entitySet := range dataService.EntitySets {

			entityName := entitySet.Name
			entityType := strings.Split(entitySet.EntityType, ".")[1]

			var keyProperties []Property
			var allProperties []Property
			for _, entityTypeElem := range dataService.EntityTypes {
				if entityTypeElem.Name == entityType {
					for _, key := range entityTypeElem.Key.PropertyRef {
						for _, prop := range entityTypeElem.Properties {
							if prop.Name == key.Name {
								keyProperties = append(keyProperties, prop)
							}
						}
					}
					allProperties = append(allProperties, entityTypeElem.Properties...)
				}
			}

			var filterExample []string
			for _, prop := range keyProperties {
				filterExample = append(filterExample, fmt.Sprintf("%s eq %s", prop.Name, generateExampleValue(prop.Type)))
			}

			var selectProps []string
			for _, prop := range keyProperties {
				selectProps = append(selectProps, prop.Name)
			}

			// Создаем папку для сущности, если она еще не создана
			if _, exists := entityFolders[entityName]; !exists {
				entityFolders[entityName] = &Item{
					Name: entityName,
					Item: []Item{},
				}
			}

			types := parseFile.RunParser(entityName)
			var entityTypeForUrl, entityNameTypeForUrl string
			for _, param := range types {
				entityTypeForUrl = param.Type
				entityNameTypeForUrl = param.Value
			}
			partUrl := generateExampleValue(entityTypeForUrl)

			// GET Request
			getRequest := Item{
				Name: fmt.Sprintf(entityName),
				Request: Request{
					Method: "GET",
					Header: []Header{
						{
							Key:   "Accept",
							Value: "application/json",
							Type:  "text",
						},
					},
					URL: URL{
						Raw:  fmt.Sprintf("https://{{odata}}/pg/odata/%s/%s/?$filter=%s&$select=%s&$top=10&$skip=0", entityName, partUrl, strings.Join(filterExample, " and "), strings.Join(selectProps, ",")),
						Host: []string{"{{odata}}"},
						Path: []string{entityName},
						Query: []Query{
							{
								Key:         "$filter",
								Value:       strings.Join(filterExample, " and "),
								Description: "Filter results by key properties",
							},
							{
								Key:         "$select",
								Value:       strings.Join(selectProps, ","),
								Description: "Select key properties",
							},
							{
								Key:         "$top",
								Value:       "10",
								Description: "Limit number of results to 10",
							},
							{
								Key:         "$skip",
								Value:       "0",
								Description: "Skip 0 results",
							},
						},
					},
				},
			}
			entityFolders[entityName].Item = append(entityFolders[entityName].Item, getRequest)

			// POST Request
			postRequest := Item{
				Name: fmt.Sprintf(entityName),
				Request: Request{
					Method: "POST",
					Header: []Header{
						{
							Key:   "Content-Type",
							Value: "application/json",
							Type:  "text",
						},
					},
					URL: URL{
						Raw:  fmt.Sprintf("https://{{odata}}/pg/odata/%s", entityName),
						Host: []string{"{{odata}}"},
						Path: []string{entityName},
					},
					Body: Body{
						Mode: "raw",
						Raw:  generateRequestBody(allProperties),
						Options: struct {
							Raw struct {
								Language string `json:"language"`
							} `json:"raw"`
						}{
							Raw: struct {
								Language string `json:"language"`
							}{
								Language: "json",
							},
						},
					},
				},
			}
			entityFolders[entityName].Item = append(entityFolders[entityName].Item, postRequest)

			// PUT Request
			putRequest := Item{
				Name: fmt.Sprintf(entityName),
				Request: Request{
					Method: "PUT",
					Header: []Header{
						{
							Key:   "Content-Type",
							Value: "application/json",
							Type:  "text",
						},
					},
					URL: URL{
						Raw:  fmt.Sprintf("https://{{odata}}/pg/odata/%s/%s", entityName, partUrl),
						Host: []string{"{{odata}}"},
						Path: []string{entityName, partUrl},
						Query: []Query{
							{
								Disabled:    true,
								Description: "В URL вставлен " + partUrl + " в sherp это поле " + entityNameTypeForUrl,
							},
						},
					},
					Body: Body{
						Mode: "raw",
						Raw:  generateRequestBody(allProperties),
						Options: struct {
							Raw struct {
								Language string `json:"language"`
							} `json:"raw"`
						}{
							Raw: struct {
								Language string `json:"language"`
							}{
								Language: "json",
							},
						},
					},
				},
			}
			entityFolders[entityName].Item = append(entityFolders[entityName].Item, putRequest)

			// PATCH Request
			patchRequest := Item{
				Name: fmt.Sprintf(entityName),
				Request: Request{
					Method: "PATCH",
					Header: []Header{
						{
							Key:   "Content-Type",
							Value: "application/json",
							Type:  "text",
						},
					},
					URL: URL{
						Raw:  fmt.Sprintf("https://{{odata}}/pg/odata/%s/%s", entityName, partUrl),
						Host: []string{"{{odata}}"},
						Path: []string{entityName, partUrl},
						Query: []Query{
							{
								Disabled:    true,
								Description: "В URL вставлен " + partUrl + " в sherp это поле " + entityNameTypeForUrl,
							},
						},
					},
					Body: Body{
						Mode: "raw",
						Raw:  generateRequestBody(allProperties),
						Options: struct {
							Raw struct {
								Language string `json:"language"`
							} `json:"raw"`
						}{
							Raw: struct {
								Language string `json:"language"`
							}{
								Language: "json",
							},
						},
					},
				},
			}
			entityFolders[entityName].Item = append(entityFolders[entityName].Item, patchRequest)
		}
	}

	// Добавляем папки в коллекцию
	for _, folder := range entityFolders {
		collection.Item = append(collection.Item, *folder)
	}

	file, err := json.MarshalIndent(collection, "", "    ")
	if err != nil {
		fmt.Println("Error marshalling collection:", err)
		return
	}

	err = ioutil.WriteFile("sherp_odata_collection.json", file, 0644)
	if err != nil {
		fmt.Println("Error writing collection to file:", err)
		return
	}

	fmt.Println("Postman collection created")

}
