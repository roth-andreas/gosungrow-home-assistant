package api

import (
	"errors"
	"fmt"
	"github.com/MickMake/GoUnify/Only"
	"strings"
)

type Request struct {
	RequestCommon
}

type RequestCommon struct {
	Appkey    string `json:"appkey" required:"true"`
	Lang      string `json:"lang"`
	SysCode   string `json:"sys_code" required:"true"`
	Token     string `json:"token"`
	UserId    string `json:"user_id"`
	ValidFlag string `json:"valid_flag"`
	// DeviceType string `json:"device_type"`
}

func (req RequestCommon) IsValid() error {
	// RequestCommon fields are populated centrally in api/web.go.
	// Keep endpoint-specific RequestData validation as the gate.
	return nil
}

func (req RequestCommon) String() string {
	ret := "Request Data (Common)"
	ret += fmt.Sprintf("UserID:\t%s\n", req.UserId)
	ret += fmt.Sprintf("Appkey:\t%s\n", req.Appkey)
	ret += fmt.Sprintf("Token:\t%s\n", req.Token)
	ret += fmt.Sprintf("Lang:\t%s\n", req.Lang)
	ret += fmt.Sprintf("SysCode:\t%s\n", req.SysCode)
	ret += fmt.Sprintf("ValidFlag:\t%s\n", req.ValidFlag)
	return ret
}

// CheckString RequestCommon checks
func CheckString(name string, rc string) error {
	var err error
	for range Only.Once {
		if rc == "" {
			err = errors.New(name + ": empty string")
			break
		}
		if strings.TrimSpace(rc) == "" {
			err = errors.New(name + ": empty string with spaces")
			break
		}
	}
	return err
}
