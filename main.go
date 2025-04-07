package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	deepseek "github.com/cohesion-org/deepseek-go"
	"github.com/gocolly/colly"
	"github.com/joho/godotenv"
)

var env = godotenv.Load()

type Subchapter struct {
	Title string
	Text  string
}

func NewSubchapter(title string, text string) Subchapter {
	return Subchapter{Title: title, Text: text}
}

type Subchapters []Subchapter

type EncodedResponse struct {
	OriginalText string `json:"original_text"`
	EncodedText  []int  `json:"encoded_text"`
	TokenLength  int    `json:"token_length"`
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

func checkToken(text string) (EncodedResponse, error) {
	url := "http://127.0.0.1:8080/encode"
	// Text with newlines
	// Create request payload
	requestBody := map[string]string{
		"text": text,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return EncodedResponse{}, fmt.Errorf("error creating request: %w", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return EncodedResponse{}, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return EncodedResponse{}, fmt.Errorf("error creating request: %w", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return EncodedResponse{}, fmt.Errorf("error creating request: %w", err)
	}
	// Define the response struct
	// Parse the response into the struct
	var result EncodedResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return EncodedResponse{}, fmt.Errorf("error creating request: %w", err)
	}
	return result, nil
}

func initCheckServer(port int, description string) {
	port_conv := strconv.Itoa(port)
	url := "http://127.0.0.1:" + port_conv + "/"
	req, err := http.NewRequest("GET", url, nil)
	client := &http.Client{}
	_, err = client.Do(req)
	if err != nil {
		panic(err)
	}
	fmt.Println(description)
}

func init() {
	initCheckServer(8000, "server running on port 8000, python simple http server")
	initCheckServer(8080, "server running on port 8080, python tokenizer server")
}

func main() {
	var systemPrompt string = fmt.Sprintf(`You are an AI transformation agent tasked with converting raw YouTube caption texts about knowledge into a polished, engaging, and readable blog post. Your responsibilities include: - **Paraphrasing**: Transform the original caption text into fresh, original content while preserving the key information and insights. - **Structure**: Organize the content into a well-defined structure featuring a captivating introduction, clearly delineated subheadings in the body, and a strong conclusion. - **Engagement**: Ensure the blog post is outstanding by using a professional yet conversational tone, creating smooth transitions, and emphasizing clarity and readability. - **Retention of Key Elements**: Maintain all essential elements and core ideas from the original text, while enhancing the narrative to captivate the reader. - **Adaptation**: Simplify technical details if necessary, ensuring that the transformed content is accessible to a broad audience without losing depth or accuracy. - **Quality**: Aim for a high-quality article that is both informative and engaging, ready for publication. Follow these guidelines to generate a comprehensive, coherent, and outstanding blog post from the provided YouTube captions text. Your final output should be **only** the paraphrased text, styled in Markdown format, and in english language.`)
	var Subchapters = []Subchapter{}
	c := colly.NewCollector()
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})
	c.OnHTML("section[data-pdf-bookmark][data-type='sect1']", func(e *colly.HTMLElement) {
		Subchapters = append(Subchapters, NewSubchapter(e.Attr("data-pdf-bookmark"), e.Text))
	})
	filePath, err := scanHTMLFiles("test_data")
	if err != nil {
		fmt.Println(err)
		return
	}
	for i, file := range filePath {
		fmt.Printf("%d: %s\n", i+1, file)
	}
	var userInput string
	fmt.Println("choose a chapter based on number")
	fmt.Scanln(&userInput)
	chapterNumber, err := strconv.Atoi(userInput)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = c.Visit("http://127.0.0.1:8000/" + filePath[chapterNumber-1])
	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Request URL: %v | failed with response %v", r.Request.URL, err)
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	var fullText string
	for _, subchapter := range Subchapters {
		fullText += subchapter.Text
	}
	tokenize, err := checkToken(fullText)
	if err != nil {
		fmt.Println(err)
		return
	}
	// fmt.Println(tokenize.OriginalText)
	fmt.Println(tokenize.TokenLength)
	fmt.Println(tokenize.OriginalText)
	client := deepseek.NewClient(os.Getenv("DEEPSEEK_API_KEY"))
	// Create a chat completion request
	request := &deepseek.ChatCompletionRequest{
		Model: deepseek.DeepSeekChat,
		Messages: []deepseek.ChatCompletionMessage{
			{Role: deepseek.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: deepseek.ChatMessageRoleUser, Content: tokenize.OriginalText},
		},
	}
	// Send the request and handle the response
	deepseek_ctx := context.Background()
	response, err := client.CreateChatCompletion(deepseek_ctx, request)
	if err != nil {
		panic(err)
	}
	// Print the response
	output := response.Choices[0].Message.Content
	fmt.Println("Response:", output)

}
