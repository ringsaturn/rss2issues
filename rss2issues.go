package rss2issues

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v48/github"
	"golang.org/x/exp/slog"
	"golang.org/x/time/rate"
)

var (
	limiter = rate.NewLimiter(rate.Every(time.Minute), 20)
)

func CreateIssue(
	ctx context.Context,
	client *github.Client,
	owner string,
	repo string,
	req *github.IssueRequest,
) (*github.Response, error) {
	_, resp, err := client.Issues.Create(ctx, owner, repo, req)
	return resp, err
}

func UpdateIssue(
	ctx context.Context,
	client *github.Client,
	owner string,
	repo string,
	number int,
	req *github.IssueRequest,
) (*github.Response, error) {
	_, resp, err := client.Issues.Edit(ctx, owner, repo, number, req)
	return resp, err
}

func ComposeQuery(owner, repo, title string) string {
	rawQuery := fmt.Sprintf(`repo:%v/%v "%v"`, owner, repo, title)
	return rawQuery
}

func UpsertRSSFeeds(
	ctx context.Context,
	client *github.Client,
	owner string,
	repo string,
	title string,
	body string,
	labels []string,
	state string,
	stateReason string,
	milestone *int,
	assignees []string,
) error {
	req := &github.IssueRequest{
		Title:       &title,
		Body:        &body,
		Labels:      &labels,
		State:       &state,
		StateReason: &stateReason,
		Milestone:   milestone,
		Assignees:   &assignees,
	}
	opts := &github.SearchOptions{
		TextMatch: true,
	}
	query := ComposeQuery(owner, repo, title)
	_ = limiter.Wait(ctx)
	results, _, err := client.Search.Issues(ctx, query, opts)
	slog.Info("try search issues", "title", title, "query", query, "results", results, "err", err)
	if err == nil {
		for _, issue := range results.Issues {
			if *issue.Title == title {
				issueAlias := fmt.Sprintf("%v/%v#%d", owner, repo, *issue.Number)
				slog.Info("found issue", "title", title, "issueAlias", issueAlias)
				_, err = UpdateIssue(ctx, client, owner, repo, *issue.Number, req)
				if err != nil {
					slog.Error("update failed", err, "title", title, "issueAlias", issueAlias)
				}
				return err
			}
		}
	}
	_, err = CreateIssue(ctx, client, owner, repo, req)
	return err
}
