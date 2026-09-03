package hostuser

import (
	"fmt"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/GVALFER/WEBYCP/internal/validate"
)

const WebGroup = "www-data"

type Identity struct {
	Home       string
	SystemUser string
	UID        int
	GID        int
}

func ValidateNames(accountID, systemUser string) error {
	if err := validate.ID("accountId", accountID); err != nil {
		return err
	}
	if err := validate.SystemUser(systemUser); err != nil {
		return err
	}
	if systemUser != "wcp_"+accountID[:12] {
		return &validate.Error{
			Field: "systemUser", Message: "System user does not match account ID",
		}
	}
	return nil
}

func Validate(found *user.User, homeRoot, accountID, systemUser string) (Identity, error) {
	if err := ValidateNames(accountID, systemUser); err != nil {
		return Identity{}, err
	}
	expectedHome := filepath.Join(homeRoot, systemUser)
	if found.Uid == "0" {
		return Identity{}, fmt.Errorf("system user resolves to root")
	}
	if found.HomeDir != expectedHome {
		return Identity{}, fmt.Errorf("system user home is %q, expected %q", found.HomeDir, expectedHome)
	}
	if found.Name != "WEBYCP:"+accountID {
		return Identity{}, fmt.Errorf("system user is not owned by this account")
	}
	uid, err := strconv.Atoi(found.Uid)
	if err != nil || uid <= 0 {
		return Identity{}, fmt.Errorf("invalid system user UID")
	}
	gid, err := strconv.Atoi(found.Gid)
	if err != nil || gid <= 0 {
		return Identity{}, fmt.Errorf("invalid system user GID")
	}
	return Identity{
		Home: expectedHome, SystemUser: systemUser, UID: uid, GID: gid,
	}, nil
}
