package main

import (
	"context"
	"fmt"
	"github.com/google/go-github/v72/github"
	"os"
	"regexp"
	"strconv"
)

var (
	rePull      = regexp.MustCompile(`pull/[0-9]*`)
	reIssues    = regexp.MustCompile(`issues/[0-9]*`)
	issuesNumRe = regexp.MustCompile(`issues/[0-9]+`)
	pullsNumRe  = regexp.MustCompile(`pull/[0-9]+`)
	numRe       = regexp.MustCompile(`[0-9]+`)
)

func printInfo(notification *github.Notification, state string, prNumberStr string, url string) {
	fmt.Println("Number: " + prNumberStr)
	fmt.Println("Url: " + url)
	fmt.Println("Subject: " + *notification.Subject.Title)
	fmt.Println("Reason: " + *notification.Reason)
	fmt.Println("state: " + state)
}

func main() {
	client := github.NewClient(nil).WithAuthToken(os.Getenv("GH_TOKEN"))
	fmt.Println("Got client")
	if err := markNotificationsAsRead(context.Background(), client); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func markNotificationsAsRead(ctx context.Context, client *github.Client) error {
	opt := &github.NotificationListOptions{}
	var allNotifications []*github.Notification
	notifications, response, err := client.Activity.ListNotifications(ctx, opt)
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

	for {
		notifications, response, err = client.Activity.ListNotifications(ctx, opt)
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
		allNotifications = append(allNotifications, notifications...)
		if response.NextPage == 0 {
			break
		}
		opt.Page = response.NextPage
		fmt.Println("Page: ", opt.Page)
	}

	for _, notification := range allNotifications {
		url := *notification.Subject.URL
		urlBytes := []byte(url)
		matched := rePull.MatchString(url)
		issueMatched := reIssues.MatchString(url)
		if matched {
			if err := handlePR(urlBytes, client, ctx, notification, url); err != nil {
				return err
			}
		}
		if issueMatched {
			if err := handleIssue(urlBytes, client, ctx, notification, url); err != nil {
				fmt.Println(err)
				// return err
			}
		}
	}
	return nil
}

func handleIssue(urlBytes []byte, client *github.Client, ctx context.Context, notification *github.Notification, url string) error {
	foundB := issuesNumRe.Find(urlBytes)
	issueNumber := numRe.Find(foundB)
	issueNumberStr := string(issueNumber)
	issueNumberInt, err := strconv.Atoi(issueNumberStr)
	if err != nil {
		return err
	}
	issue, _, err := client.Issues.Get(ctx, *notification.Repository.Owner.Login, *notification.Repository.Name, issueNumberInt)
	if err != nil {
		return err
	}
	if *issue.State == "closed" || *notification.Reason == "assign" {
		printInfo(notification, *issue.Title, issueNumberStr, url)
		idInt, err := strconv.Atoi(*notification.ID)
		if err != nil {
			return err
		}
		resp, err := client.Activity.MarkThreadDone(ctx, int64(idInt))
		if err != nil {
			return fmt.Errorf("error %w %s", err, resp.Status)
		}
		fmt.Println("Marked as done")
		fmt.Println()
	}
	return nil
}

func handlePR(urlBytes []byte, client *github.Client, ctx context.Context, notification *github.Notification, url string) error {
	foundB := pullsNumRe.Find(urlBytes)
	prNumber := numRe.Find(foundB)
	prNumberStr := string(prNumber)
	prNumberInt, err := strconv.Atoi(prNumberStr)
	if err != nil {
		return err
	}

	pr, _, err := client.PullRequests.Get(ctx, *notification.Repository.Owner.Login, *notification.Repository.Name, prNumberInt)
	if err != nil {
		return err
	}
	if *pr.State == "closed" || *pr.Merged || *notification.Reason == "assign" {
		printInfo(notification, *pr.State, prNumberStr, url)
		idInt, err := strconv.Atoi(*notification.ID)
		if err != nil {
			return err
		}
		resp, err := client.Activity.MarkThreadDone(ctx, int64(idInt))
		if err != nil {
			return fmt.Errorf("error %w %s", err, resp.Status)
		}
		fmt.Println("Marked as done")
		fmt.Println()
	}
	return nil
}
