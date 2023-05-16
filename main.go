package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/abadojack/whatlanggo"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

var port int
var apiKey string
var host string

func init() {
	const (
		defaultPort = 1188
		usage       = "set up the port to listen on"
	)

	flag.IntVar(&port, "port", defaultPort, usage)
	flag.IntVar(&port, "p", defaultPort, usage)
	flag.StringVar(&apiKey, "key", "", "set up the api key")
	flag.StringVar(&host, "host", "", "set up the host to listen on")

	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

type TranslateRequest struct {
	ApiKey     string `json:"apikey"`
	Text       string `json:"text"`
	SourceLang string `json:"source_lang"`
	TargetLang string `json:"target_lang"`
}

func getRandomNumber() int64 {
	rand.Seed(time.Now().Unix())
	num := rand.Int63n(99999) + 8300000
	return num * 1000
}

type ResData struct {
	TransText  string `json:"text"`
	SourceLang string `json:"source_lang"`
	TargetLang string `json:"target_lang"`
}

func main() {
	// parse flags
	flag.Parse()

	if apiKey == "" {
		log.Println("No api key found, please use -key to set up the api key")
		return
	}

	// display information
	fmt.Printf("DeepL proxy has been successfully launched! Listening on %s:%v\n", host, port)

	// create a random id
	id := getRandomNumber()

	// set release mode
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(cors.Default())

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"code":    200,
			"message": "DeepL Free API, Made by sjlleo and missuo. Go to /translate with POST. http://github.com/OwO-Network/DeepLX",
		})
	})

	r.POST("/translate", func(c *gin.Context) {
		reqj := ResData{}
		c.BindJSON(&reqj)

		sourceLang := reqj.SourceLang
		targetLang := reqj.TargetLang
		translateText := reqj.TransText
		if sourceLang == "" {
			lang := whatlanggo.DetectLang(translateText)
			deepLLang := strings.ToUpper(lang.Iso6391())
			sourceLang = deepLLang
		}
		if targetLang == "" {
			targetLang = "EN"
		}

		if translateText == "" {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    http.StatusNotFound,
				"message": "No Translate Text Found",
			})
		} else {
			url := "https://api.openl.club/services/deepl/translate"

			postData := TranslateRequest{
				ApiKey:     apiKey,
				Text:       translateText,
				SourceLang: strings.ToLower(sourceLang),
				TargetLang: strings.ToLower(targetLang),
			}

			postBytes, _ := json.Marshal(postData)
			reader := bytes.NewReader(postBytes)
			request, err := http.NewRequest("POST", url, reader)
			request.Header.Set("Content-Type", "application/json")
			if err != nil {
				log.Println(err)
				return
			}

			client := &http.Client{}
			resp, err := client.Do(request)
			if err != nil {
				log.Println(err)
				return
			}

			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			res := gjson.ParseBytes(body)

			// display response
			// fmt.Println(res)
			errorCode := res.Get("error.code").String()
			if errorCode == "-32600" || errorCode == "-32503" {
				log.Println(res.Get("error").String())
				c.JSON(http.StatusNotAcceptable, gin.H{
					"code":    http.StatusNotAcceptable,
					"message": "Invalid targetLang",
				})
				return
			}

			if resp.StatusCode == http.StatusTooManyRequests {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"code":    http.StatusTooManyRequests,
					"message": "Too Many Requests",
				})
			} else {
				var alternatives []string
				res.Get("result.texts.0.alternatives").ForEach(func(key, value gjson.Result) bool {
					alternatives = append(alternatives, value.Get("alternative").String())
					return true
				})
				c.JSON(http.StatusOK, gin.H{
					"code":         http.StatusOK,
					"id":           id,
					"data":         res.Get("result").String(),
					"alternatives": alternatives,
				})
			}
		}
	})
	// by default, listen and serve on 0.0.0.0:1188
	r.Run(fmt.Sprintf("%s:%v", host, port))
}
