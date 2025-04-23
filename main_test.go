package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cohesion-org/deepseek-go"
	"github.com/gocolly/colly"
	"github.com/tiktoken-go/tokenizer"
)

type SomeMethod struct {
	Bookmark string
	Text     string
}

func (s SomeMethod) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Bookmark string
		Text     string
	}{
		Bookmark: s.Bookmark,
		Text:     s.Text,
	})
}

func TestInit(t *testing.T) {
	// delete tmpdir when test run,
	os.RemoveAll(tmpDir.Path)
}

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
	t.Skip()
	tokenize, err := checkToken("hello world")
	fmt.Println(tokenize.OriginalText)
	fmt.Println(err)
}

func TestIfCondition(t *testing.T) {
	t.Skip()
	someText := ""
	if len(someText) == 0 {
		fmt.Println("empty")
	}
}

func TestTiktoken(t *testing.T) {
	t.Skip()
	text := "hello world"
	enc, err := tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		t.Error(err)
	}
	// this should print a list of token ids
	ids, _, _ := enc.Encode(text)
	fmt.Println(ids)
	// this should print the original string back
	text, _ = enc.Decode(ids)
	fmt.Println(text)
}

func TestChannel(t *testing.T) {
	addition := func(a, b int) int {
		return a + b
	}
	c := make(chan int)
	go func() {
		c <- addition(1, 2)
	}()
	fmt.Println(<-c)
}

func TestChannelIntegration(t *testing.T) {
	var err error
	fullText := "lorem ipsum"
	tokenizeChannel := make(chan EncodedResponse)
	go func() {
		val, err2 := checkTokenv2(fullText)
		tokenizeChannel <- val
		err = err2
	}()
	// tokenize, err := checkTokenv2(fullText)
	if err != nil {
		fmt.Println(err)
	}
	tokenize := <-tokenizeChannel
	fmt.Println(tokenize)
}

func TestPurposefullyPanic(t *testing.T) {
	t.Skip()
	funcTest := func(a, b int) int {
		return a / b
	}
	result := funcTest(10, 0)
	fmt.Println(result)
}

func TestJsonSave(t *testing.T) {
	fileName := "test.json"
	data := []string{"hello", "world"}
	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(fileName, jsonData, 0644)
	if err != nil {
		t.Fatal(err)
	}
}

func TestJsonLoad(t *testing.T) {
	readJson := func(filename string) ([]string, error) {
		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		var result []string
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, err
		}
		return result, nil
	}
	data, err := readJson("test.json")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data[0])
	fmt.Println(data[1])
}

func TestJsonLoadNotExist(t *testing.T) {
	readJson := func(filename string) ([]string, error) {
		data, err := os.ReadFile(filename)
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("file not exist")
			return []string{}, nil
		}
		var result []string
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, err
		}
		fmt.Println("file found")
		return result, nil
	}
	_, err := readJson("someJson.json")
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteTMPFolders(t *testing.T) {
	deleteTempFolders([]string{".tmp3/", ".tmp4/"})
}

func TestDeepseekJson(t *testing.T) {
	t.Skip()
	type deepseekOutput struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	type Book struct {
		ISBN            string `json:"isbn"`
		Title           string `json:"title"`
		Author          string `json:"author"`
		Genre           string `json:"genre"`
		PublicationYear int    `json:"publication_year"`
		Available       bool   `json:"available"`
	}

	type Books struct {
		Books []Book `json:"books"`
	}
	client := deepseek.NewClient(os.Getenv("DEEPSEEK_API_KEY"))
	// systemPrompt := `Provide blog post in JSON format.
	// Please provide the JSON in the following format example: {"title": "How to be healthy", "content": "to be healthy you can try do some upper exercises"}`
	// userPrompt := "a blog post about health for strengthening lower body, please return the json format"
	prompt := `Provide a blogpost details in JSON format. 
	Please provide the JSON in the following format example: {"title": "How to be healthy", "content": "to be healthy you can try do some upper exercises"}`
	ctx := context.Background()
	resp, err := client.CreateChatCompletion(ctx, &deepseek.ChatCompletionRequest{
		Model: "deepseek-chat", // Or another suitable model
		Messages: []deepseek.ChatCompletionMessage{
			// {Role: deepseek.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: deepseek.ChatMessageRoleUser, Content: prompt},
		},
		JSONMode: true,
	})
	if err != nil {
		t.Fatalf("Failed to create chat completion: %v", err)
	}
	if resp == nil || len(resp.Choices) == 0 {
		t.Fatal("No response or choices found")
	}
	log.Printf("Response: %s", resp.Choices[0].Message.Content)
	extractor := deepseek.NewJSONExtractor(nil)
	var output deepseekOutput
	// var books Books
	if err := extractor.ExtractJSON(resp, &output); err != nil {
		t.Fatalf("Failed to extract JSON: %v", err)
	}
	fmt.Println(output.Content)
	fmt.Println("--------------------")
	fmt.Println(output.Title)
}

func TestGetDeepseekAllModels(t *testing.T) {
	t.Skip()
	func() {
		client := deepseek.NewClient(os.Getenv("DEEPSEEK_API_KEY"))
		ctx := context.Background()
		models, err := deepseek.ListAllModels(client, ctx)
		if err != nil {
			t.Errorf("Error listing models: %v", err)
		}
		fmt.Printf("\n%+v\n", *models)
	}()
}
