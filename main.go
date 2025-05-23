package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/cohesion-org/deepseek-go"
	"github.com/gocolly/colly"
	"github.com/joho/godotenv"
	"github.com/tiktoken-go/tokenizer"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

var systemPrompt string = fmt.Sprintf(`You are an AI transformation agent tasked with converting book texts about knowledge into a polished, engaging, and readable blog post. Your responsibilities include: - **Paraphrasing**: Transform the original caption text into fresh, original content while preserving the key information and insights. - **Structure**: Organize the content into a well-defined structure featuring a captivating introduction, clearly delineated subheadings in the body, and a strong conclusion. - **Engagement**: Ensure the blog post is outstanding by using a professional yet conversational tone, creating smooth transitions, and emphasizing clarity and readability. - **Retention of Key Elements**: Maintain all essential elements and core ideas from the original text, while enhancing the narrative to captivate the reader. - **Adaptation**: Simplify technical details if necessary, ensuring that the transformed content is accessible to a broad audience without losing depth or accuracy. - **Quality**: Aim for a high-quality article that is both informative and engaging, ready for publication. Follow these guidelines to generate a comprehensive, coherent, and outstanding blog post from the provided YouTube captions text. Your final output should be **only** the paraphrased text, styled in Markdown format, and in english language.

	please return the user response in json format example: {"title": "How to be healthy", "content": "to be healthy you can try do some upper exercises"}`)

const outputPath string = "output"
const outputTestPath string = "output_test"

type Dir uint

const (
	CurrentDir Dir = iota
	TmpDir
)

type Dirkind struct {
	Kind         Dir
	Path         string
	RelativePath *string
}

func (dk *Dirkind) SetRelativePath() {
	getLastUri := func(uri string) string {
		output := strings.Split(uri, "/")
		return output[len(output)-1]
	}(dk.Path)
	dk.RelativePath = &getLastUri
}

var currentDir = Dirkind{Kind: CurrentDir,
	Path:         func() string { dir, _ := os.Getwd(); return dir }(),
	RelativePath: nil}

var tmpDir = Dirkind{Kind: TmpDir,
	Path:         func() string { dir, _ := os.MkdirTemp(currentDir.Path, ".tmp"); return dir }(),
	RelativePath: nil,
}

var env = godotenv.Load()

type Subchapter struct {
	Title string
	Text  string
}

type HTMLFile struct {
	Path string
}

func NewSubchapter(title string, text string) Subchapter {
	return Subchapter{Title: title, Text: text}
}

type Subchapters []Subchapter

type EncodedResponse struct {
	OriginalText string `json:"original_text"`
	EncodedText  []uint `json:"encoded_text"`
	TokenLength  int    `json:"token_length"`
}

func saveToMD(filename, text string) error {
	filename = fmt.Sprintf("%s/%s.md", outputPath, filename)
	file, err := os.Create(filename)
	if err != nil {
		os.Mkdir(outputPath, 0755)
		file, err = os.Create(filename)
	}
	_, err = file.WriteString(string(text))
	if err != nil {
		return err
	}
	defer file.Close()
	defer fmt.Println("saved at: ", filename)
	return nil
}

func scanHTMLFiles(folderPath string) ([]string, error) {
	var htmlFiles []string
	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".html" {
			htmlFiles = append(htmlFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return htmlFiles, nil
}

func checkTokenv2(text string) (EncodedResponse, error) {
	enc, err := tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		return EncodedResponse{}, err
	}
	// this should print a list of token ids
	ids, _, _ := enc.Encode(text)
	return EncodedResponse{OriginalText: text, EncodedText: ids, TokenLength: len(ids)}, nil
	// this should print the original string back
}

func checkToken(text string) (EncodedResponse, error) {
	url := "http://127.0.0.1:8080/encode"
	var result EncodedResponse
	result.OriginalText = text
	// Text with newlines
	// Create request payload
	requestBody := map[string]string{
		"text": text,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return result, fmt.Errorf("error marshaling into json: %w", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return result, fmt.Errorf("error creating new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("error posting request bruh...: %w", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("error creating request: %w", err)
	}
	// Define the response struct
	// Parse the response into the struct
	err = json.Unmarshal(body, &result)
	if err != nil {
		return result, fmt.Errorf("error creating request: %w", err)
	}
	return result, nil
}

// ExtractEpub extracts the contents of an EPUB file (which is a ZIP archive)
// to the specified target directory
func ExtractEpub(epubPath string, targetDir string) error {
	// Create extraction directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create extraction directory: %v", err)
	}
	// Open the EPUB file
	reader, err := zip.OpenReader(epubPath)
	if err != nil {
		return fmt.Errorf("error opening EPUB file: %v", err)
	}
	defer reader.Close()
	// Extract each file
	for _, file := range reader.File {
		extractPath := filepath.Join(targetDir, file.Name)
		// Create directories if needed
		if file.FileInfo().IsDir() {
			os.MkdirAll(extractPath, 0755)
			continue
		}
		// Make sure the parent directory exists
		parentDir := filepath.Dir(extractPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", parentDir, err)
		}
		// Create the file
		outFile, err := os.Create(extractPath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %v", extractPath, err)
		}
		// Open the zipped file
		zipFile, err := file.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("failed to open zipped file %s: %v", file.Name, err)
		}
		// Copy the contents
		_, err = io.Copy(outFile, zipFile)
		outFile.Close()
		zipFile.Close()
		if err != nil {
			return fmt.Errorf("failed to extract file %s: %v", file.Name, err)
		}
	}
	return nil
}

func ScanHTMLFiles(rootDir string) ([]HTMLFile, error) {
	var htmlfiles []HTMLFile
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(path), ".html") || strings.HasSuffix(strings.ToLower(path), ".htm") || strings.HasSuffix(strings.ToLower(path), ".xhtml") || strings.HasSuffix(strings.ToLower(path), ".xhtm") {
			htmlfiles = append(htmlfiles, HTMLFile{Path: path})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return htmlfiles, nil
}

func renderTemplate(tmplFile string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ext := strings.ToLower(filepath.Ext(tmplFile))
		if ext == ".xhtml" {
			w.Header().Set("Content-Type", "application/xhtml+xml")
		}
		tmpl, err := template.ParseFiles(tmplFile)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
	}
}

