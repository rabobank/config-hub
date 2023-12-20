package server

import (
	"encoding/json"
	"fmt"
	"time"

	"config-hub/cfg"
	"config-hub/util"
	"github.com/gomatbase/go-we"
	"github.com/gomatbase/go-we/security"
)

const (
	PermissionsUrl = "%s/v3/service_instances/%s/permissions"
	UsersUrl       = "%s/Users/%s"
)

type CfServiceInstancePermissions struct {
	Read   bool `json:"read"`
	Manage bool `json:"manage"`
}

type UaaUser struct {
	Id         string `json:"id"`
	ExternalId string `json:"externalId"`
	Meta       struct {
		Version      int       `json:"version"`
		Created      time.Time `json:"created"`
		LastModified time.Time `json:"lastModified"`
	} `json:"meta"`
	UserName string `json:"userName"`
	Name     struct {
		FamilyName string `json:"familyName"`
		GivenName  string `json:"givenName"`
	} `json:"name"`
	Emails []struct {
		Value   string `json:"value"`
		Primary bool   `json:"primary"`
	} `json:"emails"`
	Groups []struct {
		Value   string `json:"value"`
		Display string `json:"display"`
		Type    string `json:"type"`
	} `json:"groups"`
	Approvals []struct {
		UserId        string    `json:"userId"`
		ClientId      string    `json:"clientId"`
		Scope         string    `json:"scope"`
		Status        string    `json:"status"`
		LastUpdatedAt time.Time `json:"lastUpdatedAt"`
		ExpiresAt     time.Time `json:"expiresAt"`
	} `json:"approvals"`
	PhoneNumbers []struct {
		Value string `json:"value"`
	} `json:"phoneNumbers"`
	Active               bool      `json:"active"`
	Verified             bool      `json:"verified"`
	Origin               string    `json:"origin"`
	ZoneId               string    `json:"zoneId"`
	PasswordLastModified time.Time `json:"passwordLastModified"`
	PreviousLogonTime    int64     `json:"previousLogonTime"`
	LastLogonTime        int64     `json:"lastLogonTime"`
	Schemas              []string  `json:"schemas"`
}

func isDeveloper(user *security.User, scope we.RequestScope) bool {
	if user == nil {
		return false
	}

	bearerToken := scope.Request().Header.Get("Authorization")
	if len(bearerToken) == 0 {
		// call is not authenticated with a bearer token, the user should have the token in the metadata
		if token, isTokenData := user.Data.(*security.TokenData); !isTokenData {
			// can't get a token to validate
			return false
		} else {
			bearerToken = "Bearer " + token.Raw
		}
	}

	body, e := util.Request(fmt.Sprintf(PermissionsUrl, cfg.CfUrl, cfg.ServiceInstanceId)).WithAuthorization(bearerToken).Get()
	if e != nil {
		// log it
	} else {
		var permissions CfServiceInstancePermissions
		if e = json.Unmarshal(body, &permissions); e != nil {
			fmt.Printf("Unable to check user %s permissions for service %s: %v\n", user.Username, cfg.ServiceInstanceId, e)
		} else if permissions.Manage {
			return true
		} else {
			fmt.Printf("[AUTH] User %s has no permissions to manage requested service %s\n", user.Username, cfg.ServiceInstanceId)
		}
	}
	return false
}

func enrichUaaUser(user *security.User) (*security.User, error) {
	uaaUser := new(UaaUser)
	if tokenData, isType := user.Data.(*security.TokenData); !isType {
		// We expect authentication to have a token, so... it should never happen
		return user, nil
	} else if body, e := util.Request(fmt.Sprintf(UsersUrl, cfg.UaaUrl, user.OriginId)).WithBearerToken(tokenData.Raw).Get(); e != nil {
		return user, e
	} else if e = json.Unmarshal(body, uaaUser); e != nil {
		fmt.Println(string(body), e)
		return user, e
	} else {
		user.Username = uaaUser.UserName
		// Let's completely replace the scopes with what is returned from the user service
		user.Scopes = make([]string, len(uaaUser.Groups))
		for i, group := range uaaUser.Groups {
			user.Scopes[i] = group.Display
		}
	}

	return user, nil
}
