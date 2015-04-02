package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strings"
)

type Mix struct {
	Filename string
	ID       string
}

type DownloadStatus struct {
	io.Reader
	Filename string
	Total    int64
	Length   int64
	Progress float64
}

func (ds *DownloadStatus) Read(p []byte) (int, error) {
	n, err := ds.Reader.Read(p)
	if n > 0 {
		ds.Total += int64(n)
		ds.Progress = float64(ds.Total) / float64(ds.Length) * float64(100)

		fmt.Printf("Downloading %s..........%.2f%%\r", ds.Filename, ds.Progress)
	}

	return n, err
}

var (
	MixChan  chan int
	Mixes    []Mix
	Dist     string
	Source   string
	Username string
	Password string
)

type SipClient struct {
	Client *http.Client
}

func NewSipClient(username string, password string) (*SipClient, error) {
	sc := SipClient{}

	cj, err := cookiejar.New(nil)
	if err != nil {
		return &sc, err
	}

	sc.Client = &http.Client{Jar: cj}

	resp, err := sc.Client.PostForm(
		"https://elixirsips.dpdcart.com/subscriber/login?__dpd_cart=45a48c06-43ab-4ab6-ba84-74fd0b10f85e",
		url.Values{
			"username": {username},
			"password": {password},
		})
	resp.Body.Close()

	return &sc, nil
}

func (m *Mix) Download(client *http.Client, dist string) {
	fileFullPath := fmt.Sprintf("%s/%s", dist, m.Filename)
	file, _ := os.Create(fileFullPath)
	defer file.Close()

	url := fmt.Sprintf("https://elixirsips.dpdcart.com/subscriber/download?file_id=%s", m.ID)
	response, err := client.Get(url)
	if err != nil {
		fmt.Printf("Failed to download %s", m.Filename)
	}

	defer response.Body.Close()
	status := &DownloadStatus{Filename: m.Filename, Reader: response.Body, Length: response.ContentLength}

	body, _ := ioutil.ReadAll(status)
	ioutil.WriteFile(fileFullPath, body, 0666)
	fmt.Printf("Finished Downloading %s", m.Filename)
}

func GetMixes(filePath string) ([]Mix, error) {
	ms := make([]Mix, 0)

	dat, err := ioutil.ReadFile(filePath)
	if err != nil {
		return ms, err
	}

	content := string(dat)

	re, err := regexp.Compile(`<li><a href="https://elixirsips.dpdcart.com/subscriber/download\?file_id=(?P<FileID>.*)">(?P<Filename>.*)</a></li>`)
	if err != nil {
		return ms, err
	}

	matches := re.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		filename := match[2]
		id := match[1]
		if strings.HasSuffix(filename, ".mkv") {
			continue
		}

		mix := Mix{
			Filename: filename,
			ID:       id,
		}
		ms = append(ms, mix)
	}

	return ms, nil
}

func init() {
	MixChan = make(chan int, 6)
	flag.Parse()

	args := flag.Args()

	if len(args) != 4 {
		log.Fatal("Should have four args - source, dist, username and password")
	}
	Source = args[0]
	Dist = args[1]
	Username = args[2]
	Password = args[3]
}

func Download(client *http.Client, ms []Mix) {
	for _, mix := range ms {
		MixChan <- 1
		go func() {
			fmt.Println("")
			mix.Download(client, Dist)
			<-MixChan
		}()
	}
}

func main() {
	fmt.Println(Dist)
	fmt.Println(Source)

	mixes, err := GetMixes(Source)
	if err != nil {
		log.Fatal(err)
	}

	client, err := NewSipClient(Username, Password)
	if err != nil {
		log.Fatal(err)
	}

	Download(client.Client, mixes)

	var over string
	fmt.Printf("Press Enter to continue...")
	fmt.Scanf("%s", &over)
}
