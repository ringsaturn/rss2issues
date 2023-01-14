package main

import (
	"context"
	"os"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/google/go-github/v48/github"
	"github.com/mmcdole/gofeed"
	"github.com/ringsaturn/rss2issues"
	"golang.org/x/exp/slog"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

type RSSConfig struct {
	URL    string   `yaml:"url"`
	Labels []string `yaml:"labels"`
}

func init() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr)))
}

func main() {
	owner := os.Getenv("RSS2ISSUE_REPO_OWNER")
	repo := os.Getenv("RSS2ISSUE_REPO")
	token := os.Getenv("RSS2ISSUE_TOKEN")
	configPath := os.Getenv("RSS2ISSUE_CONFIGPATH")
	content, err := os.ReadFile(configPath)
	if err != nil {
		slog.Error("unable read config", err, "configPath", configPath)
		return
	}
	feedConfigs := []RSSConfig{}
	err = yaml.Unmarshal(content, &feedConfigs)
	if err != nil {
		slog.Error("faild to parse config", err, "content", string(content))
		return
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	converter := md.NewConverter("", true, nil)

	now := time.Now()

	for _, feedConfig := range feedConfigs {
		fp := gofeed.NewParser()
		feed, err := fp.ParseURLWithContext(feedConfig.URL, ctx)
		if err != nil {
			slog.Warn("faild to parse URL", err, "url", feedConfig.URL)
			continue
		}
		for _, item := range feed.Items {
			if pubts := item.PublishedParsed.Unix(); pubts < now.Unix()-3600 {
				slog.Info("bypass too old", "pubts", pubts, "title", item.Title)
				continue
			}
			markdown, err := converter.ConvertString(item.Description)
			if err != nil {
				slog.Error("faild to convert to Markdown", err, "desc", item.Description)
				continue
			}
			err = rss2issues.UpsertRSSFeeds(
				ctx, client,
				owner, repo,
				item.Title,
				markdown,
				feedConfig.Labels,
				"",
				"not_planned",
				nil,
				[]string{},
			)
			if err == nil {
				slog.Info("sucess upsert issue", err, "title", item.Title)
				continue
			}
			slog.Error("faild to upsert issue", err, "title", item.Title)
		}
	}
}
