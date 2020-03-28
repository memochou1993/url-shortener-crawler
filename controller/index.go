package controller

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/memochou1993/crawler/helper"
	"golang.org/x/net/html"
)

const (
	baseURL = "https://risu.io/"
)

// Image struct
type Image struct {
	Code      string
	FileInfos []FileInfo `json:"file_infos"`
}

// FileInfo struct
type FileInfo struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	ByteSize    string `json:"byte_size"`
	FilePath    string `json:"file_path"`
	CreatedAt   string `json:"created_at"`
}

// Handle func
func Handle() {
	nums := int(math.Pow(52, 2))
	codes := generateCodes(nums)

	codeChan := make(chan string)
	imageChan := make(chan Image)

	go func() {
		for _, code := range codes {
			codeChan <- code
		}
	}()

	for i := 0; i < 20; i++ {
		go func() {
			defer helper.Measure(time.Now(), "main")

			for code := range codeChan {
				image := fetchImage(code)

				nums--

				fmt.Println("nums", nums)

				go func() {
					defer helper.Measure(time.Now(), "fetch")

					imageChan <- image
				}()
			}
		}()
	}

	for image := range imageChan {
		if len(image.FileInfos) > 0 {
			image.download()
		}
	}
}

func (image *Image) setCode(code string) {
	image.Code = code
}

func (image *Image) download() error {
	defer helper.Measure(time.Now(), "download")

	name := "storage/" + image.Code + ".jpg"
	url := image.FileInfos[0].FilePath

	return storeImage(name, url)
}

func storeImage(path string, url string) error {
	resp, err := http.Get(url)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	file, err := os.Create(path)

	if err != nil {
		return err
	}

	defer file.Close()

	_, err = io.Copy(file, resp.Body)

	return err
}

func fetchImage(code string) Image {
	var image Image

	client := &http.Client{
		Timeout: time.Duration(10 * time.Second),
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	req, err := http.NewRequest("GET", baseURL+code, nil)

	if err != nil {
		return image
	}

	resp, err := client.Do(req)

	if err != nil {
		return image
	}

	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)

	if err != nil {
		return image
	}

	node := getNode(doc)

	if err = json.Unmarshal([]byte(node), &image); err != nil {
		return image
	}

	image.setCode(code)

	return image
}

func getNode(n *html.Node) string {
	node := ""

	var f func(*html.Node)

	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "page-image" {
			for _, a := range n.Attr {
				node = a.Val
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(n)

	return node
}

func generateCodes(nums int) []string {
	codes := helper.Codes(nums)

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(codes), func(i, j int) {
		codes[i], codes[j] = codes[j], codes[i]
	})

	return codes
}
