package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Submission struct {
	ProblemType string `json:"problemType"`
	Details     []any  `json:"details"`
}

//go:embed "icon.ico"
var iconData []byte

func main() {
	myApp := app.New()
	icon := fyne.NewStaticResource("icon.ico", iconData)
	//if err == nil {
	myApp.SetIcon(icon)
	//} else {
	//	fmt.Println(err)
	//}
	window := myApp.NewWindow("Let's Fuck up PTA !!!!")

	var compiler string
	// 创建界面元素
	cookieEntry := widget.NewEntry()
	urlEntry := widget.NewEntry()
	compilerSelector := widget.NewSelect([]string{"G++", "Clang++"}, func(selected string) {
		switch selected {
		case "G++":
			compiler = "GXX"
		case "Clang++":
			compiler = "CLANGXX"
		default:
			compiler = "GXX"
		}
	})
	contentEntry := widget.NewMultiLineEntry()
	cookieEntry.SetPlaceHolder("Cookie")
	contentEntry.SetPlaceHolder("program")
	urlEntry.SetPlaceHolder("url")
	compilerSelector.SetSelected("G++")
	helpDetail := widget.NewRichTextFromMarkdown(
		"url就是你正在做的这道题的连接，浏览器顶部那个\n\ncookie需要你在登录pta的情况下打开浏览器工具，找到（更多工具-）应用-储存-Cookie里面一个名叫PTASession的粘贴过来，注意这个Cookie可能经常变化，应当尝试重新复制粘贴过来\n\n[一图流（chrome为例）](https://cloudreve.zsgbp.site/f/lrcj/PixPin_2025-03-26_18-35-25.png)")

	helpDetail.Wrapping = fyne.TextWrapWord
	help := widget.NewAccordion(
		&widget.AccordionItem{
			Title:  "如何使用",
			Detail: helpDetail,
		},
	)

	sendBtn := widget.NewButton("提交", func() {
		go func() {
			cookie := "PTASession=" + cookieEntry.Text
			content := contentEntry.Text
			parsedURL, err := url.Parse(urlEntry.Text)
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			problemSetProblemId := parsedURL.Query().Get("problemSetProblemId")
			path := parsedURL.Path
			prefix := "problem-sets/"
			start := strings.Index(path, prefix)
			start += len(prefix)
			end := strings.Index(path[start:], "/")
			if start == -1 || end == -1 {
				dialog.ShowError(errors.New("url有误，应当像这样：\n https://pintia.cn/problem-sets/1900019993728618496/exam/problems/type/7?problemSetProblemId=1900019993753784320&page=0"), window)
				return
			}
			problemSetId := path[start : start+end]
			client := &http.Client{}
			req, err := http.NewRequest("GET", "https://pintia.cn/api/problem-sets/"+problemSetId+"/exams", nil)
			req.Header.Set("Cookie", cookie)
			req.Header.Set("Accept", "application/json;charset=UTF-8")
			resp, err := client.Do(req)
			defer resp.Body.Close()
			fmt.Println("Content-Encoding:", resp.Header.Get("Content-Encoding"))
			fmt.Println("Content-Type:", resp.Header.Get("Content-Type"))
			//reader, err := gzip.NewReader(resp.Body)
			body, err := io.ReadAll(resp.Body)
			fmt.Println(string(body))
			var data map[string]interface{}
			var examId string
			err = json.Unmarshal(body, &data)

			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			if exam, ok := data["exam"].(map[string]interface{}); ok {
				if examId, ok = exam["id"].(string); ok {
				} else {
					dialog.ShowError(errors.New("未能找到exam id"), window)
					return
				}
			} else {
				dialog.ShowError(errors.New("未能找到exam"), window)
				return
			}
			// 替换换行符
			replaced := strings.ReplaceAll(content, "\r\n", "\n")
			replaced = strings.ReplaceAll(replaced, "\n", `\n`)
			body, err = json.Marshal(Submission{
				ProblemType: "PROGRAMMING",
				Details: []any{
					map[string]any{
						"problemId":           "0",
						"problemSetProblemId": problemSetProblemId,
						"programmingSubmissionDetail": map[string]any{
							"program":  replaced,
							"compiler": compiler,
						},
					},
				},
			})
			req, err = http.NewRequest("POST", "https://pintia.cn/api/exams/"+examId+"/exam-submissions", bytes.NewBuffer(body))
			if err != nil {
				dialog.ShowError(err, window)
				return
			}

			req.Header.Set("Cookie", cookie)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json;charset=UTF-8")

			resp, err = client.Do(req)
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			defer resp.Body.Close()

			ss, err := io.ReadAll(resp.Body)
			err = json.Unmarshal(ss, &data)
			if submissionId, ok := data["submissionId"].(string); ok {
				for i := 0; i < 10; i++ {
					req, err = http.NewRequest("GET", "https://pintia.cn/api/exams/"+examId+"/submissions/"+submissionId+"?", nil)
					req.Header.Set("Cookie", cookie)
					req.Header.Set("Accept", "application/json;charset=UTF-8")
					resp, err = client.Do(req)
					defer resp.Body.Close()
					body, err = io.ReadAll(resp.Body)
					fmt.Println(string(body))
					err = json.Unmarshal(body, &data)
					if submission, ok := data["submission"].(map[string]interface{}); ok {
						if status, ok := submission["status"].(string); ok && status != "WAITING" && status != "JUDGING" {
							score, _ := submission["score"].(float32)
							dialog.ShowInformation("fuck up pta:", "status:"+status+"\nscore:"+fmt.Sprintf("%.2f", score), window)
							return
						}
					}
					if i == 9 {
						dialog.ShowError(errors.New("已经尝试查询10次未能得到结果，请自行查询"), window)
						return
					}
					time.Sleep(time.Second * 3)
				}
			}

			if err != nil {
				dialog.ShowError(err, window)
				return
			}
		}()

	})

	// 布局设置
	content := container.NewVBox(
		widget.NewLabel("Settings:"),
		urlEntry,
		cookieEntry,
		compilerSelector,
		contentEntry,
		sendBtn,
		help,
		widget.NewHyperlink("github", &url.URL{
			Scheme: "https",
			Host:   "github.com",
			Path:   "/NIA-sai/fkpta",
		}),
	)

	window.SetContent(content)
	window.Resize(fyne.NewSize(400, 300))
	window.SetFixedSize(false)
	window.ShowAndRun()
}
