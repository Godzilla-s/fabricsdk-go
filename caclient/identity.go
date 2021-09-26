package caclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"
	"io"
	"net/http"
	"strconv"
)

type identity struct {
	key    cryptoutil.Key
	cert   []byte
	client *Client
}

func addQueryParm(req *http.Request, name, value string) {
	url := req.URL.Query()
	url.Add(name, value)
	req.URL.RawQuery = url.Encode()
}

func marshal(req interface{}, id string) ([]byte, error) {
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf( "%v; Failed to marshal %s", err, id)
	}
	return buf, nil
}

func (id *identity) setAuthToken(req *http.Request, body []byte) error {
	token, err := cryptoutil.GenECDSAToken(id.cert, id.key, req.Method, req.URL.RequestURI(), body)
	if err != nil {
		return err
	}
	req.Header.Set("authorization", token)
	return nil
}

func (id *identity) newHTTPRequest(method, uri string, body io.Reader) (*http.Request, error) {
	return id.client.newRequest(method, uri, body)
}

func (id *identity) post(uri string, reqBody []byte, result interface{}, queryParam map[string]string) error {
	req, err := id.newHTTPRequest("POST", uri, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	if queryParam != nil {
		for key, val := range queryParam {
			addQueryParm(req, key, val)
		}
	}
	if err = id.setAuthToken(req, reqBody); err != nil {
		return err
	}
	return id.client.send(req, result)
}

func (id *identity) get(uri, caname string, result interface{}) error {
	req, err := id.newHTTPRequest("GET", uri, bytes.NewReader([]byte{}))
	if err != nil {
		return err
	}
	if caname != "" {
		addQueryParm(req, "ca", caname)
	}
	err = id.setAuthToken(req, nil)
	if err != nil {
		return fmt.Errorf("addAuthToken: %v", err)
	}
	return id.client.send(req, result)
}

func (id *identity) put(uri string, queryParam map[string]string, reqBody []byte, result interface{}) error {
	req, err := id.newHTTPRequest("PUT", uri, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	if queryParam != nil {
		for key, value := range queryParam {
			addQueryParm(req, key, value)
		}
	}
	err = id.setAuthToken(req, reqBody)
	if err != nil {
		return err
	}
	return id.client.send(req, result)
}

func (id *identity) delete(uri string, result interface{}, queryParam map[string]string) error {
	req, err := id.newHTTPRequest("DELETE", uri, bytes.NewReader([]byte{}))
	if err != nil {
		return err
	}
	if queryParam != nil {
		for key, value := range queryParam {
			addQueryParm(req, key, value)
		}
	}
	err = id.setAuthToken(req, nil)
	if err != nil {
		return err
	}
	return id.client.send(req, result)
}

// POST http://localhost:7054/register
func (id *identity) addIdentity(req RegistrationRequest) (*RegistrationResponse, error) {
	reqBody, err := marshal(req, "RegistrationRequest")
	if err != nil {
		return nil, fmt.Errorf("Marshal: %v", err)
	}
	var result RegistrationResponse
	err = id.post("register", reqBody, &result, nil)
	if err != nil {
		return nil, fmt.Errorf("Post Error: %v", err)
	}
	return &result, nil
}

// GET http://{username}:{password}@localhost:7054/identities
func (id *identity) getIdentity(name, caname string) (*GetIdentityResponse, error) {
	var result GetIdentityResponse
	err := id.get(fmt.Sprintf("identities/%s", name), caname, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DELETE http://localhost:7054/identities/{$id}
func (id *identity) delIdentity(name, caname string, force bool) error {
	req := RemoveIdentityRequest{
		ID:     name,
		CAName: caname,
		Force:  force,
	}
	var result IdentityResponse
	queryParam := make(map[string]string)
	queryParam["force"] = strconv.FormatBool(req.Force)
	if req.CAName != "" {
		queryParam["ca"] = req.CAName
	}
	err := id.delete(fmt.Sprintf("identities/%s", req.ID), result, queryParam)
	if err != nil {
		return err
	}
	return nil
}

// UPDATE http://localhost:7054/modify/identity/{id}
func (id *identity) updateIdentity(req ModifyIdentityRequest) error {
	reqBody, err := marshal(req, "modifyIdentity")
	if err != nil {
		return err
	}
	queryParam := make(map[string]string)
	queryParam["force"] = "true"
	result := IdentityResponse{}
	return id.put(fmt.Sprintf("identities/%s", req.ID), queryParam, reqBody, result)
}

// POST http://localhost:7054/affiliations
func (id *identity) addAffiliation(name, caname string, force bool) error {
	req := AddAffiliationRequest{
		Name:   name,
		CAName: caname,
		Force:  force,
	}
	reqBody, err := marshal(req, "addAffiliation")
	if err != nil {
		return fmt.Errorf("Marshal: %v", err)
	}
	queryParam := make(map[string]string)
	queryParam["force"] = strconv.FormatBool(req.Force)
	err = id.post("affiliations", reqBody, nil, queryParam)
	if err != nil {
		return err
	}
	return nil
}

// DELETE http://localhost:7054/affiliations/{$id}
func (id *identity) delAffiliation(req RemoveAffiliationRequest) error {
	result := &AffiliationResponse{}
	queryParam := make(map[string]string)
	queryParam["force"] = strconv.FormatBool(req.Force)
	if req.CAName != "" {
		queryParam["ca"] = req.CAName
	}
	err := id.delete(fmt.Sprintf("affiliations/%s", req.Name), result, queryParam)
	if err != nil {
		return err
	}
	return nil
}

// GET http://localhost:7054/affiliations
func (id *identity) getAffiliation(name, caname string) (*AffiliationResponse, error) {
	req := GetAffiliationRequest{
		Name:   name,
		CAName: caname,
	}
	var result AffiliationResponse
	err := id.get(fmt.Sprintf("affiliations/%s", req.Name), req.CAName, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}


