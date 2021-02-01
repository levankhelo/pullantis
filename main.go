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
var logging bool

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
	list []GitPL
}

func (q Queue) push(el GitPL) (qu Queue) {
	// push element at the end of array

	q.list = append(q.list, el)

	if logging {
		queue.disp()
	}
	return q
}
func (q Queue) put(el GitPL, index int) (qu Queue) {
	// put element at index of array

	// check if index is out of range, return
	if len(q.list) <= index {
		f.Println("Queue.put", index, "out of range. length:", len(q.list))
		return
	}

	// store old array and restructure new one based on action
	oldList := q.list
	q.list = make([]GitPL, 0, 100)

	for i := 0; i < len(oldList); i++ {
		if i == index {

			q.list = append(q.list, el)
			if logging {
				f.Println("|Queue: Put ", el)
			}

		}

		q.list = append(q.list, oldList[i])
	}

	q.list = append(q.list, el)

	if logging {
		queue.disp()
	}
	return q
}
func (q Queue) pop() (qu Queue) {
	// pop first element of array

	if len(q.list) > 1 {
		q.list = q.list[1:]
	} else {
		q.list = make([]GitPL, 0, 100)
	}

	f.Printf("\nQueue pop: %v\n", q.list)

	if logging {
		queue.disp()
	}
	return q
}
func (q Queue) remove(index int) (qu Queue) {
	// remove element by index

	// check if index is out of range, return
	if len(q.list) <= index {
		f.Println("Queue.remove", index, "out of range. length:", len(q.list))
		return
	}

	// store old array and restructure new one based on action
	oldList := q.list
	q.list = make([]GitPL, 0, 100)

	for i := 0; i < len(oldList); i++ {
		if i == index {

			if logging {
				f.Println("|Queue: removed ", oldList[i])
			}

			continue

		}

		q.list = append(q.list, oldList[i])
	}

	if logging {
		queue.disp()
	}
	return q
}
func (q Queue) removeByID(ID int) (qu Queue) {
	// run throught elements and remove elemet by mathcing ID

	// store old array and restructure new one based on action
	oldList := q.list
	q.list = make([]GitPL, 0, 100)

	for i := 0; i < len(oldList); i++ {
		if oldList[i].ID == ID {

			if logging {
				f.Println("|Queue: popped element with ID:", ID)
			}

			continue

		}

		q.list = append(q.list, oldList[i])
	}

	if logging {
		queue.disp()
	}
	return q
}
func (q Queue) removeByObject(obj GitPL) (qu Queue) {
	// run throught elements and remove element by mathcing structure

	oldList := q.list
	q.list = make([]GitPL, 0, 100)

	for i := 0; i < len(oldList); i++ {
		if oldList[i] == obj {

			continue

		}

		q.list = append(q.list, oldList[i])
	}

	if logging {
		queue.disp()
	}
	return q
}
func (q Queue) disp() {
	f.Printf("\n--Queue: %v\n", q.list)
}
func (q Queue) getByID(ID int) GitPL {
	// run throught elements and return one with mathcing ID

	for i := 0; i < len(q.list); i++ {
		if q.list[i].ID == ID {

			return q.list[i]

		}
	}

	return GitPL{}
}
func (q Queue) findIfRunning(ID int) bool {
	// run throught elements and check if any element with ID is running

	for i := 0; i < len(q.list); i++ {
		if q.list[i].ID == ID && q.list[i].running {

			return true

		}
	}

	return false
}
func (q Queue) getLast() (pl GitPL) {
	// return last (pushed) element of queue

	return q.list[len(q.list)-1]
}

var queue Queue

//// FUNCTIONS

// SetupCloseHandler - handle clean up after exit
func SetupCloseHandler() {

	// make waiter for SIGKILL
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// new thread that handles cleanup on SIGKILL
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

	exec.Command("git", "clone", gitCommand).Run()

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
	/*
		Args:
			PL: (GitPL) Pull Request Structure that carries GitHub's parsed json
	*/

	os.Chdir("tmp/" + PL.project)

	exec.Command("git", "fetch", "--all").Run()
	exec.Command("git", "checkout", PL.branch).Run()

	f.Println("| Git: check out on branch", PL.branch)

	os.Chdir("../../")
}

// goGetGitRepo - get targetted github repo we need as structure
func goGetGitRepo(ctx context.Context, gitClient *github.Client, targetRepo string) (rep github.Repository) {
	/*
		Args:
			ctx: (context.Context)
			gitClient: (*github.Client) Github client where user, repo and other data is stored
			targetRepo: (string) name of repo we are running pullantis on

		Return:
			rep: (github.Repository) github repository as type with all nformation about it
	*/
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
	/*
		Args:
			PL: (GitPL) Pull Request Structure that carries GitHub's parsed json
			message: (string) message that should be send to GitHub api (written as comment)
	*/

	// store message in byte arrat and POST it to PL's address
	var jsonStr = []byte(`{"body":"` + message + `"}`)
	req, err := http.NewRequest("POST", PL.comment.link, bytes.NewBuffer(jsonStr))
	req.Header.Set("Authorization", "token "+PL.user.token)
	req.Header.Set("Content-Type", "application/json")

	// try catch request sendind and close if everything goes well
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// display output
	/*
		f.Println("response Status:", resp.Status); f.Println("response Headers:", resp.Header)
		body, _ := ioutil.ReadAll(resp.Body)
		f.Println("response Body:", string(body))
	*/
}

