package cmd

import (
	"party-dl/internal/coomer"
	"party-dl/internal/downloader"
	"party-dl/internal/metadata"
	"path"

	"github.com/alitto/pond"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"party-dl/internal/utils"
	"sync"
)

var (
	failedMutex    sync.Mutex
	baseLocation   string
	numThreads     int
	defaultThreads = 3
)

func downloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "download {url}",
		Short:   "download a creator's page",
		Example: "party-dl download {url}",
		Aliases: []string{"d", "download"},
		Args:    cobra.ExactArgs(1),
		RunE:    download,
	}
	cmd.Flags().StringVarP(&baseLocation, "base-location", "b", "./", "Base download location")
	cmd.Flags().IntVarP(&numThreads, "threads", "t", defaultThreads, "Number of download threads")
	return cmd
}

func download(cmd *cobra.Command, args []string) error {
	url := args[0]
	if !utils.IsURlSupported(url) {
		log.Errorf("%s is not a supported url", url)
		return nil
	}
	log.Infof("Downloading %s", url)

	coomerManager, err := coomer.New()
	if err != nil {
		log.Error(err)
		return nil
	}

	info, err := coomerManager.CreatorInfo(url)
	if err != nil {
		log.Error(err)
		return nil
	}

	log.Info("Scraping creator", "name", info.Name, "service", info.Service, "page", info.ServiceLink, "posts", info.Posts)

	posts, err := scrapePosts(coomerManager, url)
	if err != nil {
		log.Error(err)
		return nil
	}

	basePath := path.Join(baseLocation, info.Name)

	downloadManager := downloader.NewDownloader(basePath, metadata.CreatorInfo{
		Name:     info.Name,
		Service:  info.Service,
		PageLink: info.ServiceLink,
	})

	failedPosts := downloadPosts(coomerManager, downloadManager, posts)

	retryFailedPosts(downloadManager, failedPosts)

	log.Infof("Done.")

	return nil
}

func scrapePosts(coomerManager *coomer.Manager, url string) ([]coomer.Post, error) {
	scrapeIndex := 0
	var posts []coomer.Post
	for {
		log.Infof("Scraping page %v", scrapeIndex+1)
		pagePosts, done, err := coomerManager.ScrapePage(url, scrapeIndex)
		if err != nil {
			return nil, err
		}
		if done {
			log.Infof("Page %v doesn't exists. Finished scraping.", scrapeIndex+1)
			break
		}
		posts = append(posts, pagePosts...)
		scrapeIndex++
	}
	log.Infof("Total scraped posts: %v", len(posts))
	return posts, nil
}

func downloadPosts(coomerManager *coomer.Manager, downloadManager *downloader.Downloader, posts []coomer.Post) []coomer.PostContent {
	pool := pond.New(numThreads, len(posts))
	var failedPosts []coomer.PostContent

	for _, post := range posts {
		postCopy := post                   // Create a copy of post inside the loop
		coomerManagerCopy := coomerManager // Create a copy of coomerManager inside the loop
		pool.Submit(func() {
			postContent, err := coomerManagerCopy.GetPostContent("https://coomer.su" + postCopy.URL)
			if err != nil {
				log.Error(err)
				return
			}
			for _, url := range postContent.DownloadURLS {
				downloadedPath, exists, err := downloadManager.DownloadURL(url, postContent.Description, postContent.Published)
				if err != nil {
					log.Error(err)
					failedMutex.Lock()
					failedPosts = append(failedPosts, coomer.PostContent{Description: postContent.Description, DownloadURLS: []string{url}, Published: postContent.Published})
					failedMutex.Unlock()
					continue
				}
				if exists {
					log.Infof("%s has already been downloaded", url)
				} else {
					log.Infof("Downloaded %s to %s", url, downloadedPath)
				}
			}
		})
	}

	pool.StopAndWait()
	return failedPosts
}

func retryFailedPosts(downloadManager *downloader.Downloader, failedPosts []coomer.PostContent) {
	log.Infof("Retrying %v failed posts", len(failedPosts))
	for _, post := range failedPosts {
		for _, url := range post.DownloadURLS {
			downloadedPath, exists, err := downloadManager.DownloadURL(url, post.Description, post.Published)
			if err != nil {
				log.Error(err)
				failedMutex.Lock()
				failedPosts = append(failedPosts, coomer.PostContent{Description: post.Description, DownloadURLS: []string{url}})
				failedMutex.Unlock()
				continue
			}
			if exists {
				log.Infof("%s has already been downloaded", url)
			} else {
				log.Infof("Downloaded %s to %s", url, downloadedPath)
			}
		}
	}
}
