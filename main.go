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
	var allPullRequests []*github.PullRequest

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

	var reviewablePullRequests []*github.PullRequest
	for _, pr := range allPullRequests {
		fmt.Println(fmt.Sprintf("%v  %v", pr.GetMergeable(), pr.GetMergeableState()))
		if pr.Mergeable == nil || !*pr.Mergeable {
			continue
		}
		reviewablePullRequests = append(reviewablePullRequests, pr)
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

//func handleIssue(urlBytes []byte, client *github.Client, ctx context.Context, notification *github.Notification, url string) error {
//	foundB := issuesNumRe.Find(urlBytes)
//	issueNumber := numRe.Find(foundB)
//	issueNumberStr := string(issueNumber)
//	issueNumberInt, err := strconv.Atoi(issueNumberStr)
//	if err != nil {
//		return err
//	}
//	issue, _, err := client.Issues.Get(ctx, *notification.Repository.Owner.Login, *notification.Repository.Name, issueNumberInt)
//	if err != nil {
//		return err
//	}
//	if *issue.State == "closed" || *notification.Reason == "assign" {
//		printInfo(notification, *issue.Title, issueNumberStr, url)
//		idInt, err := strconv.Atoi(*notification.ID)
//		if err != nil {
//			return err
//		}
//		resp, err := client.Activity.MarkThreadDone(ctx, int64(idInt))
//		if err != nil {
//			return fmt.Errorf("error %w %s", err, resp.Status)
//		}
//		fmt.Println("Marked as done")
//		fmt.Println()
//	}
//	return nil
//}

//func handlePR(urlBytes []byte, client *github.Client, ctx context.Context, notification *github.Notification, url string) error {
//	foundB := pullsNumRe.Find(urlBytes)
//	prNumber := numRe.Find(foundB)
//	prNumberStr := string(prNumber)
//	prNumberInt, err := strconv.Atoi(prNumberStr)
//	if err != nil {
//		return err
//	}
//
//	pr, _, err := client.PullRequests.Get(ctx, *notification.Repository.Owner.Login, *notification.Repository.Name, prNumberInt)
//	if err != nil {
//		return err
//	}
//	if *pr.State == "closed" || *pr.Merged || *notification.Reason == "assign" {
//		printInfo(notification, *pr.State, prNumberStr, url)
//		idInt, err := strconv.Atoi(*notification.ID)
//		if err != nil {
//			return err
//		}
//		resp, err := client.Activity.MarkThreadDone(ctx, int64(idInt))
//		if err != nil {
//			return fmt.Errorf("error %w %s", err, resp.Status)
//		}
//		fmt.Println("Marked as done")
//		fmt.Println()
//	}
//	return nil
//}
