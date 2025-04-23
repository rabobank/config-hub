package git_source

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/rabobank/config-hub/util"
	"github.com/rabobank/credhub-client"
)

type spnCredentials struct {
	tenantId         string
	clientId         string
	cachedSecret     string
	secretExpiration time.Time
	cachedToken      azcore.AccessToken

	// optional credhub client and reference if secret is coming from credhub
	credhubClient credhub.Client
	credhubRef    *string

	mutex sync.Mutex
}

func (spnc *spnCredentials) secret() (string, error) {
	if spnc.credhubRef == nil || spnc.secretExpiration.After(time.Now()) {
		return spnc.cachedSecret, nil
	}

	spnc.mutex.Lock()
	defer spnc.mutex.Unlock()

	if spnc.credhubRef == nil || spnc.secretExpiration.After(time.Now()) {
		return spnc.cachedSecret, nil
	}

	if credential, e := spnc.credhubClient.GetJsonCredentialByName(*spnc.credhubRef); e != nil {
		return "", e
	} else {
		json.NewEncoder(os.Stdout).Encode(credential)
		spnc.cachedSecret = credential.Value["secret"].(string)
		spnc.secretExpiration = credential.VersionCreatedAt.Add(time.Hour * 24)
	}
	return spnc.cachedSecret, nil
}

func (spnc *spnCredentials) token() (string, error) {
	if spnc.cachedToken.ExpiresOn.After(time.Now().Add(10 * time.Second)) {
		return spnc.cachedToken.Token, nil
	}

	if secret, e := spnc.secret(); e != nil {
		return "", e
	} else if credential, e := azidentity.NewClientSecretCredential(spnc.tenantId, spnc.clientId, secret, nil); e != nil {
		// JV: maybe handle one retry, in case of 401, to handle racing conditions
		return "", e
	} else if token, e := credential.GetToken(context.Background(), policy.TokenRequestOptions{
		Scopes: []string{"499b84ac-1321-427f-aa17-267ca6975798/.default"}}); e != nil {
		return "", e
	} else {
		spnc.cachedToken = token
	}
	return spnc.cachedToken.Token, nil
}

func newSpnCredentials(tenantId, clientId string, secret, credhubClient, credhubSecret, credhubReference *string) (result *spnCredentials, e error) {
	result = &spnCredentials{
		tenantId:     tenantId,
		clientId:     clientId,
		cachedSecret: util.EmptyIfNil(secret),
		credhubRef:   credhubReference,
	}

	result.credhubClient, e = util.CredhubClient(credhubClient, credhubReference)
	return
}
