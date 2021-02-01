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
	"bytes"
	"context"
	"encoding/json"
	"flag"
	f "fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	github "github.com/google/go-github/v33/github"
	"golang.org/x/oauth2"
)

/// GLOBAL VARIABLES
var gitClient *github.Client

//// STRUCTURES

// GitUser is current user got github client
type GitUser struct {
	username string
	token    string
	repo     string
}

var gituser GitUser

// GitComment is Review Comment on PL
type GitComment struct {
	body   string
	link   string
	header string
	id     string
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
	target  string     // target review for comment
	user    GitUser
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
	f.Printf("\nQueue push: %v\n", q.list)
	q.disp()
	return q
}
func (q Queue) put(el GitPL, index int) (qu Queue) {
	// put element at index of array
	if q.length <= index {
		f.Println("\nqueue.put ", index, "out of range. length:", q.length, len(q.list))
		return
	}

	oldList := q.list
	q.list = make([]GitPL, 0, 100)
	for i := 0; i < len(oldList); i++ {
		if i == index {
			q.list = append(q.list, el)
			f.Printf("\nPut: %v\n", el)
			q.length++
		}
		q.list = append(q.list, oldList[i])
	}
	q.list = append(q.list, el)
	q.length++
	f.Printf("\nQueue push: %v\n", q.list)
	q.disp()
	return q
}
func (q Queue) pop() (qu Queue) {
	// pop first element of array
	if q.length > 1 {
		q.list = q.list[1:]
	} else {
		q.list = make([]GitPL, 0, 100)
	}
	q.length--
	f.Printf("\nQueue pop: %v\n", q.list)
	q.disp()
	return q
}
func (q Queue) remove(index int) (qu Queue) {
	// rewrite elements by skipping indexed elements
	if q.length <= index {
		f.Println("\nqueue.remove ", index, "out of range. length:", q.length, len(q.list))
		return
	}

	oldList := q.list
	q.list = make([]GitPL, 0, 100)
	for i := 0; i < len(oldList); i++ {
		if i == index {
			f.Printf("\nRemoved: %v\n", oldList[i])
			q.length--
			continue
		}
		q.list = append(q.list, oldList[i])
	}
	f.Printf("\nQueue remove: %v\n", q.list)
	q.disp()
	return q
}
func (q Queue) removeByID(ID int) (qu Queue) {
	// run throught elements and find mathcing ID
	oldList := q.list
	q.list = make([]GitPL, 0, 100)
	for i := 0; i < len(oldList); i++ {
		if oldList[i].ID == ID {
			f.Println("\nQueue: popped element with ID:", ID)
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
	q.list = make([]GitPL, 0, 100)
	for i := 0; i < len(oldList); i++ {
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
	f.Printf("\n\n--Queue: %v\n", q.list)
}
func (q Queue) getByID(ID int) GitPL {
	q.disp()
	// run and return element by mathcing ID
	for i := 0; i < q.length; i++ {
		if q.list[i].ID == ID {
			return q.list[i]
		}
	}
	return GitPL{}
}
func (q Queue) findIfRunning(ID int) bool {

	// run and return element by mathcing ID
	for i := 0; i < len(q.list); i++ {
		if q.list[i].ID == ID && q.list[i].running {
			return true
		}
	}
	return false
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

func commentOnReview(PL GitPL, message string) {
	// TODO: comment on github
	// makeRequest(PL.comment.link, PL.comment.header, message, PL)
	makeRequest2(PL.comment.link, message, PL)
	// resp, err := http.PostForm(PL.target, url.Values{"body": {message}})
	// if err != nil {
	// f.Println("COMMENT FAILED")
	// return
	// }
	// f.Printf("\n\nCOMMENT RESPONSE: %v", resp)
	f.Printf("\n\n----COMMENT: %v", PL)
	f.Printf("\n------MESSAGE: %v", message)
}

func findCommandInComment(PL GitPL) {
	if strings.Contains(PL.comment.body, "pullantis apply") {
		f.Println("found pullantis apply in comment - running apply")
		PL.command = "apply"
		if queue.findIfRunning(PL.ID) {
			f.Println("Pullantis is already running on this issue. running apply")
			PL.running = true
			pulumiApply(PL)
			queue = queue.put(PL, 0)
		} else {
			queue = queue.push(PL)
		}

	} else if strings.Contains(PL.comment.body, "pullantis plan") {
		f.Println("found pullantis apply in comment - running plan")
		if queue.findIfRunning(PL.ID) {
			f.Println("Pullantis is already running on this issue. running plan")
			PL.running = true
			pulumiPlan(PL)
			queue = queue.put(PL, 0)
		} else {
			queue = queue.push(PL)
		}
	}
}

func pulumiPlan(PL GitPL) {
	f.Println("Pulumi plan")
	os.Chdir("tmp/" + PL.project)

	// cmd := exec.Command("pulumi", "refresh", "-y")
	// cmd.Run()
	cmd := exec.Command("pulumi", "refresh", "-y")

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()

	if err != nil {
		commentOnReview(PL, "Something went wrong during planning")
		log.Fatal(err)
	}
	commentOnReview(PL, "Pullantis: Pulumi refresh successfuly executed")
	f.Printf("\n\nPulumi Plan: %v", out.String())

	f.Println("Pulumi apply executed")
	os.Chdir("../../")

}

func pulumiApply(PL GitPL) {
	f.Println("Pulumi apply")
	os.Chdir("tmp/" + PL.project)

	// cmd := exec.Command("pulumi", "up", "-y")
	// cmd.Run()
	cmd := exec.Command("pulumi", "up", "-y")

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()

	if err != nil {
		commentOnReview(PL, "Something went wrong during apply")
		log.Fatal(err)
	}
	commentOnReview(PL, "Pullantis: Pulumi up Successfully executed")
	f.Printf("\n\nPulumi up: %v", out.String())
	f.Println("Pulumi apply executed")
	os.Chdir("../../")

}

func scanPL(PL GitPL) {
	if PL.command == "apply" {
		pulumiApply(PL)
	} else {
		pulumiPlan(PL)
	}

}

func runApplication(PL GitPL) {
	if queue.list[0].running != true {
		if queue.list[0].running != true {
			queue.list[0].running = true
			f.Printf("RUNNING %v", queue.list[0])
			goGitCheckout(PL)
			scanPL(queue.list[0]) // go scanPL(queue.list[0])
		}
	} else {
		commentOnReview(PL, "Pullantis is Busy")
	}
	queue.disp()
}

func handleQueue(PL GitPL) {
	// queueing system

	// if PL is created or pullantis is requested by comment, add element in queue.
	// 	if PL is closed, remove associated Queue element
	// run application after each receive to see if we can change anything
	f.Println(PL.action)
	if PL.action == "opened" {
		f.Println("Received new PL - Adding to Queue")
		queue = queue.push(PL)
	} else if PL.action == "created" {
		f.Println("Received new comment - searching")
		findCommandInComment(PL)
	} else if PL.action == "closed" {
		f.Println("Received PL close - removing from queue")
		if queue.length != 0 {
			queue = queue.removeByID(PL.ID)
		}
	}

	if queue.length+len(queue.list) == 0 {
		return
	}
	runApplication(PL)

}

func replaceInText(sub string, main string) {

}

// referrence - https://groob.io/tutorial/go-github-webhook/
func handleWebhook(w http.ResponseWriter, r *http.Request) {
	f.Println("Data received from GitHub WebHook") // f.Printf("headers: %v\n", r.Header)
	// f.Printf("\n\n%v\n\n")
	webhookData := make(map[string]interface{}, 10000)

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

	PL.project = webhookData["repository"].(map[string]interface{})["name"].(string)
	PL.action = webhookData["action"].(string)
	PL.user = gituser
	PL.comment = GitComment{}
	PL.comment.body = ""
	PL.comment.header = "Authorization: token " + PL.user.token
	if PL.action == "created" {
		explodeLink := strings.Split(webhookData["comment"].(map[string]interface{})["url"].(string), "/")
		PL.comment.id = explodeLink[len(explodeLink)-1]
		PL.comment.body = webhookData["comment"].(map[string]interface{})["body"].(string)
		PL.comment.link = webhookData["issue"].(map[string]interface{})["pull_request"].(map[string]interface{})["url"].(string) + "/comments/" + PL.comment.id + "/replies"
		PL.ID = int(webhookData["issue"].(map[string]interface{})["number"].(float64))
		PL.target = webhookData["issue"].(map[string]interface{})["url"].(string)
	} else {
		PL.comment.link = webhookData["pull_request"].(map[string]interface{})["comments_url"].(string)
		PL.ID = int(webhookData["number"].(float64))
		PL.target = webhookData["pull_request"].(map[string]interface{})["url"].(string)
		PL.branch = webhookData["pull_request"].(map[string]interface{})["head"].(map[string]interface{})["ref"].(string)
	}
	handleQueue(PL)
}

func makeRequest2(address string, text string, PL GitPL) {

	var jsonStr = []byte(`{"body":"` + text + `"}`)
	req, err := http.NewRequest("POST", address, bytes.NewBuffer(jsonStr))
	req.Header.Set("Authorization", "token "+PL.user.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	f.Println("response Status:", resp.Status)
	f.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	f.Println("response Body:", string(body))
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
	f.Println("----GIT----", "\n|\tUser:", *userGit, "\n|\tToken:", *tokenGit, "\n|\tRepo:", *targetRepo)
	f.Println("---PULUMI--", "\n|\tToken:", *tokenPulumi)
	f.Println("--WEBHOOK--", "\n|\tHook:", *webhookGit, "\n|\tPort:", *localPort)

	gituser.username = *userGit
	gituser.token = *tokenGit
	gituser.repo = *targetRepo
	// init git user using Access Token
	gitClient, ctx := goGitLogin(gituser.token)

	// get target reository
	repo := goGetGitRepo(ctx, gitClient, gituser.repo)
	SetupCloseHandler()

	os.Mkdir("tmp/", 0700)
	os.Chdir("tmp/")
	goCloneGitRepo(gituser.token, &repo)
	os.Chdir("../")

	// handle pulumi login
	os.Setenv("PULUMI_ACCESS_TOKEN", *tokenPulumi)
	exec.Command("pulumi", "login").Run()

	if *noServer != true {
		log.Println("server started")
		http.HandleFunc(*webhookGit, handleWebhook)
		log.Fatal(http.ListenAndServe(":"+(*localPort), nil))
	} else {
		log.Println("building without server")
		os.RemoveAll("./tmp/")
	}

}
