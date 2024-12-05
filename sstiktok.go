package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	baseURL             = "https://ssstik.io"
	tiktokURLPattern    = `https:\/\/(?:m|www|vm|vt|lite)?\.?tiktok\.com\/((?:.*\b(?:(?:usr|v|embed|user|video|photo)\/|\?shareId=|\&item_id=)(\d+))|\w+)`
	ssstikTokenPattern  = `s_tt\s*=\s*'([^']+)'`
	overlayURLPattern   = `#mainpicture \.result_overlay\s*{\s*background-image:\s*url\(["']?([^"']+)["']?\);\s*}`
)

func main() {
	// Check and get the TikTok URL argument
	if len(os.Args) < 2 {
		fmt.Println("The arguments must contain a valid TikTok URL.")
		return
	}
	url := strings.Join(os.Args[1:], " ")

	// Validate TikTok URL
	if !validateTikTokURL(url) {
		fmt.Println("Must be a valid TikTok URL.")
		return
	}

	// Get the SSSTik token
	token, err := extractToken()
	if err != nil {
		fmt.Println(err)
		return
	}

	// Scrape the SSSTik page with the provided URL
	data, err := scrapeSSSTik(url, token)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Print the resulting data with formatted output
	printFormattedResult(data)
}

func validateTikTokURL(url string) bool {
	regex := regexp.MustCompile(tiktokURLPattern)
	return regex.MatchString(url)
}

func extractToken() (string, error) {
	resp, err := http.Get(baseURL)
	if err != nil {
		return "", errors.New("failed to fetch the SSSTik homepage")
	}
	defer resp.Body.Close()

	// Read the HTML content
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New("failed to read SSSTik homepage content")
	}

	// Extract the token using regex
	regex := regexp.MustCompile(ssstikTokenPattern)
	matches := regex.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		return "", errors.New("unable to extract SSSTik token")
	}

	return matches[1], nil
}

func scrapeSSSTik(url, token string) (map[string]interface{}, error) {
	// Prepare form data
	formData := fmt.Sprintf("id=%s&locale=en&tt=%s", url, token)
	req, err := http.NewRequest("POST", baseURL+"/abc?url=dl", bytes.NewBufferString(formData))
	if err != nil {
		return nil, errors.New("failed to create POST request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Referer", baseURL+"/en")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("failed to send POST request")
	}
	defer resp.Body.Close()

	// Parse the response HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, errors.New("failed to parse response HTML")
	}

	// Extract data using selectors
	username := strings.TrimSpace(doc.Find("h2").Text())
	description := strings.TrimSpace(doc.Find(".maintext").Text())
	likeCount := strings.TrimSpace(doc.Find("div.trending-actions > div.justify-content-start").Eq(0).Text())
	commentCount := strings.TrimSpace(doc.Find("div.trending-actions > div.justify-content-center > div").Text())
	shareCount := strings.TrimSpace(doc.Find("div.trending-actions > div.justify-content-end > div").Text())
	avatarURL, _ := doc.Find("img.result_author").Attr("src")
	videoURL, _ := doc.Find("a.without_watermark").Attr("href")
	musicURL, _ := doc.Find("a.music").Attr("href")

	// Parse style for overlay URL
	styleContent := doc.Find("style").Text()
	regex := regexp.MustCompile(overlayURLPattern)
	overlayMatch := regex.FindStringSubmatch(styleContent)
	var overlayURL string
	if len(overlayMatch) > 1 {
		overlayURL = overlayMatch[1]
	}

	// Return the result as a map
	return map[string]interface{}{
		"username":    username,
		"description": description,
		"statistics": map[string]string{
			"likeCount":    likeCount,
			"commentCount": commentCount,
			"shareCount":   shareCount,
		},
		"downloads": map[string]string{
			"avatarUrl":  avatarURL,
			"overlayUrl": overlayURL,
			"videoUrl":   videoURL,
			"musicUrl":   musicURL,
		},
	}, nil
}

func printFormattedResult(data map[string]interface{}) {
	fmt.Println("Result:")
	fmt.Printf("\tUsername:\t%s\n", data["username"])
	fmt.Printf("\tDescription:\t%s\n", data["description"])

	fmt.Println("\tStatistics:")
	statistics := data["statistics"].(map[string]string)
	fmt.Printf("\t\tLike Count:\t%s\n", statistics["likeCount"])
	fmt.Printf("\t\tComment Count:\t%s\n", statistics["commentCount"])
	fmt.Printf("\t\tShare Count:\t%s\n", statistics["shareCount"])

	fmt.Println("\tDownloads:")
	downloads := data["downloads"].(map[string]string)
	fmt.Printf("\t\tAvatar URL:\t%s\n", downloads["avatarUrl"])
	fmt.Printf("\t\tOverlay URL:\t%s\n", downloads["overlayUrl"])
	fmt.Printf("\t\tVideo URL:\t%s\n", downloads["videoUrl"])
	fmt.Printf("\t\tMusic URL:\t%s\n", downloads["musicUrl"])
}
