package coomer

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	//tls_client "github.com/bogdanfinn/tls-client"
	//"github.com/bogdanfinn/tls-client/profiles"
	"net/http"
	"party-dl/internal/utils"
	"strconv"
	"strings"
)

var (
	metaMutex             sync.Mutex
	extensionDirectoryMap = map[string]string{
		".jpg":  "images",
		".jpeg": "images",
		".png":  "images",
		".gif":  "images",
		".mp4":  "videos",
		".mov":  "videos",
		// Add more extensions and directories as needed
	}
)

type CreatorInfo struct {
	Service     string
	ServiceLink string
	Posts       int
	Name        string
}

type Post struct {
	URL string
}

type Manager struct {
	Client http.Client
}

type PostContent struct {
	DownloadURLS []string
	Description  string
	Published    time.Time
}

func New() (*Manager, error) {
	c := Manager{}
	client := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	c.Client = client
	return &c, nil
}

func (c *Manager) CreatorInfo(url string) (*CreatorInfo, error) {
	res, err := c.Client.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	header := doc.Find("#user-header__info-top > a").First()
	link, exists := header.Attr("href")
	if !exists {
		return nil, fmt.Errorf("creator link not found")
	}
	service := utils.GetService(link)

	name := doc.Find("#user-header__info-top > a > span:nth-child(2)").First().Text()

	paginator := doc.Find("#paginator-top > small").First()

	splitPagination := strings.Split(strings.TrimSpace(paginator.Text()), " ")
	posts, err := strconv.Atoi(splitPagination[len(splitPagination)-1])
	if err != nil {
		return nil, err
	}

	return &CreatorInfo{Service: service, ServiceLink: link, Posts: posts, Name: strings.ToLower(name)}, nil
}

func (c *Manager) ScrapePage(url string, i int) ([]Post, bool, error) {
	res, err := c.Client.Get(fmt.Sprintf("%s?o=%v", url, i*50))
	if err != nil {
		return nil, false, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 && res.StatusCode != 302 {
		return nil, false, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}
	if res.StatusCode == 302 {
		return nil, true, nil
	}
	var posts []Post

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, false, err
	}
	cardList := doc.Find("#main > section > div.card-list.card-list--legacy > div.card-list__items")
	cardList.Children().Each(func(i int, selection *goquery.Selection) {
		a := selection.Children().First()
		link, exists := a.Attr("href")
		if exists {
			posts = append(posts, Post{URL: link})
		}
	})
	return posts, false, nil
}

func (c *Manager) GetPostContent(url string) (*PostContent, error) {
	res, err := c.Client.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	postContent := PostContent{}

	postContent.Description = doc.Find("#page > div > div.post__content > pre").First().Text()

	published := doc.Find("#page > header > div.post__info > div.post__published").First().Text()

	layout := "2006-01-02 15:04:05" // layout string for the given date format
	parsedTime, err := time.Parse(layout, strings.TrimSpace(strings.Split(published, ": ")[1]))
	if err != nil {
		return nil, err
	}
	postContent.Published = parsedTime
	files := doc.Find("#page > div > div.post__files")
	files.Children().Each(func(i int, selection *goquery.Selection) {
		link, exists := selection.Find("a").Attr("href")
		if exists {
			postContent.DownloadURLS = append(postContent.DownloadURLS, link)
		}
	})

	attachments := doc.Find("#page > div > ul.post__attachments")
	attachments.Children().Each(func(i int, selection *goquery.Selection) {
		link, exists := selection.Find("a").Attr("href")
		if exists {
			postContent.DownloadURLS = append(postContent.DownloadURLS, link)
		}
	})

	return &postContent, nil
}

func DownloadPost(postContent *PostContent, baseDir string) error {
	for _, url := range postContent.DownloadURLS {
		fileName := getFileNameFromURL(url)
		var filePath string
		extension := strings.ToLower(filepath.Ext(fileName))
		dir, ok := extensionDirectoryMap[extension]
		if !ok {
			fmt.Printf("Unsupported file type for URL: %s\n", url)
			continue
		}
		dirPath := filepath.Join(baseDir, dir)
		filePath = filepath.Join(dirPath, fileName)
		if err := downloadFile(url, filePath); err != nil {
			return err
		}
	}
	return nil
}

func downloadFile(url string, filePath string) error {
	// Create the file to store the downloaded content
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Perform the HTTP request
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// Copy the response body to the file
	_, err = io.Copy(out, response.Body)
	if err != nil {
		return err
	}

	return nil
}

func getFileNameFromURL(url string) string {
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}
