package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"github.com/google/go-github/v72/github"
	"os"
)

const cacheFile = "pull_requests.gob"

func main() {
	client := github.NewClient(nil).WithAuthToken(os.Getenv("GH_TOKEN"))
	fmt.Println("Got client")
	if err := handlePullRequests(context.Background(), client); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func handlePullRequests(ctx context.Context, client *github.Client) error {
	gob.Register([]*github.PullRequest{})
	gob.Register(github.PullRequest{})
	gob.Register(github.User{})
	gob.Register(github.Repository{})
	gob.Register(github.Label{})
	gob.Register(github.Milestone{})
	gob.Register(github.PullRequestBranch{})
	gob.Register(github.PullRequestLinks{})
	gob.Register(github.PullRequestReview{})
	gob.Register(github.PullRequestComment{})
	allPullRequests := make([]*github.PullRequest, 0, 1000)

	// Check if cached data exists
	if fileExists(cacheFile) {
		if err := loadFromFile(cacheFile, &allPullRequests); err != nil {
			return fmt.Errorf("failed to load cache: %w", err)
		}
		fmt.Println("Loaded pull requests from cache")
	} else {
		opt := &github.PullRequestListOptions{}
		for {
			pullRequestBatch, response, err := client.PullRequests.List(ctx, "helm", "helm", opt)
			if _, ok := err.(*github.RateLimitError); ok {
				fmt.Println("hit rate limit")
			}
			if err != nil {
				if response != nil && response.Status != "" {
					return fmt.Errorf("error %w %s", err, response.Status)
				} else {
					return fmt.Errorf("error %w", err)
				}
			}
			allPullRequests = append(allPullRequests, pullRequestBatch...)
			if response.NextPage == 0 {
				break
			}
			opt.Page = response.NextPage
			fmt.Println("Page: ", opt.Page)
		}

		// Save to cache
		if err := saveToFile(cacheFile, allPullRequests); err != nil {
			return fmt.Errorf("failed to save cache: %w", err)
		}
		fmt.Println("Saved pull requests to cache")
	}

	var allDetailedPrs []github.PullRequest
	for _, pr := range allPullRequests {

		var detailedPr github.PullRequest
		prFile := fmt.Sprintf("pr/%d.gob", *pr.Number)
		if !fileExists(prFile) {
			detailedPr, _, err := client.PullRequests.Get(ctx, "helm", "helm", *pr.Number)
			if err != nil {
				fmt.Printf("failed to fetch PR #%d: %v\n", *pr.Number, err)
				continue
			}
			if err := os.MkdirAll("pr", 0755); err != nil {
				return fmt.Errorf("failed to create pr directory: %w", err)
			}
			if err := saveToFile(prFile, detailedPr); err != nil {
				fmt.Printf("failed to save PR #%d: %v\n", *pr.Number, err)
			}
		} else {
			if err := loadFromFile(prFile, &detailedPr); err != nil {
				fmt.Printf("failed to load PR #%d: %v\n", *pr.Number, err)
				continue
			}
		}
		allDetailedPrs = append(allDetailedPrs, detailedPr)
	}

	var reviewablePullRequests []*github.PullRequest
	//var jessePullRequests []*github.PullRequest
	for _, pr := range allDetailedPrs {
		//if pr.GetUser() != nil && *pr.GetUser().Login == "jessesimpson36" {
		//	jessePullRequests = append(jessePullRequests, &pr)
		//}
		if pr.Mergeable == nil || !*pr.Mergeable {
			continue
		}
		if pr.GetMergeableState() == "dirty" || pr.GetMergeableState() == "blocked" || pr.GetMergeableState() == "unknown" {
			continue
		}
		reviewablePullRequests = append(reviewablePullRequests, &pr)
	}
	for _, pr := range reviewablePullRequests {
		fmt.Println(fmt.Sprintf("%v\tPR #%v\t%v", *pr.GetUser().Login, pr.GetNumber(), pr.GetTitle()))
	}

	fmt.Println("hi")
	return nil
}

func saveToFile(filename string, data interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	return encoder.Encode(data)
}

func loadFromFile(filename string, data interface{}) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	return decoder.Decode(data)
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
