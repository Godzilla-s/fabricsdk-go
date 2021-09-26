package caclient

import "fmt"

type errHasRegistered struct {
	role  string
	roleType string
}

func (err errHasRegistered) Error() string {
	return fmt.Sprintf("role %s with type %s has registered in CA server", err.role, err.roleType)
}

func ErrIsRegistered(err error) bool {
	_, ok := err.(errHasRegistered)
	return ok
}

