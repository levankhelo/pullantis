/*

	initialize git things - (token, repo)
	initialize pulumi things - (token)

	start server to listen to github webhooks
	if webhook is received execute handler
		if webhook is PullRquest Created than push Queue
		if webhook is PullRequest Deleted/Resolved than pop Queue
		if webhook is comment created and comment content = pullantis plan than push Queue

	in Queue
		if queue is empty and new element is pushed than do Action
		if queue is not empty, wait for next event and check again
		if something got deleted, execute Action on next element

	in Action
		check out on PR branch
		run pullantis
		comment results on PL review



*/

package main

// import libraries
import (
	"context"
	"encoding/json"
	"flag"
	f "fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	github "github.com/google/go-github/v33/github"
	"github.com/pulumi/pulumi/sdk/go/pulumi"
	"golang.org/x/oauth2"
)

/// GLOBAL VARIABLES
var gitClient *github.Client

//// STRUCTURES

// GitComment is Review Comment on PL
type GitComment struct {
	body string
}

// GitPL is Pull request structure
type GitPL struct {
	project string     // project name
	branch  string     // branch name
	action  string     // action done: comment pull request close -> opened=new PL, created=new Review, closed=closed PL
	ID      int        // ID of pull request -> pull_request.number
	commit  string     // commit body
	comment GitComment // commit body
	running bool       //  queue mechanism - running or not
	command string     // apply or plan
}

// Queue primitive data structure
type Queue struct {
	list   []GitPL
	length int
}

func (q Queue) push(el GitPL) (qu Queue) {
	// push element at bottom of array
	q.list = append(q.list, el)
	q.length++
	q.disp()
	return q
}
func (q Queue) pop() (qu Queue) {
	// pop first element of array
	if q.length > 1 {
		q.list = q.list[1:]
	} else {
		q.list = make([]GitPL, 0)
	}
	q.length--
	q.disp()
	return q
}
func (q Queue) remove(index int) (qu Queue) {
	// rewrite elements by skipping indexed elements
	if q.length <= index {
		f.Println("queue.remove ", index, "out of range. length:", q.length, len(q.list))
		return
	}

	oldList := q.list
	q.list = make([]GitPL, 0)
	for i := 0; i < q.length; i++ {
		if i == index {
			q.length--
			continue
		}
		q.list = append(q.list, oldList[i])
	}
	q.disp()
	return q
}
func (q Queue) removeByID(ID int) (qu Queue) {
	// run throught elements and find mathcing ID
	oldList := q.list
	q.list = make([]GitPL, 0)
	for i := 0; i < q.length; i++ {
		if oldList[i].ID == ID {
			q.length--
			continue
		}
		q.list = append(q.list, oldList[i])
	}
	q.disp()
	return q
}
func (q Queue) removeByObject(obj GitPL) (qu Queue) {
	// run throught elements and find mathcing ID
	oldList := q.list
	q.list = make([]GitPL, 0)
	for i := 0; i < q.length; i++ {
		if oldList[i] == obj {
			q.length--
			continue
		}
		q.list = append(q.list, oldList[i])
	}
	q.disp()
	return q
}
func (q Queue) disp() {
	f.Printf("Queue: %v\n", q)
}
func (q Queue) getByID(ID int) GitPL {
	// run and return element by mathcing ID
	for i := 0; i < q.length; i++ {
		if q.list[i].ID == ID {
			return q.list[i]
		}
	}
	return GitPL{}
}
func (q Queue) getLast() (pl GitPL) {
	// run and change element by mathcing ID
	return q.list[len(q.list)-1]
}

var queue Queue

//// FUNCTIONS

// SetupCloseHandler - handle clean up after exit
func SetupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// new thread for cleanup
	go func() {
		<-c
		f.Println("\r- Ctrl+C pressed in Terminal")
		// remove temp directory
		os.RemoveAll("./tmp/")
		os.Exit(0)
	}()
}

// goCloneGitRepo - Clone GitHub Repo in tmp/ directory
func goCloneGitRepo(tokenGit string, repo *github.Repository) {
	cloneurl := *repo.CloneURL
	gitCommand := cloneurl[:8] + tokenGit + "@" + cloneurl[8:]
	cmd := exec.Command("git", "clone", gitCommand)
	cmd.Run()
	f.Println("Git: Clonned ", cloneurl)
}

// goGitLogin - log in to git and return client
func goGitLogin(token string) (user *github.Client, contx context.Context) {
	/*
		Args:
			token: git token
		Returns:
			user: github client
			contx: context
	*/
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	// authenticate new client using context and token
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc), ctx
}

// goGitCheckout - checkout clonned repo to target branch
func goGitCheckout(PL GitPL) {

	// move to git directory
	os.Chdir("tmp/" + PL.project)

	// execute command to checkout on branch
	cmd := exec.Command("git", "checkout", PL.branch)
	cmd.Run()
	f.Println("Git: check out on branch", PL.branch)

	// move back to work directory
	os.Chdir("../../")
}

// ?? goGetGitRepo - get github repo we need as variable
func goGetGitRepo(ctx context.Context, gitClient *github.Client, targetRepo string) (rep github.Repository) {
	// get repository we are looking for
	repos, _, err := gitClient.Repositories.List(ctx, "", nil)
	if err != nil {
		f.Println("Could not authenticate git")
		return
	}
	var repo github.Repository
	for i := 0; i < len(repos); i++ {
		if *repos[i].Name == targetRepo {
			repo = *repos[i]
		}
	}
	return repo
}

