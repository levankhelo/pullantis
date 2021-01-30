package main

// import libraries
import (
	flag "flag"
	f "fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	// taken from https://groob.io/tutorial/go-github-webhook/
	f.Printf("headers: %v\n", r.Header)

	_, err := io.Copy(os.Stdout, r.Body)
	if err != nil {
		log.Println(err)
		return
	}
}

func main() {

	f.Println("Visit https://github.com/levankhelo/pullantis/blob/master/README.md for setup information")

	// Arguments
	var userGit = flag.String("git-user", "", "GitHub username")
	var tokenGit = flag.String("git-token", "", "GitHub Token")
	var userPulumi = flag.String("pulumi-user", "", "Pulumi username")
	var tokenPulumi = flag.String("pulumi-token", "", "Pulumi Token")
	var webhookGit = flag.String("webhook", "/events", "GitHub webhook tag")

	// Parse Arguments
	flag.Parse()

	f.Println("----GIT----\n", "user: ", *userGit, "\nToken: ", *tokenGit)
	f.Println("---PULUMI--\n", "user: ", *userPulumi, "\nToken: ", *tokenPulumi)
	f.Println("--WEBHOOK--\n", "hook: ", *webhookGit)

	http.HandleFunc(*webhookGit, handleWebhook)

}
