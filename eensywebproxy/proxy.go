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

// getBaseUrl :Nginx will inject headers before passing the request to the proxy here.
// We get the X-Host from the header and then form the the baseurl
func getBaseUrl(c *gin.Context) (string, error) {
	baseIP := c.Request.Header.Get("X-Host")
	if baseIP == "" {
		return "", fmt.Errorf("invalid X-Host in the header: expecting host IP/domain in it")
	}
	return fmt.Sprintf("http://%s", baseIP), nil
}

// sendIndexHtml : common function that sends out index.html with basic page information
func sendIndexHtml(c *gin.Context) {

	absBaseUrl, _ := getBaseUrl(c)
	c.HTML(http.StatusOK, "index.html", gin.H{
		"pageTitle": "Eensymachines",
		"ogImage":   fmt.Sprintf("%s/images/circuit06.jpg", absBaseUrl),
		"ogUrl":     absBaseUrl,
		"ogTitle":   "EensyMachines",
		"ogDesc":    "Internet of things company",
	})
}

// dispatchError : will send out the error page when called
// depending on the httpstatus code this picks the correct page and dispatches the error message
func dispatchError(code int, c *gin.Context, e string) {
	title := "Unknown error"
	if code == 400 {
		title = "Bad request"
	} else if code == 404 {
		title = "Not found"
	} else if code == 500 {
		title = "Internal server error"
	}
	c.HTML(code, fmt.Sprintf("%d.html", code), gin.H{
		"pageTitle":  "Eensymachines",
		"imagePath":  fmt.Sprintf("/images/%d.jpg", code),
		"errTitle":   title,
		"errMessage": e,
	})
}
func main() {
	gin.SetMode(gin.DebugMode)
	r := gin.Default()
	r.LoadHTMLGlob(fmt.Sprintf("%s/pages/*", statics))
	// r.LoadHTMLGlob("/usr/src/eensy/web/views/*")
	r.GET("/", sendIndexHtml)

	r.GET("/blogs/:bid", func(c *gin.Context) {
		jsonFile, err := os.Open(fmt.Sprintf("%s/data/blogs.json", statics))
		bid := c.Param("bid")
		if err != nil {
			fmt.Println(err)
			// failed to open the data file
			// sending std. index.html without any modification on the meta tags
			dispatchError(500, c, "We arent able to find the blog information right now. Wait for admin to fix this.")
			return
		}
		defer jsonFile.Close() // the file is closed on exit
		byteValue, _ := ioutil.ReadAll(jsonFile)
		data := blogData{[]blogMeta{}}
		if json.Unmarshal(byteValue, &data) != nil {
			dispatchError(500, c, "Failed to read blog content. Something on the server side isnt quite right. Wait for admin to fix this")
			return
		}
		fmt.Printf("Data from json file..%v", data)
		absBaseUrl, err := getBaseUrl(c)

		if err != nil {
			// this happens when the nginx server url wasnt read back into the proxy
			// nginx server IP/domain is passed into the proxy as a part of a header
			dispatchError(400, c, "Something not right about the request being sent. Wait for an admin to fix this")
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
		dispatchError(404, c, "Blog content was moved from here, perhaps retired. Cannot say when shall it be back.")

	})
	r.GET("/blogs", sendIndexHtml)
	r.GET("/products/:bid", sendIndexHtml)
	r.GET("/products", sendIndexHtml)
	r.GET("/about", sendIndexHtml)
	r.GET("/error", func(c *gin.Context) {
		// Sends out an error page
		// this route is only for testing purposes as the error page is essentially sent from other handlers
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"pageTitle":  "Eensymachines",
			"imagePath":  "/images/400.jpg",
			"errTitle":   "Not found",
			"errMessage": "Failed to get resource at the desired location.",
		})
	})
	log.Fatal(r.Run(":8080"))
}