func findCommandInComment(PL GitPL) {
	/*
		Args:
			PL: (GitPL) Pull Request Structure that carries GitHub's parsed json
	*/

	if strings.Contains(PL.comment.body, "Pullantis:") == true {

	} else if strings.Contains(PL.comment.body, "pullantis apply") || strings.Contains(PL.comment.body, "pullantis pply") {
		// check if string contains pullantis apply

		f.Println(" |  |---- found pullantis apply in comments: running apply")

		PL.command = "apply"

		if queue.findIfRunning(PL.ID) {
			f.Println("|   |---- Pullantis is already running on this issue: running apply")
			PL.running = true
			scanPL(PL)
			queue = queue.put(PL, 0)
		} else {
			f.Println("|   |---- Pullantis is not running on this issue: adding in queue")
			queue = queue.push(PL)
		}

	} else if strings.Contains(PL.comment.body, "pullantis plan") {
		f.Println("|   |---- found pullantis plan in comments: running plan")
		if queue.findIfRunning(PL.ID) {
			f.Println("|   |---- Pullantis is already running on this issue: running plan")
			PL.running = true
			scanPL(PL)
			queue = queue.put(PL, 0)
		} else {
			f.Println("|   |---- Pullantis is not running on this issue: adding in queue")
			queue = queue.push(PL)
		}
	} else {
		f.Println("|   |---- Pullantis command not found in string")
	}
}

// pulumiPlan - run pulumi refresh on PL branch and post results as comment
func pulumiPlan(PL GitPL) {
	/*
		Args:
			PL: (GitPL) Pull Request Structure that carries GitHub's parsed json
	*/
	f.Println("|-- Pulumi plan")
	os.Chdir("tmp/" + PL.project)

	// execute command and store output
	cmd := exec.Command("pulumi", "preview", "-y")

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()

	// post command output on github as PL comment
	if err != nil {
		commentOnReview(PL, "Pullantis: Something went wrong during planning")
		return
	}
	commentOnReview(PL, "Pullantis: Pulumi refresh successfuly executed")

	f.Printf("\n\nPulumi Plan: %v", out.String())
	f.Println("Pulumi apply executed")
	os.Chdir("../../")
}

// pulumiApply - run pulumi up on PL branch and post results as comment
func pulumiApply(PL GitPL) {
	/*
		Args:
			PL: (GitPL) Pull Request Structure that carries GitHub's parsed json
	*/

	f.Println("|-- Pulumi apply")
	os.Chdir("tmp/" + PL.project)

	// execute command and store output
	cmd := exec.Command("pulumi", "up", "-y")

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()

	// post command output on github as PL comment
	if err != nil {
		commentOnReview(PL, "Pullantis: Something went wrong during apply")
		return
	}
	commentOnReview(PL, "Pullantis: Pulumi up Successfully executed")

	f.Printf("\n\nPulumi up: %v", out.String())
	f.Println("Pulumi apply executed")
	os.Chdir("../../")

}

// scanPL - check PL.command and run pulumi up or pulumi refresh based on that
func scanPL(PL GitPL) {
	/*
		Args:
			PL: (GitPL) Pull Request Structure that carries GitHub's parsed json
	*/

	if PL.command == "apply" {
		pulumiApply(PL)
	} else {
		pulumiPlan(PL)
	}

}

// runApplication runs 0th element of queue and scans it. otherwise comments that we are busy
func runApplication(PL GitPL) {
	/*
		Args:
			PL: (GitPL) Pull Request Structure that carries GitHub's parsed json
	*/

	if queue.list[0].running != true {
		if queue.list[0].running != true {

			queue.list[0].running = true
			if logging {
				queue.disp()
			}
			goGitCheckout(PL)
			scanPL(queue.list[0])

		}
	} else {
		commentOnReview(PL, "Pullantis is Busy")
	}

	if logging {
		queue.disp()
	}

}

