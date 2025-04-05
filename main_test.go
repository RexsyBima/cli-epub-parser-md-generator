package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"testing"

	"github.com/gocolly/colly"
)

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

func TestExtractEpub(t *testing.T) {
	t.Skip("skipping for now...")
	tests := []struct {
		filename string // description of this test case
	}{
		struct{ filename string }{"book1.epub"},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			fmt.Println(tt.filename)
		})
	}
}

func TestScrapeHTML(t *testing.T) {
	var Subchapters = []Subchapter{}
	c := colly.NewCollector()
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})
	c.OnHTML("section[data-pdf-bookmark][data-type='sect1']", func(e *colly.HTMLElement) {
		// fmt.Println(e.Text)
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
