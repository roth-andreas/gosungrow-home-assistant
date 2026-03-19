package iSolarCloud

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/MickMake/GoUnify/Only"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/getDeviceList"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/getPsDetail"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/getPsList"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/getUserList"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/login"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/queryDeviceList"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/queryDeviceRealTimeDataByPsKeys"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/NullArea/NullEndpoint"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/WebAppService/getDevicePointAttrs"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/WebIscmAppService/getPsTreeMenu"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct/output"
)

const (
	DefaultCacheTimeout = time.Minute * 5
)

type SunGrow struct {
	ApiRoot     api.Web
	Auth        login.EndPoint
	Areas       api.Areas
	Error       error
	NeedLogin   bool
	AuthDetails *login.SunGrowAuth

	Directory  string
	OutputType output.OutputType
	SaveAsFile bool
}

func NewSunGro(baseURL string, cacheDir string) *SunGrow {
	var p SunGrow

	for range Only.Once {
		p.Error = p.ApiRoot.SetUrl(baseURL)
		if p.Error != nil {
			break
		}

		p.Error = p.ApiRoot.SetCacheDir(cacheDir)
		if p.Error != nil {
			break
		}
	}
	return &p
}

func (sg *SunGrow) Init() error {
	for range Only.Once {
		nullArea := api.GetArea(NullEndpoint.EndPoint{})
		appArea := api.GetArea(login.EndPoint{})
		webAppArea := api.GetArea(getDevicePointAttrs.EndPoint{})
		webIscmArea := api.GetArea(getPsTreeMenu.EndPoint{})

		sg.Areas = api.Areas{
			nullArea: {
				ApiRoot: sg.ApiRoot,
				Name:    nullArea,
				EndPoints: api.TypeEndPoints{
					api.GetName(NullEndpoint.EndPoint{}): NullEndpoint.Init(sg.ApiRoot),
				},
			},
			appArea: {
				ApiRoot: sg.ApiRoot,
				Name:    appArea,
				EndPoints: api.TypeEndPoints{
					api.GetName(login.EndPoint{}):                           login.Init(sg.ApiRoot),
					api.GetName(getUserList.EndPoint{}):                     getUserList.Init(sg.ApiRoot),
					api.GetName(getPsList.EndPoint{}):                       getPsList.Init(sg.ApiRoot),
					api.GetName(getPsDetail.EndPoint{}):                     getPsDetail.Init(sg.ApiRoot),
					api.GetName(getDeviceList.EndPoint{}):                   getDeviceList.Init(sg.ApiRoot),
					api.GetName(queryDeviceList.EndPoint{}):                 queryDeviceList.Init(sg.ApiRoot),
					api.GetName(queryDeviceRealTimeDataByPsKeys.EndPoint{}): queryDeviceRealTimeDataByPsKeys.Init(sg.ApiRoot),
				},
			},
			webAppArea: {
				ApiRoot: sg.ApiRoot,
				Name:    webAppArea,
				EndPoints: api.TypeEndPoints{
					api.GetName(getDevicePointAttrs.EndPoint{}): getDevicePointAttrs.Init(sg.ApiRoot),
				},
			},
			webIscmArea: {
				ApiRoot: sg.ApiRoot,
				Name:    webIscmArea,
				EndPoints: api.TypeEndPoints{
					api.GetName(getPsTreeMenu.EndPoint{}): getPsTreeMenu.Init(sg.ApiRoot),
				},
			},
		}
	}

	return sg.Error
}

func (sg *SunGrow) IsError() bool {
	return sg.Error != nil
}

func (sg *SunGrow) IsNotError() bool {
	return sg.Error == nil
}

func (sg *SunGrow) AppendUrl(endpoint string) api.EndPointUrl {
	return sg.ApiRoot.AppendUrl(endpoint)
}

func (sg *SunGrow) GetEndpoint(ae string) api.EndPoint {
	var ep api.EndPoint

	for range Only.Once {
		area, endpoint := sg.SplitEndPoint(ae)
		if sg.IsError() {
			break
		}

		ep = sg.Areas.GetEndPoint(area, endpoint)
		if ep.IsError() {
			sg.Error = ep.GetError()
			break
		}

		if sg.Auth.Token() != "" {
			appKey := sg.GetAppKey()
			if appKey == "" {
				appKey = DefaultApiAppKey
			}
			ep = ep.SetRequest(api.RequestCommon{
				Appkey:    appKey,
				Lang:      "_en_US",
				SysCode:   "200",
				Token:     sg.GetToken(),
				UserId:    sg.GetUserId(),
				ValidFlag: "1,3",
			})
		}
	}

	return ep
}

func (sg *SunGrow) GetByJson(endpoint string, request string) api.EndPoint {
	var ret api.EndPoint
	for range Only.Once {
		if sg.NeedLogin {
			sg.Error = errors.New("currently logged out")
			break
		}

		ret = sg.GetEndpoint(endpoint)
		if sg.IsError() {
			break
		}

		if request != "" {
			ret = ret.SetRequestByJson(output.Json(request))
			sg.Error = ret.GetError()
			if sg.IsError() {
				fmt.Println(ret.Help())
				break
			}
		}

		ret = ret.Call()
		sg.Error = ret.GetError()
		if sg.IsLoggedOut() {
			break
		}

		switch {
		case sg.OutputType.IsNone():
			if sg.IsError() {
				fmt.Println(ret.Help())
				break
			}
		case sg.OutputType.IsRaw():
			if sg.SaveAsFile {
				sg.Error = ret.WriteDataFile()
				break
			}
			fmt.Println(ret.GetJsonData(true))
		case sg.OutputType.IsJson():
			if sg.IsError() {
				fmt.Println(ret.Help())
				break
			}
			if sg.SaveAsFile {
				sg.Error = ret.WriteDataFile()
				break
			}
			fmt.Println(ret.GetJsonData(false))
		default:
			if sg.IsError() {
				fmt.Println(ret.Help())
				break
			}
		}
	}
	return ret
}

