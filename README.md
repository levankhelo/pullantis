<hr> 

# pullantis

## Install pulumi
install `pulumi` using package manager 
- linux: `apt-get install pulumi` 
- Mac OS: `brew install pulumi` 

Now we need to create directory and 

## Start ngrok
install `ngrok` using package manager 
- linux: `apt-get install ngrok` 
- Mac OS: `brew install ngrok` 
```
ngrok http 4141
```
**save your ngrok's forwarded address**

<br> 
<hr> 
<br> 


## Setup GitHub WebHook
Go to your [GitHub](https://www.github.com) repo and move in `settings > webhooks`  
Click `Add Webhook`   
 
Payload URL:  `https://MYADRESS.ngrok.io/events` where `MYADRESS` is your Ngrok's adress forwarding

Content Type: `application/json`  

Secret: Any Secret text

Event Triggers:  `Let me select individual events.`  

- `Commit comments` 
- `Issue comments`
- `Pull request reviews`
- `Pushes` 
  
Click `Add Webhook` and that's it!

## Create GitHub Access Token
Create [https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token#creating-a-token](Access Token) and name it pullantis
> now we have `git-token`


 ## Retrieve Pulumi Access Token

 Visit [Pulumi](https://app.pulumi.com/) and go in `user` > `settings` > `Access Tokens`  
 Click `NEW ACCESS TOKEN` and input `pullantis` in description!   
 > Now we have `pulumi_token`


<hr> 
<br> 
<br> 

## using application 
to start application run:
```
go build 
go run main.go --git-user <GitHubUsername> --repo <GitHubRepoName> --webhook "/myHook" --port "<myPort>"  --git-token <git access token> --pulumi-token "<pulumi token>"
```
Arguments:
- **--git-user**: *(string)* your git username. example: `--git-user levankhelo`
- **--repo**: *(string)* your git repository with Pulumi.yaml in it. example: `--repo test-repo`
- **--webhook**: *(string)* webhook you created on github project. example: `--webhook "/events"`
- **--port**: *(string)/(int)* ngrok port. example: `--port "4141"`
- **--git-token**: *(string)* your git access token with permissions. example: `--git-token asdjhasbd*******asdasd`
- **--pulumi-token**: *(string)*you pulumi access token. example: `--pulumi-token ad-3adsasd****dasd`

Trigger Pullantis:
- create new Pull Request
- comment `pullantis plan` to execute planning
- comment `pullantis apply` to apply changes
  > queueing system will allow only 1 Pull Request to be monitored.  
  >  so you can only run pullantis (including commenting `plan` and `apply`) only on 1 Pull Request  
  >  if you will finish reviewing (`close` or `merge`) Pullantis move on next Queue element. example: if i had `PL-1` running and i had to scan it 10 times with pullantis plan, and at the same time, someone created `PL-2`, pullantis will tell `PL-2` that it is busy and will get to it when `PL-1` is merged/closed

Notes:
- if github repo we are looking at, does not support Pulumi (or does not have pullumi files), than you will always get `Pullantis failed` as PullRequest comment
  