package caclient

import (
	"os"
	"testing"
)

// before test: need set fabric-ca-server

func TestClient_New(t *testing.T) {
	url := os.Getenv("TEST_FABRIC_CA_URL")
	c := New(Config{
		Username: "admin",
		Password: "adminpw",
		URL: url,
	})

	if err := c.CheckConnect(); err != nil {
		t.Fatal(err)
	}
}

func TestClient_Register(t *testing.T) {
	url := os.Getenv("TEST_FABRIC_CA_URL")
	c := New(Config{
		Username: "admin",
		Password: "adminpw",
		URL: url,
	})

	req := NewRegisterRequest("test1", "123456", ROLE_ID_ADMIN)
	err := c.Register(req, nil)
	if err != nil {
		t.Fatal(err)
	}
}
