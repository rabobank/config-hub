package git_source

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/cloudfoundry-community/go-uaa/passwordcredentials"
	"golang.org/x/oauth2"
)

type miWifCredentials struct {
	tenantId    string
	name        string
	tokenSource oauth2.TokenSource
	cachedToken azcore.AccessToken

	mutex sync.Mutex
}

func (mwc *miWifCredentials) getFederatedToken(_ context.Context) (string, error) {
	if t, e := mwc.tokenSource.Token(); e != nil {
		fmt.Println("Error getting token from uaa", e)
		return "", e
	} else {
		return t.AccessToken, nil
	}
}

func (mwc *miWifCredentials) token() (string, error) {
	mwc.mutex.Lock()
	defer mwc.mutex.Unlock()

	if mwc.cachedToken.ExpiresOn.After(time.Now().Add(10 * time.Second)) {
		return mwc.cachedToken.Token, nil
	}

	if credential, e := azidentity.NewClientAssertionCredential(mwc.tenantId, mwc.name, mwc.getFederatedToken, nil); e != nil {
		return "", e
	} else if token, e := credential.GetToken(context.Background(), policy.TokenRequestOptions{
		Scopes: []string{"499b84ac-1321-427f-aa17-267ca6975798/.default"}}); e != nil {
		return "", e
	} else {
		mwc.cachedToken = token
	}

	return mwc.cachedToken.Token, nil
}

func newMiWifCredentials(tenantId, miName, tokenIssuer, clientId, secret, user, password string) (*miWifCredentials, error) {
	uaaCredentials := &passwordcredentials.Config{
		ClientID:     clientId,
		ClientSecret: secret,
		Username:     user,
		Password:     password,
		Endpoint:     oauth2.Endpoint{TokenURL: tokenIssuer},
		Scopes:       []string{"openid"},
	}
	tokenSource := uaaCredentials.TokenSource(context.Background())

	result := &miWifCredentials{
		tenantId:    tenantId,
		name:        miName,
		tokenSource: tokenSource,
	}

	// get a token to test it
	if _, e := result.token(); e != nil {
		return nil, e
	}

	return result, nil
}