func handlerHomepage(w http.ResponseWriter, r *http.Request) {
	// Set the content type to HTML
	w.Header().Set("Content-Type", "text/html")
	// Write HTML directly
	fmt.Fprintf(w, `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Hello from Go</title>
		</head>
		<body>
			<h1>Welcome to my Go server!</h1>
			<p>This is an HTML response.</p>
		</body>
		</html>
	`)
}

func handlerHomepage2(w http.ResponseWriter, r *http.Request) {
	// Set the content type to HTML
	w.Header().Set("Content-Type", "text/html")
	// Write HTML directly
	fmt.Fprintf(w, `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Hello from Go</title>
		</head>
		<body>
			<h1>Welcome to my Go server!</h1>
			<p>This is an HTML response.</p>
		</body>
		</html>
	`)
}

func makeHandler(text string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
			<!DOCTYPE html>
			<html>
			<head><title>Custom Message</title></head>
			<body>
				%s
			</body>
			</html>
		`, text)
	}
}

func startHTTPServer() {
	routes, _ := ScanHTMLFiles(*tmpDir.RelativePath)
	var href string
	// Register handlers in a loop
	for _, file := range routes {
		http.HandleFunc("/"+file.Path, renderTemplate(file.Path))
		filePathsep := strings.Split(file.Path, "/")
		chapterName := filePathsep[len(filePathsep)-1]
		href += "<a href='/" + file.Path + "'>" + chapterName + "</a> <br>"
	}
	http.HandleFunc("/", makeHandler(href))
	// log.Println("Server started at http://localhost:8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func readJson(filename string) ([]string, error) {
	var result []string
	data, err := os.ReadFile(filename)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Println("file not exist")
		return result, nil
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func deleteTempFolders(folders []string) {
	for _, folder := range folders {
		err := os.RemoveAll(folder)
		if os.IsNotExist(err) {
			fmt.Println("Folder does not exist:", folder)
		}
	}
}

func main() {
	book := flag.String("book", "", "book name, ex: book.epub")
	portStr := flag.String("port", "8000", "Port number")
	flag.Parse()
	fmt.Println(*portStr)
	if len(*book) == 0 {
		fmt.Println("Usage: cli-epub-parser-md-generator -book <book_name>")
		fmt.Println("Optional to give the custom port usage: cli-epub-parser-md-generator -book <book_name> -port <portnumber>")
		os.Exit(1)
	}
	tmpDir.SetRelativePath()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println("interrupt signal received, deleting temp folder", sig)
		os.RemoveAll(tmpDir.Path)
		os.Exit(0)
	}()
	// initCheckServer(8080, "server running on port 8080, python tokenizer server")
	if len(os.Args) < 2 {
		os.RemoveAll(tmpDir.Path)
		fmt.Println("Usage: cli-epub-parser-md-generator <epub_file>")
		os.Exit(1)
	}
	// err := ExtractEpub(bookName, ".tmp")
	err := ExtractEpub(*book, tmpDir.Path)
	// tmpDirs, err := readJson(".tmpDirs.json")
	// tmpDirs = append(tmpDirs, tmpDir.Path)
	// // tmpDirs
	// defer deleteTempFolders(tmpDirs)
	if err != nil {
		panic(err)
	}
	initCheckServer := func(port int, description string) {
		port_conv := strconv.Itoa(port)
		url := "http://127.0.0.1:" + port_conv + "/"
		req, err := http.NewRequest("GET", url, nil)
		client := &http.Client{}
		_, err = client.Do(req)
		if err != nil {
			// panic(err)
		}
		fmt.Println(description)
	}

	// go startHTTPServer()

	// go func() {
	// 	routes, _ := ScanHTMLFiles(*tmpDir.RelativePath)
	// 	var href string
	// 	// Register handlers in a loop
	// 	for _, file := range routes {
	// 		http.HandleFunc("/"+file.Path, renderTemplate(file.Path))
	// 		filePathsep := strings.Split(file.Path, "/")
	// 		chapterName := filePathsep[len(filePathsep)-1]
	// 		href += "<a href='/" + file.Path + "'>" + chapterName + "</a> <br>"
	// 	}
	// 	http.HandleFunc("/", makeHandler(href))
	// 	// log.Println("Server started at http://localhost:8000")
	// 	err := http.ListenAndServe(":8000", nil)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// }()

	go func() {
		// Directory you want to serve
		dir, _ := os.Getwd()
		fs := http.FileServer(http.Dir(dir))

		// Serve the files at root path "/"
		http.Handle("/", fs)

		// Start the server
		log.Printf("Starting server on :%s...", *portStr)
		addressPort := fmt.Sprintf(":%s", *portStr)
		log.Fatal(http.ListenAndServe(addressPort, nil))
	}()
	port, err := strconv.Atoi(*portStr)
	initCheckServer(port, "server running on port 8000, go simple http server")
	// create channel so that when user exit program by pressing ctrl+c, the temp folder is deleted
	// ☝️ it just works btw
	var Subchapters = []Subchapter{}
	c := colly.NewCollector()
	c.OnRequest(func(r *colly.Request) {
		// fmt.Println("Visiting", r.URL)
		fmt.Println("Processing...")
	})
	c.OnHTML("body", func(e *colly.HTMLElement) {
		// Subchapters = append(Subchapters, NewSubchapter(e.Attr("data-pdf-bookmark"), e.Text))
		fmt.Println("len of text is: ", len(e.Text))
		if len(e.Text) == 0 {
			fmt.Println("emtpy text")
			os.Exit(0)
		}
		Subchapters = append(Subchapters, NewSubchapter("output", e.Text))
	})
	// c.OnHTML("section[data-pdf-bookmark][data-type='sect1']", func(e *colly.HTMLElement) {
	// 	Subchapters = append(Subchapters, NewSubchapter(e.Attr("data-pdf-bookmark"), e.Text))
	// })
	// filePath, err := scanHTMLFiles("test_data")
	filePath, err := ScanHTMLFiles(*tmpDir.RelativePath)
	if err != nil {
		fmt.Println(err)
		return
	}
	for i, file := range filePath {
		texts := strings.Split(file.Path, "/")
		fmt.Printf("%d: %s\n", i+1, texts[len(texts)-1])
	}
	var userInput string
	fmt.Println("choose a chapter based on number")
	fmt.Scanln(&userInput)
	var chapterNumber int
	chapterNumber, err = strconv.Atoi(userInput)
	if err != nil {
		fmt.Println(err)
		fmt.Println("changing user input to 10, means its in testing")
		chapterNumber = 10
	}
	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Request URL: %v | failed with response %v", r.Request.URL, err)
	})
	targetUrl := fmt.Sprintf("http://127.0.0.1:%s/%s", *portStr, filePath[chapterNumber-1].Path)
	err = c.Visit(targetUrl)
	if err != nil {
		fmt.Println(err)
		return
	}
	var fullText string
	for _, subchapter := range Subchapters {
		fullText += subchapter.Text
	}
	tokenizeChannel := make(chan EncodedResponse)
	err = nil
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
	fmt.Println("Original token length is: ", tokenize.TokenLength)

	type deepseekOutput struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	client := deepseek.NewClient(os.Getenv("DEEPSEEK_API_KEY"))
	// Create a chat completion request

	request := &deepseek.ChatCompletionRequest{
		Model: deepseek.DeepSeekChat,
		Messages: []deepseek.ChatCompletionMessage{
			{Role: deepseek.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: deepseek.ChatMessageRoleUser, Content: tokenize.OriginalText},
		},
		JSONMode: true,
	}

	// Send the request and handle the response
	deepseek_ctx := context.Background()
	extractor := deepseek.NewJSONExtractor(nil)
	if err != nil {
		panic(err)
	}
	response, err := client.CreateChatCompletion(deepseek_ctx, request)
	var output deepseekOutput

	if err := extractor.ExtractJSON(response, &output); err != nil {
		panic(err)
	}
	err = saveToMD(output.Title, output.Content)
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir.Path)
}
