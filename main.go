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
	"syscall"

	github "github.com/google/go-github/v33/github"
	"golang.org/x/oauth2"
)

//// STRUCTURES

// GitPL is Pull request structure
type GitPL struct {
	project string
	branch  string // branch name -
	action  string // action done: comment pull request close -> created
	ID      int    // ID of pull request -> pull_request.number
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
		f.Println("queue.remove ", index, "out of range")
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
		f.Println("Exited")
		os.Exit(0)
	}()
}

// goCloneGitRepo - Clone GitHub Repo in tmp/ directory
func goCloneGitRepo(tokenGit string, repo *github.Repository) {
	f.Printf("\n\n%v\n\n", *repo.CloneURL)
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
func goGitCheckout(branch string, project string) {

	// move to git directory
	os.Chdir("tmp/" + project)
	f.Println("Moved in ", "tmp/"+project, "directory")

	// execute command to checkout on branch
	cmd := exec.Command("git", "checkout", branch)
	cmd.Run()
	f.Println("Git: check out on branch", branch)

	// move back to work directory
	os.Chdir("../../")
	f.Println("Moved back to work directory")
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
		f.Println(*repos[i].Name)
		if *repos[i].Name == targetRepo {
			repo = *repos[i]
		}
	}
	return repo
}

func runPulumiPlan(PL GitPL) {
	os.Chdir("tmp/" + PL.project)
}

func commentOnReview() {

}

func handleQueue(PL GitPL) {
	if queue.length == 0 {
		goGitCheckout(PL.branch, PL.project)
		runPulumiPlan(PL)
		return
	}
	f.Println("NOPE")
	queue = queue.push(PL)
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
	PL.ID = webhookData["number"].(int)
	PL.action = webhookData["action"].(string)
	PL.branch = webhookData["pull_request"].(map[string]interface{})["head"].(map[string]interface{})["ref"].(string)
	PL.project = webhookData["pull_request"].(map[string]interface{})["head"].(map[string]interface{})["repo"].(map[string]interface{})["name"].(string)

	handleQueue(PL)

	f.Printf("%v", PL)
}

func main() {

	os.RemoveAll("./tmp/")

	f.Println("Visit https://github.com/levankhelo/pullantis/blob/master/README.md for setup information")

	// Arguments
	var targetRepo = flag.String("repo", "pullantis", "GitHub repository name")
	var userGit = flag.String("git-user", "levankhelo", "GitHub Token")
	var tokenGit = flag.String("git-token", "", "GitHub Token")
	var tokenPulumi = flag.String("pulumi-token", "", "Pulumi Token")
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
	f.Printf("\n|%v\n\n\n", repo)
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
