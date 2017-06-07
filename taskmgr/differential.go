package taskmgr

import (
	"github.com/danieldanciu/gonduit/entities"
	"github.com/danieldanciu/gonduit/requests"
	"errors"
	"fmt"
	"github.com/danieldanciu/gonduit"
	"github.com/danieldanciu/gonduit/core"

	"github.com/daedaleanai/reqtraq/linepipes"
)

var cachedApiToken string

//TODO(danieldanciu) - avoid duplication of getApiToken and getApiClient in maniphest.go

// getApiToken returns the Phabricator API token that allows us to make authenticated API operations.
func getApiToken() (string, error) {
	if cachedApiToken != "" {
		return cachedApiToken, nil
	}

	// The Phabricator API token comes from https://p.daedalean.ai/settings/user/git/page/apitokens/ (the already
	// generated Standard Api Token is good to use
	//
	// set it on the git server repository like this:
	// $ ssh git.daedalean.ai
	// $ cd /var/git/exp.git
	// $ sudo -u git sh -c "git config --local --replace-all daedalean.taskmgr-api-token <TOKEN>"
	apiToken, err := linepipes.Single(linepipes.Run("git", "config", "--get", "daedalean.taskmgr-api-token"))
	if err != nil {
		msg := `No Phabricator API token set. Please go to
	https://p.daedalean.ai/settings/user/<YOUR_USERNAME_HERE>/page/apitokens/
click on <Generate API Token>, and then paste the token into this command
	git config --local --replace-all daedalean.taskmgr-api-token <PASTE_TOKEN_HERE>`
		return "", fmt.Errorf(msg)
	}
	cachedApiToken = apiToken
	return apiToken, nil
}

// getApiClient returns a client connection to the Phabricator server
func getApiClient() (*gonduit.Conn, error) {
	apiToken, err := getApiToken()
	if err != nil {
		return nil, err
	}
	return gonduit.Dial("https://p.daedalean.ai", &core.ClientOptions{APIToken: apiToken})
}

// GetRevision returns the Differential revision with the given ID or an error if the revision is not found
func GetRevision(revisionID uint64) (*entities.DifferentialRevision, error) {
	client, err := getApiClient()
	if err != nil {
		return nil, err
	}
	res, err := client.DifferentialQuery(requests.DifferentialQueryRequest{IDs: []uint64{revisionID}})
	if err != nil {
		return nil, err
	}
	if len(*res) == 0 {
		return nil, errors.New("Cannot be found")
	}
	return (*res)[0], nil
}
