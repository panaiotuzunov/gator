package fetch

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
)

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func FetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return &RSSFeed{}, fmt.Errorf("error creating request - %v", err)
	}
	req.Header.Set("User-Agent", "gator")
	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return &RSSFeed{}, fmt.Errorf("error getting a response - %v", err)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return &RSSFeed{}, fmt.Errorf("error reading response body - %v", err)
	}
	var data RSSFeed
	if err := xml.Unmarshal(body, &data); err != nil {
		return &RSSFeed{}, fmt.Errorf("error unmarshaling body - %v", err)
	}
	data.Channel.Title = html.UnescapeString(data.Channel.Title)
	data.Channel.Description = html.UnescapeString(data.Channel.Description)
	for i := range data.Channel.Item {
		data.Channel.Item[i].Title = html.UnescapeString(data.Channel.Item[i].Title)
		data.Channel.Item[i].Description = html.UnescapeString(data.Channel.Item[i].Description)
	}
	return &data, nil
}
