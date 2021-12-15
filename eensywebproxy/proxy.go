package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	// json and html files that are needed to be referred inside the reverse proxy
	statics = "/usr/src/eensy/web"
)

type blogMeta struct {
	Title    string `json:"title"`
	SubTitle string `json:"sub_title"`
	Cover    string `json:"cover"`
	Id       string `json:"id"`
}
type blogData struct {
	Data []blogMeta `json:"d"`
}

// sendIndexHtml : common function that sends out index.html with basic page information
func sendIndexHtml(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"pageTitle": "Eensymachines",
	})
	return
}
func main() {
	gin.SetMode(gin.DebugMode)
	r := gin.Default()
	r.LoadHTMLGlob(fmt.Sprintf("%s/pages/*", statics))
	// r.LoadHTMLGlob("/usr/src/eensy/web/views/*")
	r.GET("/", sendIndexHtml)
	r.GET("/blogs", sendIndexHtml)
	r.GET("/blogs/:bid", func(c *gin.Context) {
		jsonFile, err := os.Open(fmt.Sprintf("%s/blogs.json", statics))
		bid := c.Param("bid")
		if err != nil {
			// failed to open the data file
			// sending std. index.html without any modification on the meta tags
			c.HTML(http.StatusOK, "index.html", gin.H{
				"pageTitle": "Eensymachines",
			})
			return
		}
		defer jsonFile.Close() // the file is closed on exit
		byteValue, _ := ioutil.ReadAll(jsonFile)
		data := blogData{[]blogMeta{}}
		if json.Unmarshal(byteValue, &data) != nil {
			c.HTML(http.StatusInternalServerError, "500.html", gin.H{
				"pageTitle": "Eensymachines",
			})
			return
		}
		baseIP := c.Request.Header.Get("X-Host")
		fmt.Printf("Header values %v\n", c.Request.Header)
		absBaseUrl := fmt.Sprintf("http://%s", baseIP)

		if baseIP == "" {
			c.HTML(http.StatusBadRequest, "400.html", gin.H{
				"pageTitle": "Eensymachines",
			})
			return
		}
		for _, m := range data.Data {
			fmt.Println(m.Id)
			if m.Id == bid {
				// this is the blog we are looking for
				c.HTML(http.StatusOK, "index.html", gin.H{
					"pageTitle": "Eensymachines",
					"ogImage":   fmt.Sprintf("%s/images/%s", absBaseUrl, m.Cover),
					"ogUrl":     fmt.Sprintf("%s/blogs/%s", absBaseUrl, bid),
					"ogTitle":   m.Title,
					"ogDesc":    m.SubTitle,
				})
				return
			}
		}
		fmt.Printf("Blog with the id wasnt found %s\n", bid)
		c.HTML(http.StatusBadRequest, "400.html", gin.H{
			"pageTitle":  "Eensymachines",
			"errMessage": "Blog with id wasnt found- it may have been removed!",
		})

	})
	r.GET("/products", sendIndexHtml)
	r.GET("/products/:bid", sendIndexHtml)
	r.GET("/about", sendIndexHtml)
	log.Fatal(r.Run(":8080"))
}
