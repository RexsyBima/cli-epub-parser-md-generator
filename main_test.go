package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gocolly/colly"
)

func TestScrapeHTML(t *testing.T) {
	t.Skip("skipping for now...")
	var Subchapters = []Subchapter{}
	c := colly.NewCollector()
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})
	c.OnHTML("section[data-pdf-bookmark][data-type='sect1']", func(e *colly.HTMLElement) {
		Subchapters = append(Subchapters, NewSubchapter(e.Attr("data-pdf-bookmark"), e.Text))
	})
	err := c.Visit("http://127.0.0.1:8000/test_data/ch07.html")
	c.OnError(func(r *colly.Response, err error) {
		t.Error("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})
	if err != nil {
		t.Fatal(err)
	}
	var fullText string
	for _, subchapter := range Subchapters {
		fullText += subchapter.Text
	}
	tokenize, err := checkToken(fullText)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(tokenize.OriginalText)
	fmt.Println(tokenize.TokenLength)
}

func TestCheckToken(t *testing.T) {
	t.Skip()
	url := "http://127.0.0.1:8080/encode"
	// Text with newlines
	textWithNewlines := "hello\nworld\nthis has\nmultiple lines"
	// Create request payload
	requestBody := map[string]string{
		"text": textWithNewlines,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	// Define the response struct
	type EncodedResponse struct {
		OriginalText string `json:"original_text"`
		EncodedText  []int  `json:"encoded_text"`
		TokenLength  int    `json:"token_length"`
	}
	// Parse the response into the struct
	var result EncodedResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		t.Fatal("Error parsing JSON response:", err)
	}
	fmt.Printf("Original text: %s\n", result.OriginalText)
	fmt.Printf("Encoded text: %v\n", result.EncodedText)
	fmt.Printf("Token length: %d\n", result.TokenLength)
}

func TestScanFolder(t *testing.T) {
	t.Skip("skipping for now...")
	filePath, err := scanHTMLFiles("test_data")
	if err != nil {
		t.Fatal(err)
	}
	for i, file := range filePath {
		fmt.Printf("%d: %s\n", i+1, file)
	}
}

func TestMkdir(t *testing.T) {
	t.Skip("skipping for now...")
	err := os.Mkdir(outputTestPath, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(outputTestPath)
}

func TestSaveTextMD(t *testing.T) {
	t.Skip()
	output := "some text"
	fmt.Println("Response:", output)
	filename := fmt.Sprintf("%s/%s.md", outputTestPath, "test")
	file, err := os.Create(filename)
	if err != nil {
		os.Mkdir(outputTestPath, 0755)
		file, err = os.Create(filename)
	}
	// os.ReadDir()
	_, err = file.WriteString(string(output))
	if err != nil {
		panic(err)
	}
	defer file.Close()
	defer os.Remove(outputTestPath)
}

func TestExtractEpub(t *testing.T) {
	t.Skip()
	ExtractEpub("book1.epub", ".temp")
}

func TestScanHTML(t *testing.T) {
	t.Skip()
	type HTMLFile struct {
		Path string
	}
	ScanHTMLFiles := func(rootDir string) ([]HTMLFile, error) {
		var htmlfiles []HTMLFile
		err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if strings.HasSuffix(strings.ToLower(path), ".html") || strings.HasSuffix(strings.ToLower(path), ".htm") {
				htmlfiles = append(htmlfiles, HTMLFile{Path: path})
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		return htmlfiles, nil
	}
	files, err := ScanHTMLFiles(".temp")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		fmt.Println(file.Path)
	}
}

func TestExtractTitle(t *testing.T) {
	t.Skip()
	data := ".temp/OEBPS/toc01.html"
	title := strings.Split(data, "/")
	fmt.Println(strings.Split(data, "/")[len(title)-1])
}

func TestUniqueFolderName(t *testing.T) {
	t.Skip()
	currentDir, _ := os.Getwd()
	dir, err := os.MkdirTemp(currentDir, ".tmp")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(dir)
	// defer os.RemoveAll(dir)
}

func TestBook2(t *testing.T) {
	t.Skip()
	var Subchapters = []Subchapter{}
	c := colly.NewCollector()
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})
	c.OnHTML("body", func(e *colly.HTMLElement) {
		Subchapters = append(Subchapters, NewSubchapter(e.Attr("data-pdf-bookmark"), e.Text))
	})
	bookName := "book1.epub"
	err := ExtractEpub(bookName, ".tmp")
	filePath, err := ScanHTMLFiles(".tmp")
	if err != nil {
		fmt.Println(err)
		return
	}
	for i, file := range filePath {
		texts := strings.Split(file.Path, "/")
		fmt.Printf("%d: %s\n", i+1, texts[len(texts)-1])
	}
	fmt.Println("choose a chapter based on number")
	chapterNumber := 10
	err = c.Visit("http://127.0.0.1:8000/" + filePath[chapterNumber-1].Path)
	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Request URL: %v | failed with response %v", r.Request.URL, err)
	})
	var fullText string
	for _, subchapter := range Subchapters {
		fullText += subchapter.Text
	}
	res, err := checkToken(fullText)
	fmt.Println(res.OriginalText)
	fmt.Println(res.TokenLength)
}

func TestGetRelativePath(t *testing.T) {
	t.Skip()
	fmt.Println(tmpDir.RelativePath)
	tmpDir.SetRelativePath()
	fmt.Println(tmpDir.RelativePath)
	fmt.Println(*tmpDir.RelativePath)
}

func TestScanHTMLFiles2(t *testing.T) {
	t.Skip()
	var routes []string
	bookName := "progit.epub"
	err := ExtractEpub(bookName, tmpDir.Path)
	if err != nil {
		fmt.Println(err)
		return
	}
	files, _ := ScanHTMLFiles(tmpDir.Path)
	for _, file := range files {
		startIndex := strings.Index(file.Path, "/.tmp")
		if startIndex != -1 {
			subPath := file.Path[startIndex:]
			routes = append(routes, subPath)
		}
	}
	fmt.Println(routes)
	defer os.RemoveAll(tmpDir.Path)
}

func TestSplitText(t *testing.T) {
	t.Skip()
	fullPath := "/home/rexsybimatw/go/cli-epub-parser-md-generator/.tmp3823640299/EPUB/toc.xhtml"
	// Find index where ".tmp" starts
	startIndex := strings.Index(fullPath, "/.tmp")
	if startIndex != -1 {
		subPath := fullPath[startIndex:]
		fmt.Println(subPath)
	} else {
		fmt.Println("'.tmp' not found in path")
	}
}

func TestCheckTokenv2(t *testing.T) {
	tokenize, err := checkToken("hello world")
	fmt.Println(tokenize.OriginalText)
	fmt.Println(err)
}
