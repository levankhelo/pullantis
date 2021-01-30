package main

// import libraries
import (
	"encoding/json"
	flag "flag"
	f "fmt"
	"log"
	"net/http"
)

func CheckOpen() {

}

// webhook handler
// referrenced to https://groob.io/tutorial/go-github-webhook/
func handleWebhook(w http.ResponseWriter, r *http.Request) {
	f.Println("HookHandler: Data received")
	// fmt.Printf("headers: %v\n", r.Header)
	webhookData := make(map[string]interface{})

	err := json.NewDecoder(r.Body).Decode(&webhookData)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	f.Println("got webhook payload: ")

	switch action := webhookData["action"]; action {
	case "opened":

	default:
		f.Print("closed")
	}

	// for k := range webhookData {
	// f.Println("\n\n", k)
	// f.Printf("\n----------------------------\n\n\n%s : %v\n\n\n----------------------------\n\n\n", k, v)
	// }
}

func main() {

	f.Println("Visit https://github.com/levankhelo/pullantis/blob/master/README.md for setup information")

	// Arguments
	var userGit = flag.String("git-user", "", "GitHub username")
	var tokenGit = flag.String("git-token", "", "GitHub Token")
	var userPulumi = flag.String("pulumi-user", "", "Pulumi username")
	var tokenPulumi = flag.String("pulumi-token", "", "Pulumi Token")
	var webhookGit = flag.String("webhook", "/events", "GitHub webhook tag")
	var localPort = flag.String("port", "4141", "local port for listener")

	// Parse Arguments
	flag.Parse()

	f.Println("----GIT----\n|\t", "User:", *userGit, "\n|\tToken:", *tokenGit)
	f.Println("---PULUMI--\n|\t", "User:", *userPulumi, "\n|\tToken:", *tokenPulumi)
	f.Println("--WEBHOOK--\n|\t", "Hook:", *webhookGit, "\n|\tPort:", *localPort)

	log.Println("server started")
	http.HandleFunc(*webhookGit, handleWebhook)
	log.Fatal(http.ListenAndServe(":"+(*localPort), nil))

}