func commentOnReview(GL GitPL, message string) {
	// TODO: comment on github
}

func findCommandInComment(PL GitPL) {
	if strings.Contains(PL.comment.body, "pullantis apply") {
		f.Println("found pullantis apply in comment - running apply")
		PL.command = "apply"
		f.Printf("%v", PL)
		queue.push(PL)
		// TODO: run pulumi "apply"
	} else if strings.Contains(PL.comment.body, "pullantis plan") {
		f.Println("found pullantis apply in comment - running plan")
		// TODO: run pulumi "plan"
	}
}

func pulumiPlan(PL GitPL) {
	os.Chdir("tmp/" + PL.project)

	pulumi.Run(func(ctx *pulumi.Context) error {
		return nil
	})

	os.Chdir("../../")

}

func pulumiApply(PL GitPL) {
	os.Chdir("tmp/" + PL.project)

	pulumi.Run(func(ctx *pulumi.Context) error {
		return nil
	})

	os.Chdir("../../")

}

func scanPL(PL GitPL) {
	goGitCheckout(PL)
	if PL.command == "apply" {
		pulumiApply(PL)
	} else {
		pulumiPlan(PL)
	}

}

func runApplication(PL GitPL) {
	// TODO: implement waiting system
	// TODO: implement pulumi
	// TODO: comment on github
	if queue.list[0].running != true {
		if queue.list[0].running != true {
			queue.list[0].running = true
			go scanPL(queue.list[0])
		}
	} else {
		commentOnReview(PL, "Pullantis is Busy")
	}
}

func handleQueue(PL GitPL) {
	// queueing system

	// if PL is created or pullantis is requested by comment, add element in queue.
	// 	if PL is closed, remove associated Queue element
	// run application after each receive to see if we can change anything
	if PL.action == "open" {
		f.Println("Received new PL - Adding to Queue")
		queue.push(PL)
	} else if PL.action == "created" {
		f.Println("Received new comment - searching")
		findCommandInComment(PL)
	} else if PL.action == "closed" {
		f.Println("Received PL close - removing from queue")
		queue.removeByID(PL.ID)
	}
	runApplication(PL)
}

// referrence - https://groob.io/tutorial/go-github-webhook/
func handleWebhook(w http.ResponseWriter, r *http.Request) {
	f.Println("Data received from GitHub WebHook") // f.Printf("headers: %v\n", r.Header)
	webhookData := make(map[string]interface{})

	err := json.NewDecoder(r.Body).Decode(&webhookData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if webhookData["action"] == nil {
		f.Println("No action found")
		return
	}

	var PL GitPL
	PL.ID = int(webhookData["number"].(float64))
	PL.action = webhookData["action"].(string)
	PL.project = webhookData["pull_request"].(map[string]interface{})["head"].(map[string]interface{})["repo"].(map[string]interface{})["name"].(string)

	if PL.action == "created" {
		PL.comment = GitComment{webhookData["comment"].(map[string]interface{})["body"].(string)}
	} else {
		PL.branch = webhookData["pull_request"].(map[string]interface{})["head"].(map[string]interface{})["ref"].(string)
	}
	handleQueue(PL)
}

func main() {

	os.RemoveAll("./tmp/")

	f.Println("Visit https://github.com/levankhelo/pullantis/blob/master/README.md for setup information")

	// Arguments
	var targetRepo = flag.String("repo", "pullantis", "GitHub repository name")
	var userGit = flag.String("git-user", "levankhelo", "GitHub Token")
	var tokenGit = flag.String("git-token", "", "GitHub Token")
	var tokenPulumi = flag.String("pulumi-token", "pul-6bc5c2d90a33078b307fd898d12683d70a26ca49", "Pulumi Token")
	var webhookGit = flag.String("webhook", "/events", "GitHub webhook tag")
	var localPort = flag.String("port", "4141", "local port for listener")
	var noServer = flag.Bool("no-server", false, "allow server listener")

	// Parse Arguments
	flag.Parse()

	// display arg values
	f.Println("----GIT----\n|\t", "\n|\tUser:", *userGit, "\n|\tToken:", *tokenGit, "\n|\tRepo:", *targetRepo)
	f.Println("---PULUMI--\n|\t", "\n|\tToken:", *tokenPulumi)
	f.Println("--WEBHOOK--\n|\t", "Hook:", *webhookGit, "\n|\tPort:", *localPort)

	// init git user using Access Token
	gitClient, ctx := goGitLogin(*tokenGit)

	// get target reository
	repo := goGetGitRepo(ctx, gitClient, *targetRepo)
	SetupCloseHandler()

	os.Mkdir("tmp/", 0700)
	os.Chdir("tmp/")
	goCloneGitRepo(*tokenGit, &repo)
	os.Chdir("../")

	if *noServer != true {
		log.Println("server started")
		http.HandleFunc(*webhookGit, handleWebhook)
		log.Fatal(http.ListenAndServe(":"+(*localPort), nil))
	} else {
		log.Println("building without server")
		os.RemoveAll("./tmp/")
	}

}

// clear; go build && go run main.go --git-user levankhelo --repo pullantis --webhook "/events" --port "4141"  --git-token 2868815ba4e32cb656ea9a8abd517a7b96ca2e2f --repo pullantis --pulumi-token "pul-6bc5c2d90a33078b307fd898d12683d70a26ca49"
