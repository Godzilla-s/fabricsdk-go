package caclient

import (
	"fmt"
	"os"
	"testing"
)

// before test: need set fabric-ca-server

const (
	TEST_FABRIC_CA_URL = "http://192.168.1.106:7054"
)

func TestClient_New(t *testing.T) {
	os.Setenv("TEST_FABRIC_CA_URL", TEST_FABRIC_CA_URL)
	url := os.Getenv("TEST_FABRIC_CA_URL")
	c := New(Config{
		Username: "admin",
		Password: "123456",
		URL: url,
	})

	key, err := c.Enroll(EnrollmentRequest{Name: "admin", Secret: "123456"})
	if err != nil {
		t.Fatal(err)
	}

	rawkey, err := key.GetKeyCert(nil)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(rawkey))
}

func TestClient_Register(t *testing.T) {
	os.Setenv("TEST_FABRIC_CA_URL", TEST_FABRIC_CA_URL)
	url := os.Getenv("TEST_FABRIC_CA_URL")
	c := New(Config{
		Username: "admin",
		Password: "123456",
		URL: url,
	})

	req := NewRegisterRequest("test1", "123456", ROLE_ADMIN)
	err := c.Register(req, nil)
	if err != nil {
		t.Fatal(err)
	}
}
