package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/bogem/id3v2"
	"github.com/go-resty/resty/v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// 清理文件名
func sanitizeFileName(name string) string {
    // Windows文件系统不允许的字符映射
    replaceMap := map[string]string{
        "<":  "＜",
        ">":  "＞",
        ":":  "：",
        "\"": "＂",
        "/":  "／",
        "\\": "＼",
        "|":  "｜",
        "?":  "？",
        "*":  "＊",
    }
    
    // 替换Windows不允许的字符
    for old, new := range replaceMap {
        // 如果被替换的字符前后有空格，一并去除
        name = strings.ReplaceAll(name, " " + old + " ", new)
        name = strings.ReplaceAll(name, " " + old, new)
        name = strings.ReplaceAll(name, old + " ", new)
        name = strings.ReplaceAll(name, old, new)
    }
    
    // 移除首尾空格
    name = strings.TrimSpace(name)
    return name
}

func fileExists(filename string) (ok bool, err error) {
	var fi os.FileInfo
	if fi, err = os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			err = nil
		} else {
			return
		}
	} else if fi.IsDir() {
		err = errors.New("unexpected directory:" + filename)
	} else {
		ok = true
	}
	return
}

func main() {
	var err error
	defer func(err *error) {
		if *err != nil {
			log.Println("exited with error:", (*err).Error())
			os.Exit(1)
		} else {
			log.Println("exited")
		}
	}(&err)

	var (
		optAlbumId string
		optAuth    string
	)

	flag.StringVar(&optAlbumId, "album", "", "有声书编号，可以在浏览器链接中找到")
	flag.StringVar(&optAuth, "auth", "", "登录机核网站网页端后，获取到的用户认证 Cookie，可以在浏览器开发工具中找到")
	flag.Parse()

	if optAlbumId == "" || optAuth == "" {
		err = errors.New("missing arguments")
		return
	}

	var OutputDirectory = fmt.Sprintf("output-%s", optAlbumId)

	_ = os.MkdirAll(OutputDirectory, 0755)

	client :=
		resty.New().
			SetHostURL("https://www.gcores.com").
			SetHeader("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15").
			SetHeader("Referer", "https://www.gcores.com").
			SetDebug(true)

	freeAudioClient := resty.New().
		SetHostURL("https://alioss.gcores.com/uploads/audio").
		SetHeader("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15").
		SetHeader("Referer", "https://www.gcores.com").
		SetDebug(true)

	protectedAudioClient :=
		resty.New().
			SetRedirectPolicy(resty.FlexibleRedirectPolicy(5)).
			SetCookie(&http.Cookie{
				Name:    "auth_token",
				Value:   optAuth,
				Domain:  ".gcores.com",
				Path:    "/",
				Expires: time.Now().Add(time.Hour * 24 * 365),
			}).
			SetHostURL("https://www.gcores.com/gapi/v1/medias/protected/radios/").
			SetHeader("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15").
			SetHeader("Referer", "https://www.gcores.com").
			SetDebug(true)

	coverClient := resty.New().
		SetHostURL("https://image.gcores.com").
		SetHeader("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15").
		SetHeader("Referer", "https://www.gcores.com").
		SetDebug(true)

	type Entity struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Title     string `json:"title"`
			Author    string `json:"author"`
			Cover     string `json:"cover"`
			Audio     string `json:"audio"`
			MediaType string `json:"media-type"`
		} `json:"attributes"`
		Relationships struct {
			Media struct {
				Data struct {
					Type string `json:"type"`
					ID   string `json:"id"`
				} `json:"data"`
			} `json:"media"`
			PublishedAudiobooks struct {
				Data []struct {
					ID   string `json:"id"`
					Type string `json:"type"`
				} `json:"data"`
			} `json:"published-audiobooks"`
		} `json:"relationships"`
	}

	type EntityResponse struct {
		Data     Entity   `json:"data"`
		Included []Entity `json:"included"`
	}

	var res EntityResponse
	if _, err = client.R().
		SetResult(&res).
		SetQueryParam("include", "published-audiobooks,published-audiobooks.media").
		SetPathParam("album_id", optAlbumId).
		Get("/gapi/v1/albums/{album_id}"); err != nil {
		return
	}
	data, _ := json.MarshalIndent(res, "", "  ")
	log.Printf("%s", data)

	if !strings.HasSuffix(res.Data.Attributes.Cover, ".jpg") {
		err = errors.New("cover is not a jpg")
		return
	}

	coverFile := filepath.Join(OutputDirectory, "cover.jpg")
	coverExisted := false
	if coverExisted, err = fileExists(coverFile); err != nil {
		return
	}

	if !coverExisted {
		if _, err = coverClient.R().SetOutput(coverFile).Get(res.Data.Attributes.Cover); err != nil {
			return
		}
	}

	var coverData []byte
	if coverData, err = ioutil.ReadFile(coverFile); err != nil {
		return
	}

	for i, radioRel := range res.Data.Relationships.PublishedAudiobooks.Data {
		log.Println("Working on:", radioRel.ID)

		var radio Entity
		for _, radio = range res.Included {
			if radio.ID == radioRel.ID && radio.Type == radioRel.Type {
				goto found
			}
		}

		log.Printf("Error: missing included:%s#%s, skipping...", radioRel.Type, radioRel.ID)
		continue

	found:
		log.Println("Found Radio:", radio.Attributes.Title)

		mediaRel := radio.Relationships.Media.Data
		var media Entity
		for _, media = range res.Included {
			if media.ID == mediaRel.ID && media.Type == mediaRel.Type {
				goto found2
			}
		}

		log.Printf("Error: missing included:%s#%s, skipping...", mediaRel.Type, mediaRel.ID)
		continue

	found2:
		log.Println("Found media:", media.Attributes.MediaType, media.Attributes.Author)

		audioFile := filepath.Join(OutputDirectory, fmt.Sprintf(
			"%s-%03d-%s%s",
			sanitizeFileName(res.Data.Attributes.Title),
			i+1,
			sanitizeFileName(radio.Attributes.Title),
			filepath.Ext(media.Attributes.Audio),
		))

		audioExisted := false

		if audioExisted, err = fileExists(audioFile); err != nil {
			log.Printf("Error checking file existence for %s: %v, skipping...", audioFile, err)
			continue
		}

		if !audioExisted {
			if media.Attributes.MediaType == "protected_audio" {
				if _, err = protectedAudioClient.R().SetOutput(audioFile).Get(radio.ID); err != nil {
					log.Printf("Error downloading protected audio %s: %v, skipping...", audioFile, err)
					continue
				}
			} else if media.Attributes.MediaType == "audio" {
				if _, err = freeAudioClient.R().SetOutput(audioFile).Get(media.Attributes.Audio); err != nil {
					log.Printf("Error downloading free audio %s: %v, skipping...", audioFile, err)
					continue
				}
			} else {
				log.Printf("Error: unknown media type:%s, skipping...", media.Attributes.MediaType)
				continue
			}
		}

		var tag *id3v2.Tag
		if tag, err = id3v2.Open(audioFile, id3v2.Options{Parse: false}); err != nil {
			log.Printf("Error opening audio file for ID3 tags %s: %v, skipping...", audioFile, err)
			continue
		}

		log.Println("Setting ID3 Info:", audioFile)

		tag.DeleteAllFrames()
		tag.SetDefaultEncoding(id3v2.EncodingUTF8)
		tag.SetGenre("Audio Book")
		tag.SetArtist(res.Data.Attributes.Author)
		tag.SetAlbum(res.Data.Attributes.Title)
		tag.SetTitle(radio.Attributes.Title)
		tag.AddAttachedPicture(id3v2.PictureFrame{
			Encoding:    tag.DefaultEncoding(),
			MimeType:    "image/jpeg",
			PictureType: 0x03,
			Picture:     coverData,
		})
		tag.AddTextFrame("TRCK", tag.DefaultEncoding(), strconv.Itoa(i+1))
		if err = tag.Save(); err != nil {
			log.Printf("Error saving ID3 tags for %s: %v, skipping...", audioFile, err)
			continue
		}
	}
	err = nil // 确保主程序不会以错误状态退出
}