// handleQueue handles actions and redirects PLs by their types
func handleQueue(PL GitPL) {
	/*
		Args:
			PL: (GitPL) Pull Request Structure that carries GitHub's parsed json
	*/

	f.Println("|")
	f.Println("| Action:", PL.action)
	f.Println("|")

	if PL.action == "opened" || PL.action == "reopened" {
		// is PL is opened than push it into queue

		f.Println("|--- Adding to Queue")

		queue = queue.push(PL)

	} else if PL.action == "created" {
		// if comment is added, check body of comment

		f.Println("|--- Parsing comment")

		findCommandInComment(PL)
		return

	} else if PL.action == "closed" {
		// if PL is closed, remove all associations from queue

		f.Println("|--- Removing from Queue")

		if len(queue.list) != 0 {
			queue = queue.removeByID(PL.ID)
		}

	}

	// if queue is empty, than dont do anything
	if len(queue.list) == 0 {
		return
	}

	// run application on PL
	runApplication(PL)

}

// handleWebhook - receives and handles http responses from github webhooks
func handleWebhook(w http.ResponseWriter, r *http.Request) {
	/*
		Args: automatically passed by http.HandleFunc
			w: (http.ResponseWriter) response that will be returned to sender using w.Write()
			r: (*http.Request) Request received fom webhook
	*/

	// make structure to parse received json in
	f.Println("\nData received from GitHub WebHook")
	webhookData := make(map[string]interface{}, 10000)

	// decode json into structure and check errors
	err := json.NewDecoder(r.Body).Decode(&webhookData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// if json has no action, than don't do anything
	if webhookData["action"] == nil {
		f.Println("No action found")
		return
	}

	// initialize GitPL structure and parse event data
	var PL GitPL

	PL.project = webhookData["repository"].(map[string]interface{})["name"].(string)
	PL.action = webhookData["action"].(string)
	PL.user = gituser
	PL.comment = GitComment{}
	PL.comment.body = ""
	PL.comment.header = "Authorization: token " + PL.user.token
	if PL.action == "created" {
		// handle github comment data storing (action - create)
		explodeLink := strings.Split(webhookData["comment"].(map[string]interface{})["url"].(string), "/")
		PL.comment.id = explodeLink[len(explodeLink)-1]
		PL.comment.body = webhookData["comment"].(map[string]interface{})["body"].(string)
		PL.comment.link = webhookData["issue"].(map[string]interface{})["comments_url"].(string)
		PL.ID = int(webhookData["issue"].(map[string]interface{})["number"].(float64))
		PL.target = webhookData["issue"].(map[string]interface{})["url"].(string)
	} else {
		// handle PL open, reopen and close data storing
		PL.comment.link = webhookData["pull_request"].(map[string]interface{})["comments_url"].(string)
		PL.ID = int(webhookData["number"].(float64))
		PL.target = webhookData["pull_request"].(map[string]interface{})["url"].(string)
		PL.branch = webhookData["pull_request"].(map[string]interface{})["head"].(map[string]interface{})["ref"].(string)
	}

	// redirect initialized PL to queue handler
	handleQueue(PL)
}

func main() {

	// cleanup temporary directory on restart in case something goes wrong
	os.RemoveAll("./tmp/")

	f.Println("Visit https://github.com/levankhelo/pullantis/blob/master/README.md for more information")

	// Arguments
	var targetRepo = flag.String("repo", "pullantis", "GitHub repository name")
	var userGit = flag.String("git-user", "levankhelo", "GitHub Token")
	var tokenGit = flag.String("git-token", "", "GitHub Token")
	var tokenPulumi = flag.String("pulumi-token", "", "Pulumi Token")
	var webhookGit = flag.String("webhook", "/events", "GitHub webhook tag")
	var localPort = flag.String("port", "4141", "local port for listener")
	var loggingEnabled = flag.Bool("logging", false, "enable queue logging")

	logging = *loggingEnabled

	// Parse Arguments
	flag.Parse()

	// display arg values
	f.Println("----GIT----", "\n|\tUser:", *userGit, "\n|\tToken:", *tokenGit, "\n|\tRepo:", *targetRepo)
	f.Println("---PULUMI--", "\n|\tToken:", *tokenPulumi)
	f.Println("--WEBHOOK--", "\n|\tHook:", *webhookGit, "\n|\tPort:", *localPort)
	f.Println()

	// store
	gituser.username = *userGit
	gituser.token = *tokenGit
	gituser.repo = *targetRepo
	// init git user using Access Token
	gitClient, ctx := goGitLogin(gituser.token)

	// get target reository
	repo := goGetGitRepo(ctx, gitClient, gituser.repo)

	// setup close handler that removes temporary directories
	SetupCloseHandler()

	// make temporary directory and clone repo
	os.Mkdir("tmp/", 0700)
	os.Chdir("tmp/")
	goCloneGitRepo(gituser.token, &repo)
	os.Chdir("../")

	// handle pulumi login
	os.Setenv("PULUMI_ACCESS_TOKEN", *tokenPulumi)
	exec.Command("pulumi", "login").Run()

	f.Println()
	log.Println("server started")
	http.HandleFunc(*webhookGit, handleWebhook)
	log.Fatal(http.ListenAndServe(":"+(*localPort), nil))

}
