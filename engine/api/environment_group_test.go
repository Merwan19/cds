package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestAddGroupsInEnvironmentHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/TestAddGroupsInEnvironmentHandler")
	router.init()

	//1. Create admin user
	u, pass := assets.InsertAdminUser(db)

	//2. Create project
	proj := assets.InsertTestProject(t, db, sdk.RandomString(10), sdk.RandomString(10), nil)
	test.NotNil(t, proj)

	//3. Create environment
	envProd := &sdk.Environment{Name: sdk.RandomString(10), ProjectID: proj.ID}
	test.NoError(t, environment.InsertEnvironment(db, envProd))

	//4. Create Group
	grp1 := &sdk.Group{Name: sdk.RandomString(10)}
	_, _, errG := group.AddGroup(db, grp1)
	test.NoError(t, errG)

	grp2 := &sdk.Group{Name: sdk.RandomString(10)}
	_, _, errG2 := group.AddGroup(db, grp2)
	test.NoError(t, errG2)

	//5. Prepare request
	gps := []sdk.GroupPermission{
		{
			Permission: 7,
			Group:      *grp1,
		},
		{
			Permission: 4,
			Group:      *grp2,
		},
	}

	jsonBody, _ := json.Marshal(gps)
	body := bytes.NewBuffer(jsonBody)

	vars := map[string]string{
		"key": proj.Key,
		"permEnvironmentName": envProd.Name,
	}

	//Prepare request
	uri := router.getRoute("POST", addGroupsInEnvironmentHandler, vars)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("POST", uri, body)
	assets.AuthentifyRequest(t, req, u, pass)

	//4. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	envUpdated := &sdk.Environment{}
	json.Unmarshal(res, &envUpdated)

	grp1Found := false
	grp2Found := false

	for _, gp := range envUpdated.EnvironmentGroups {
		if gp.Group.Name == grp1.Name {
			grp1Found = true
			assert.Equal(t, 7, gp.Permission)
		}
		if gp.Group.Name == grp2.Name {
			grp2Found = true
			assert.Equal(t, 4, gp.Permission)
		}
	}

	assert.True(t, grp1Found)
	assert.True(t, grp2Found)
}

func TestUpdateGroupRoleOnEnvironmentHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/TestUpdateGroupRoleOnEnvironmentHandler")
	router.init()

	//1. Create admin user
	u, pass := assets.InsertAdminUser(db)

	//2. Create project
	proj := assets.InsertTestProject(t, db, sdk.RandomString(10), sdk.RandomString(10), nil)
	test.NotNil(t, proj)

	//3. Create environment
	envProd := &sdk.Environment{Name: sdk.RandomString(10), ProjectID: proj.ID}
	test.NoError(t, environment.InsertEnvironment(db, envProd))

	//4. Create Group
	grp1 := &sdk.Group{Name: sdk.RandomString(10)}
	_, _, errG := group.AddGroup(db, grp1)
	test.NoError(t, errG)

	grp2 := &sdk.Group{Name: sdk.RandomString(10)}
	_, _, errG2 := group.AddGroup(db, grp2)
	test.NoError(t, errG2)

	//5. Add group on environment
	test.NoError(t, group.InsertGroupInEnvironment(db, envProd.ID, grp1.ID, 7))
	test.NoError(t, group.InsertGroupInEnvironment(db, envProd.ID, grp2.ID, 7))

	//6. Prepare request
	gp := sdk.GroupPermission{
		Permission: 4,
		Group:      *grp1,
	}

	jsonBody, _ := json.Marshal(gp)
	body := bytes.NewBuffer(jsonBody)

	vars := map[string]string{
		"key": proj.Key,
		"permEnvironmentName": envProd.Name,
		"group":               grp1.Name,
	}

	//Prepare request
	uri := router.getRoute("PUT", updateGroupRoleOnEnvironmentHandler, vars)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("PUT", uri, body)
	assets.AuthentifyRequest(t, req, u, pass)

	//4. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	envUpdated := &sdk.Environment{}
	json.Unmarshal(res, &envUpdated)

	grp1Found := false

	for _, gp := range envUpdated.EnvironmentGroups {
		if gp.Group.Name == grp1.Name {
			grp1Found = true
			assert.Equal(t, 4, gp.Permission)
		}
	}

	assert.True(t, grp1Found)
}
