package main

// import libraries
import (
	flag "flag"
	f "fmt"
)

func main() {

	f.Println("Visit https://github.com/levankhelo/pullantis/blob/master/README.md for more information")

	// Arguments
	var user_git = flag.String("git-user", "", "GitHub username")
	var token_git = flag.String("git-token", "", "GitHub Token")
	var user_pulumi = flag.String("pulumi-user", "", "Pulumi username")
	var token_pulumi = flag.String("pulumi-token", "", "Pulumi Token")

	// Parse Arguments
	flag.Parse()

	f.Println("----GIT----\n", "user: ", *user_git, "\nToken: ", *token_git)
	f.Println("----PUL----\n", "user: ", *user_pulumi, "\nToken: ", *token_pulumi)

	// os.Setenv()

	// pulumi.Run(func(ctx *pulumi.Context) error {
	// 	if err != nil {
	// 		return err
	// 	}

	// 	return nil
	// })
}
