package caclient

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"
	"github.com/mitchellh/mapstructure"
	cfsslapi "github.com/cloudflare/cfssl/api"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net/http"
)

type Client struct {
	username string
	password string
	url      string
	httpCli  *http.Client
}

type Config struct {
	Username  string
	Password  string
	URL   string
}

func New(c Config) *Client {
	cli := &Client{
		username: c.Username,
		password: c.Password,
		url: c.URL,
	}
	return cli
}

func (c *Client) send(req *http.Request, response interface{}) error {
	if c.httpCli == nil {
		c.httpCli = &http.Client{}
	}
	resp, err := c.httpCli.Do(req)
	if err != nil {
		return fmt.Errorf("client request fail: %v", err)
	}
	defer resp.Body.Close()

	bodyData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var res *cfsslapi.Response
	if len(bodyData) > 0 {
		res = new(cfsslapi.Response)
		err = json.Unmarshal(bodyData, res)
		if err != nil {
			return err
		}
		if len(res.Errors) > 0 {
			var errorMsg string
			for _, err1 := range res.Errors {
				msg := fmt.Sprintf("Response from server: Error Code: %d - %s\n", err1.Code, err1.Message)
				if errorMsg == "" {
					errorMsg = msg
				} else {
					errorMsg = errorMsg + fmt.Sprintf("\n%s", msg)
				}
			}
			return errors.Errorf(errorMsg)
		}
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("invalid request code: %d", resp.StatusCode)
	}

	if res == nil {
		return fmt.Errorf("nil response result")
	}
	if response != nil {
		return mapstructure.Decode(res.Result, response)
	}
	return nil
}

func (c *Client) newRequest(method, uri string, body io.Reader) (*http.Request, error) {
	urlStr := fmt.Sprintf("%s/%s", c.url, uri)
	return http.NewRequest(method, urlStr, body)
}

func (c *Client) newIdentity(key KeyStore) *identity {
	return &identity{
		cert: key.GetSignCert(),
		key: key.GetKey(),
		client: c,
	}
}

//Issue 下发证书
func (c *Client) Issue(req EnrollmentRequest) (KeyStore, error) {
	csr := &cryptoutil.CSRInfo{
		CN: req.CN,
		Hosts: req.Hosts,
	}
	if req.CN == "" {
		csr.CN = req.Name
	}
	csrBytes, key, err := cryptoutil.GenerateKey(csr, req.Name)
	if err != nil {
		return nil, err
	}

	reqNet := EnrollmentRequestNet{CAName: req.CAName, AttrReqs:req.AttrReq}
	reqNet.SignRequest.Request = string(csrBytes)
	if req.Profile != "" {
		reqNet.Profile = req.Profile
	}
	reqNet.Label = req.Label
	body, err := marshal(&reqNet, "SignRequest")
	if err != nil {
		return nil, err
	}

	request, err := c.newRequest("POST", "enroll", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.SetBasicAuth(req.Name, req.Secret)
	var respNet EnrollmentResponseNet
	err = c.send(request, &respNet)
	if err != nil {
		return nil, err
	}

	signCert, _ := base64.StdEncoding.DecodeString(respNet.Cert)
	rootCert, _ := base64.StdEncoding.DecodeString(respNet.ServerInfo.CAChain)
	return keystore{signCert: signCert, rootCert: rootCert, key: key}, nil
}

func (c *Client) Register(req RegistrationRequest, authorized KeyStore) error {
	var id *identity
	if authorized != nil {
		id = c.newIdentity(authorized)
	} else {
		if c.username == "" || c.password == "" {
			return fmt.Errorf("name or secret of login CA server is missing")
		}
		cakey, err := c.Issue(EnrollmentRequest{Name: c.username, Secret: c.password})
		if err != nil {
			return err
		}
		id = c.newIdentity(cakey)
	}
	if req.Affiliation != "" {
		_, err := id.getAffiliation(req.Affiliation, req.CAName)
		if err != nil {
			// 如果不存在，新增affiliation
			err = id.addAffiliation(req.Affiliation, req.CAName, true)
			if err != nil {
				return fmt.Errorf("fail to add affilications: %v", err)
			}
		}
	}
	_, err := id.getIdentity(req.Name, req.CAName)
	if err == nil {
		return errHasRegistered{role: req.Name, roleType: req.Type}
	}
	_, err = id.addIdentity(req)
	return err
}

func (c *Client) RegisterAndIssue(req RegistrationRequest, authorized KeyStore) {

}