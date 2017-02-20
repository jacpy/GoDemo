package  main

import (
	"log"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"net/http"
	"image/png"
	"image"
	"image/color"
	"os/exec"
)

func parseHtml(url string) *[]string {
	p, err := goquery.NewDocument(url)
	if err != nil {
		panic(err)
	}

	fileInfoArr, err := ioutil.ReadDir("html")
	log.Println(fileInfoArr, err)
	if len(fileInfoArr) == 0 || err != nil && os.IsNotExist(err) {
		os.Mkdir("html", os.ModePerm)
		log.Println(err)
	}

	log.Println(fileInfoArr)
	current, _ := os.Getwd()
	dir := current + "/html"
	log.Println("----p---->")
	log.Println(p)
	log.Println("<----p----")
	files := make([]string, 0, 20)
	p.Find("#ID_bbs_subjects_p1 li").Each(func(idx int, selection *goquery.Selection) {
		href, exists := selection.Find(".fname").Attr("href")
		log.Println("exists: ", exists, "href: ", href, "idx: ", idx)
		//if exists && idx == 0 {
		if exists {
			s := strconv.Itoa(idx)
			files = append(files, replaceHtml(&href, dir, s))
		}
	})

	return &files
}

func replaceHtml(url *string, dir string, name string) string {
	log.Println("replaceHtml, url: ", *url, "dir: ", dir)
	p, err := goquery.NewDocument(*url)
	if err != nil {
		panic(err)
	}

	selection := p.Find("head title");
	selection.ReplaceWithHtml("<title>" + strings.Replace(selection.Text(), "_即时通讯网(52im.net) _即时通讯开发者社区!", "", 1) + "</title>")
	p.Find("head link").Each(func(idx int, s *goquery.Selection) {
		link, exists := s.Attr("href")
		if exists {
			linkName := getFileName(link)
			path := dir + "/" + linkName
			_, err = os.Stat(path)
			if err != nil && os.IsNotExist(err) {
				// 下载css文件
				log.Println("download css: ", link)
				downloadFile(relativePath(*url, link), dir + "/" + linkName)
			}

			replaceNode(s, "href", "./" + linkName)
		}
	})

	p.Find(".net52im_copy").Remove()
	p.Find("div.net52im_p_pageou p a").Remove()
	p.Find("div.net52im_p_pageji p a").Remove()
	p.Find("div.Part div a").RemoveAttr("href").RemoveAttr("title")
	p.Find("div.Part div a img").RemoveAttr("alt")
	p.Find("p.net52im_p_img").RemoveAttr("href").RemoveAttr("title")
	p.Find("#js_ift").Remove()
	p.Find("img").Each(func(idx int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		//tmpSrc := src
		if exists {
			path := dir + "/" + name
			_, e := os.Stat(path)
			if e != nil && os.IsNotExist(e) {
				os.Mkdir(path, os.ModePerm)
			}

			suffix := getFileName(src)
			path = path + "/" + suffix
			log.Println("path: ", path)
			aUrl := *url
			log.Println(aUrl)

			if downloadFile(relativePath(aUrl, src), path) != nil {
				log.Println("download image failed: ", src, ", dir: ", path)
			} else {
				// 图片下载成功，替换img标签的src属性值
				replaceNode(s, "src", "./" + name + "/" + suffix)
				// 去掉图片的水印
				if !strings.HasSuffix(path, ".jpg") {
					watermarkMask(path)
				}
			}
		}
	})

	content, err := p.Html()
	log.Println("----content---->")
	log.Println(content)
	log.Println("<----content----")
	file := dir + "/" + name + ".html"
	err = ioutil.WriteFile(file, []byte(content), os.ModePerm)
	if err != nil {
		panic(err)
	}

	return file
}

func replaceNode(s *goquery.Selection, attrName string, v string)  {
	nodes := s.Nodes
	if len(nodes) > 0 {
		attrs := nodes[0].Attr
		for idx, value := range attrs {
			if strings.Compare(attrName, value.Key) == 0 {
				attrs[idx].Val = v
				break
			}
		}
	}
}

/**
将../相对路径转换成绝对路径
 */
func relativePath(aUrl string, src string) string {
	for {
		// 将../与HTML链接拼出图片路径
		if strings.HasPrefix(src, "../") {
			log.Println(aUrl)
			if strings.HasSuffix(aUrl, "/") {
				aUrl = aUrl[:len(aUrl) - 1]
			}

			log.Println("trim suffix: ", aUrl)
			n := strings.LastIndex(aUrl, "/")
			aUrl = aUrl[:n]
			src = strings.Replace(src, "../", "", 1)
			log.Println(src, aUrl)
		} else {
			break
		}
	}

	if strings.HasSuffix(aUrl, "/") && strings.HasPrefix(src, "/") {
		return aUrl + src[1:]
	} else if !strings.HasSuffix(aUrl, "/") && !strings.HasPrefix(src, "/") {
		return aUrl + "/" + src
	}

	return aUrl + src
}

/**
 * 获取链接的文件名，包括扩展名
 */
func getFileName(url string) string {
	i := strings.LastIndex(url, "/")
	return url[i + 1:]
}

func downloadFile(url string, dir string) error {
	log.Println("downloadImage, url: ", url, ", dir: ", dir)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(dir, buf, os.ModePerm)
}

func watermarkMask(path string) error {
	log.Println("watermarkMash: ", path)
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	pic, suffix, err := image.Decode(f)
	log.Println(suffix)
	f.Close()
	if (err != nil) {
		panic(err)
	}

	bounds := pic.Bounds()
	width := bounds.Max.X
	height := bounds.Max.Y
	offsetX, offsetY := width / 2, height * 3 / 4
	rgba := image.NewNRGBA(image.Rect(0, 0, width, height))
	for i := 0; i < height; i++ {
		for j := 0; j < width; j++ {
			c := pic.At(j, i)
			r, g, b, a := c.RGBA()
			//rgba.Set(j, i, color.NRGBA{uint8(r), uint8(g), uint8(b), uint8(a)})
			//log.Println(r & 0xFF, g & 0xFF, b & 0xFF)
			rgba.Set(j, i, color.NRGBA{uint8(r), uint8(g), uint8(b), uint8(a)})
			if i > offsetY && j > offsetX {
				if r & 0xFF > 230 && g & 0xFF > 230 && b & 0xFF > 230 {
					rgba.Set(j, i, color.NRGBA{uint8(255), uint8(255), uint8(255), uint8(255)})
				}
			}
		}
	}

	f, err = os.OpenFile(path, os.O_RDWR | os.O_TRUNC, os.ModePerm)
	//f, err = os.Create("html/0/test.png")
	if err != nil {
		panic(err)
	}

	png.Encode(f, rgba)
	f.Close()
	return nil
}

func html2pdf(arr []string)  {
	arr = append(arr, "tcp.pdf")
	args := make([]string, len(arr) + 2)
	args[0] = "--zoom"
	args[1] = "4"
	old := args[2:]
	copy(old, arr)
	for _, value := range args {
		log.Println("arg: ", value)
	}

	cmd := exec.Command("wkhtmltopdf", args...)
	err := cmd.Run()
	if err != nil {
		panic(err)
	}

	log.Println("html2pdf finish.")
}

func main() {
	log.Println("---->main")
	//watermarkMask("html/0/52im_12.png")
	dir := parseHtml("http://www.52im.net/topic-tcpipvol1.html")
	for _, value := range *dir {
		log.Println("dir: ", value)
	}

	html2pdf(*dir)
}
