package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-gomail/gomail"
	"github.com/opesun/goquery"
	"github.com/yudeguang/gather"
)

const (
	NO = "3|三"
)

var mapDT map[string]string
var taskOK bool

func main() {
	taskOK = false
	monitorPeriod := time.NewTicker(1 * time.Minute)
	defer monitorPeriod.Stop()
	go run(monitorPeriod.C)

	for {
		time.Sleep(30 * time.Second)
		if taskOK {
			monitorPeriod.Stop()
			fmt.Println("爬到了字幕")
			break
		}
	}

}

func run(c <-chan time.Time) {
	fmt.Println("run...")
	for {
		select {
		case <-c:
			fileName, err := climb(NO)
			if err != nil {
				fmt.Println(err)
				continue
			}
			if fileName == "" {
				fmt.Println(time.Now(), "重新爬")
				continue
			}
			err = sendMail(fileName)
			if err != nil {
				fmt.Println(err)
				continue
			}
			taskOK = true
		}
	}
}

func climb(id string) (string, error) {
	matchStr := "(权力的游戏)+.*(第[八|8]季)+.*(第[" + id + "]集)+.*(中|简)+"
	// 检查有无最新中文字幕
	var url = "https://www.zimuku.la"
	ga := gather.NewGather("chrome", false)
	html, returnedURL, err := ga.Get(url, "")
	p, err := goquery.ParseString(html)
	if err != nil {
		return "", err
	}
	t := p.Find("a")
	var url2 string
	for i := 0; i < t.Length(); i++ {
		match1, _ := regexp.MatchString(matchStr, t.Eq(i).Attr("title"))
		if match1 {
			url2 = t.Eq(i).Attr("href")
			if url2 != "" {
				break
			}
		}
	}
	if url2 == "" {
		return "", nil
	}
	// 获取下载页面地址
	oldurl := url
	url += "/dld"
	tmpi := strings.LastIndex(url2, "/")
	url += string([]byte(url2)[tmpi:len(url2)])
	fmt.Println(url)
	time.Sleep(1 * time.Second)
	// 进入下载页面
	html, returnedURL, err = ga.Get(url, oldurl)
	p, err = goquery.ParseString(html)
	if err != nil {
		return "", err
	}
	t = p.Find("a")
	oldurl = url
	url = "https://www.zimuku.la" + t.Eq(4).Attr("href")
	fmt.Println(url)
	html, returnedURL, err = ga.Get(url, oldurl)
	fmt.Println(returnedURL)
	filename, err := downloadFile(returnedURL, "./")
	if err != nil {
		return "", err
	}
	return filename, nil
}

func downloadFile(url string, dir string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()               //千万要关闭链接，不然会造成资源泄露（就是为了防止死循环）。
	if resp.StatusCode != http.StatusOK { //校验状态码
		fmt.Println(url, resp.StatusCode)
	}
	flysnowRegexp := regexp.MustCompile("\".*\"")
	params := flysnowRegexp.FindStringSubmatch(resp.Header["Content-Disposition"][0])
	var fileName string
	for _, param := range params {
		fileName = param[1 : len(param)-1]
	}
	fullName := filepath.Join(dir, fileName) //这是将下载到文件存放在指定到目录中去。
	f, err := os.Create(fullName)            //创建文件。
	if err != nil {
		return "", err
	}
	io.Copy(f, resp.Body) //将文件到内容拷贝到创建的文件中。
	fmt.Printf("已下载文件至：\033[31;1m%s\033[0m\n", fullName)
	//defer os.RemoveAll(file_name) //删除文件。
	return fullName, nil
}

func sendMail(filename string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", m.FormatAddress("305584612@qq.com", "子书"))
	m.SetHeader("To", // 收件人
		m.FormatAddress("zishuzy@qq.com", "子书"),
		m.FormatAddress("358755265@qq.com", "卢家付"),
	)
	m.SetHeader("Subject", "Gomail") // 主题
	m.SetBody("text/html", "字幕")     // 正文
	m.Attach("./" + filename)
	d := gomail.NewDialer("smtp.qq.com", 465, "305584612@qq.com", "****************")
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	err := d.DialAndSend(m)
	return err
}