func (sg *SunGrow) GetByStruct(endpoint string, request interface{}, cache time.Duration) api.EndPoint {
	var ret api.EndPoint
	for range Only.Once {
		if sg.NeedLogin {
			sg.Error = errors.New("currently logged out")
			break
		}

		ret = sg.GetEndpoint(endpoint)
		if sg.IsError() {
			break
		}
		if ret.IsError() {
			sg.Error = ret.GetError()
			break
		}

		if request != nil {
			ret = ret.SetRequest(request)
			if ret.IsError() {
				sg.Error = ret.GetError()
				break
			}
		}

		ret = ret.SetCacheTimeout(cache)
		ret = ret.Call()
		if !ret.IsError() {
			break
		}

		sg.Error = ret.GetError()
		if sg.IsLoggedOut() {
			break
		}
	}

	return ret
}

func (sg *SunGrow) RequestRequiresArgs(ae string) bool {
	var yes bool
	for range Only.Once {
		area, endpoint := sg.SplitEndPoint(ae)
		if sg.IsError() {
			break
		}

		yes = sg.Areas.RequestRequiresArgs(area, endpoint)
	}

	return yes
}

func (sg *SunGrow) RequestArgs(ae string) map[string]string {
	var ret map[string]string
	for range Only.Once {
		area, endpoint := sg.SplitEndPoint(ae)
		if sg.IsError() {
			break
		}

		ret = sg.Areas.RequestArgs(area, endpoint)
	}
	return ret
}

func (sg *SunGrow) SplitEndPoint(ae string) (api.AreaName, api.EndPointName) {
	var area api.AreaName
	var endpoint api.EndPointName

	for range Only.Once {
		s := strings.Split(ae, ".")
		switch len(s) {
		case 0:
			sg.Error = errors.New("empty endpoint")
		case 1:
			area = api.GetArea(login.EndPoint{})
			endpoint = api.EndPointName(s[0])
		case 2:
			area = api.AreaName(s[0])
			endpoint = api.EndPointName(s[1])
		default:
			sg.Error = errors.New("too many delimiters defined, (only one '.' allowed)")
		}
	}

	return area, endpoint
}

func (sg *SunGrow) AreaExists(area string) bool {
	return sg.Areas.Exists(area)
}

func (sg *SunGrow) AreaNotExists(area string) bool {
	return sg.Areas.NotExists(area)
}

func (sg *SunGrow) login() error {
	for range Only.Once {
		if sg.AuthDetails == nil {
			break
		}
		a := sg.GetEndpoint(login.EndPointName)
		sg.Auth = login.Assert(a)

		sg.Error = sg.Auth.Login(sg.AuthDetails)
		if sg.IsLoggedOut() {
			break
		}
		if sg.IsError() {
			break
		}
	}
	return sg.Error
}

func (sg *SunGrow) Login(auth login.SunGrowAuth) error {
	for range Only.Once {
		sg.AuthDetails = &auth

		for range Only.Twice {
			sg.Error = nil
			sg.Error = sg.login()
			if sg.Error != nil {
				break
			}

			_ = sg.GetByStruct(getUserList.EndPointName, nil, DefaultCacheTimeout)
			if !sg.IsLoggedOut() {
				break
			}

			if sg.Error == nil {
				sg.NeedLogin = false
				break
			}

			_, _ = fmt.Fprintf(os.Stderr, "Logging in again\n")
		}

		if sg.NeedLogin {
			if sg.Error == nil {
				sg.Error = errors.New("need to login again")
			}
			break
		}
		if sg.Error != nil {
			break
		}
	}

	return sg.Error
}

func (sg *SunGrow) IsLoggedOut() bool {
	for range Only.Once {
		if sg.IsNotError() {
			sg.NeedLogin = false
			break
		}
		if strings.Contains(sg.Error.Error(), "er_token_login_invalid") {
			sg.NeedLogin = true
			sg.Logout()
		}
	}
	return sg.NeedLogin
}

func (sg *SunGrow) Logout() {
	for range Only.Once {
		_ = sg.ApiRoot.WebCacheRemove(sg.Auth)
		_ = sg.Auth.RemoveToken()
	}
}

func (sg *SunGrow) GetToken() string {
	return sg.Auth.Token()
}

func (sg *SunGrow) GetUserId() string {
	return sg.Auth.UserId()
}

func (sg *SunGrow) GetAppKey() string {
	return sg.Auth.AppKey()
}

func (sg *SunGrow) GetLastLogin() string {
	return sg.Auth.LastLogin().Format(login.DateTimeFormat)
}

func (sg *SunGrow) GetUserName() string {
	return sg.Auth.UserName()
}

func (sg *SunGrow) GetUserEmail() string {
	return sg.Auth.Email()
}

func (sg *SunGrow) HasTokenChanged() bool {
	return sg.Auth.HasTokenChanged()
}

func (sg *SunGrow) SetOutputType(outputType string) {
	sg.OutputType.Set(outputType)
}
