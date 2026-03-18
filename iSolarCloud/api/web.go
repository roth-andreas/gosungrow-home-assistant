package api

import (
	"github.com/MickMake/GoSungrow/iSolarCloud/api/GoStruct/output"
	"github.com/MickMake/GoUnify/Only"
	"github.com/MickMake/GoUnify/cmdPath"
	"crypto/rand"
	"io"
	"os"
	"path/filepath"
	"time"

	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)


type Web struct {
	ServerUrl EndPointUrl
	Body      []byte
	Error     error

	cacheDir     string
	cacheTimeout time.Duration
	retry        int
	timeOffset   time.Duration
	client       http.Client
	httpRequest  *http.Request
	httpResponse *http.Response
}

const defaultWebClientVersion = "2026011301"

func randomDigits(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("invalid random digit length")
	}
	const digits = "0123456789"
	raw := make([]byte, length)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.Grow(length)
	for i := 0; i < length; i++ {
		sb.WriteByte(digits[int(raw[i])%len(digits)])
	}
	return sb.String(), nil
}

func currentGMTHeader() string {
	_, offsetSeconds := time.Now().Zone()
	sign := "%2B"
	hours := offsetSeconds / 3600
	if hours < 0 {
		sign = "-"
		hours = -hours
	}
	return fmt.Sprintf("GMT%s%d", sign, hours)
}

func (w *Web) requestTimestampMillis() int64 {
	now := time.Now().Add(w.timeOffset)
	if offset := strings.TrimSpace(os.Getenv("GOSUNGROW_TIMESTAMP_OFFSET_MS")); offset != "" {
		if ms, err := strconv.ParseInt(offset, 10, 64); err == nil {
			now = now.Add(time.Duration(ms) * time.Millisecond)
		}
	}
	return now.UnixMilli()
}

func (w *Web) refreshTimeOffsetFromResponse() bool {
	if w.httpResponse == nil {
		return false
	}
	dateHeader := strings.TrimSpace(w.httpResponse.Header.Get("Date"))
	if dateHeader == "" {
		return false
	}

	serverNow, err := http.ParseTime(dateHeader)
	if err != nil {
		return false
	}

	w.timeOffset = serverNow.Sub(time.Now().UTC())
	return true
}

func (w *Web) isExpiredRequestResponse(raw []byte, decrypted []byte) bool {
	check := strings.ToLower(string(raw))
	if strings.Contains(check, "expired request") {
		return true
	}
	check = strings.ToLower(string(decrypted))
	return strings.Contains(check, "expired request")
}


func (w *Web) SetUrl(u string) error {
	w.ServerUrl = SetUrl(u)
	return w.Error
}

func (w *Web) AppendUrl(endpoint string) EndPointUrl {
	return w.ServerUrl.AppendPath(endpoint)
}

func (w *Web) Get(endpoint EndPoint) EndPoint {
	for range Only.Once {
		w.Error = w.ServerUrl.IsValid()
		if w.Error != nil {
			w.Error = errors.New("Sungrow API EndPoint not yet implemented")
			fmt.Println(w.Error)
			break
		}

		isCached := false
		if w.WebCacheCheck(endpoint) {
			isCached = true
		}


		if isCached {
			w.Body, w.Error = w.WebCacheRead(endpoint)
			if w.Error != nil {
				break
			}

		} else {
			w.Body, w.Error = w.getApi(endpoint)
			if w.Error != nil {
				break
			}
		}


		if len(w.Body) == 0 {
			w.Error = errors.New("empty http response")
			break
		}
		endpoint = endpoint.SetResponse(w.Body)
		if endpoint.GetError() != nil {
			w.Error = endpoint.GetError()
			break
		}

		w.Error = endpoint.IsResponseValid()
		if w.Error != nil {
			_ = w.WebCacheRemove(endpoint)
			// fmt.Printf("ERROR: Body is:\n%s\n", w.Body)
			break
		}

		if isCached {
			// Do nothing.
		} else {
			w.Error = w.WebCacheWrite(endpoint, w.Body)
			if w.Error != nil {
				break
			}
		}
	}

	if w.Error != nil {
		endpoint = endpoint.SetError("%s", w.Error)
	}
	return endpoint
}

func (w *Web) getApi(endpoint EndPoint) ([]byte, error) {
	for range Only.Once {
		request := endpoint.RequestRef()

		w.Error = endpoint.IsRequestValid()
		if w.Error != nil {
			break
		}

		u := endpoint.GetUrl()
		w.Error = u.IsValid()
		if w.Error != nil {
			break
		}

		postUrl := w.ServerUrl.AppendPath(u.String()).String()
		var requestJson []byte
		requestJson, w.Error = json.Marshal(request)
		if w.Error != nil {
			break
		}

		var reqData map[string]interface{}
		w.Error = json.Unmarshal(requestJson, &reqData)
		if w.Error != nil {
			break
		}

		// queryDeviceRealTimeDataByPsKeys currently fails against the legacy endpoint
		// contract for many accounts. Route it to queryDeviceList by deriving ps_id
		// from ps_key_list, while keeping the public command shape stable.
		if endpoint.GetName().String() == "queryDeviceRealTimeDataByPsKeys" {
			rawPsKey := strings.TrimSpace(fmt.Sprintf("%v", reqData["ps_key_list"]))
			if rawPsKey != "" && rawPsKey != "<nil>" {
				psIdPart := strings.Split(rawPsKey, "_")[0]
				if psIdPart != "" {
					if parsed, perr := strconv.ParseInt(psIdPart, 10, 64); perr == nil {
						reqData["ps_id"] = parsed
					} else {
						reqData["ps_id"] = psIdPart
					}
				}
			}
			delete(reqData, "ps_key_list")
		}

		tokenPlain := ""
		// Match frontend behavior: omit empty optional auth fields from JSON body.
		if tokenAny, ok := reqData["token"]; ok {
			tokenPlain = strings.TrimSpace(fmt.Sprintf("%v", tokenAny))
			if tokenPlain == "" || tokenPlain == "<nil>" {
				delete(reqData, "token")
				tokenPlain = ""
			}
		}
		if userAny, ok := reqData["user_id"]; ok {
			user := strings.TrimSpace(fmt.Sprintf("%v", userAny))
			if user == "" || user == "<nil>" {
				delete(reqData, "user_id")
			}
		}
		if validAny, ok := reqData["valid_flag"]; ok {
			valid := strings.TrimSpace(fmt.Sprintf("%v", validAny))
			if valid == "" || valid == "<nil>" {
				delete(reqData, "valid_flag")
			}
		}
		lang := strings.TrimSpace(fmt.Sprintf("%v", reqData["lang"]))
		if lang == "" || lang == "<nil>" {
			reqData["lang"] = "_en_US"
		}

		appKey := strings.TrimSpace(fmt.Sprintf("%v", reqData["appkey"]))
		if appKey == "" || appKey == "<nil>" {
			appKey = defaultApiAppKey
		}
		reqData["appkey"] = appKey
		reqData["sys_code"] = 200

		nonce, err := randomWord(32)
		if err != nil {
			w.Error = err
			break
		}
		reqData["api_key_param"] = map[string]interface{}{
			"timestamp": w.requestTimestampMillis(),
			"nonce":     nonce,
		}

		suffix, err := randomWord(29)
		if err != nil {
			w.Error = err
			break
		}
		randomKey := "web" + suffix

		compactBody, err := json.Marshal(reqData)
		if err != nil {
			w.Error = err
			break
		}

		encryptedBody, err := encryptHex(string(compactBody), randomKey)
		if err != nil {
			w.Error = err
			break
		}

		httpReq, err := http.NewRequest("POST", postUrl, bytes.NewBufferString(encryptedBody))
		if err != nil {
			w.Error = err
			break
		}
		did := strings.TrimSpace(os.Getenv("GOSUNGROW_DID"))
		if did == "" {
			did, _ = randomDigits(16)
		}
		httpReq.Header.Set("accept", "application/json, text/plain, */*")
		httpReq.Header.Set("accept-language", "en-US,en;q=0.9")
		httpReq.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36")
		httpReq.Header.Set("origin", "https://www.isolarcloud.com")
		httpReq.Header.Set("referer", "https://www.isolarcloud.com/")
		httpReq.Header.Set("content-type", "text/plain;charset=UTF-8")
		httpReq.Header.Set("sys_code", "200")
		httpReq.Header.Set("_pl", "js")
		httpReq.Header.Set("_did", did)
		httpReq.Header.Set("_global_new_web", "1")
		httpReq.Header.Set("_vc", defaultWebClientVersion)
		httpReq.Header.Set("_browser_brand", "chrome")
		httpReq.Header.Set("_browser_version", "132.0")
		httpReq.Header.Set("x-client-tz", currentGMTHeader())
		httpReq.Header.Set("x-sign-code", "0")
		httpReq.Header.Set("x-access-key", defaultAccessKey)

		encryptedKey, err := rsaEncryptBase64(randomKey)
		if err != nil {
			w.Error = err
			break
		}
		httpReq.Header.Set("x-random-secret-key", encryptedKey)

		limitObjPlain := ""
		limitObjSet := false
		if tokenPlain != "" {
			parts := strings.Split(tokenPlain, "_")
			if len(parts) > 0 {
				limitObjPlain = strings.TrimSpace(parts[0])
				if limitObjPlain != "" {
					limitObjSet = true
				}
			}
		}
		if envLimit, ok := os.LookupEnv("GOSUNGROW_LIMIT_OBJ"); ok {
			if envLimit == "__EMPTY__" {
				envLimit = ""
			}
			limitObjPlain = envLimit
			limitObjSet = true
		}
		if limitObjSet {
			limitObjEncrypted, encErr := rsaEncryptBase64(limitObjPlain)
			if encErr == nil {
				httpReq.Header.Set("x-limit-obj", limitObjEncrypted)
			}
		}

		w.httpResponse, w.Error = http.DefaultClient.Do(httpReq)
		if w.Error != nil {
			break
		}

		if w.httpResponse.StatusCode == 401 {
			w.Error = errors.New(w.httpResponse.Status)
			break
		}

		//goland:noinspection GoUnhandledErrorResult,GoDeferInLoop
		defer w.httpResponse.Body.Close()
		if w.Error != nil {
			break
		}

		if w.httpResponse.StatusCode != 200 {
			w.Error = errors.New(fmt.Sprintf("API httpResponse is %s", w.httpResponse.Status))
			break
		}

		w.Body, w.Error = io.ReadAll(w.httpResponse.Body)
		if w.Error != nil {
			break
		}

		rawBody := append([]byte(nil), w.Body...)
		decrypted, decErr := decryptHex(string(w.Body), randomKey)
		if decErr == nil && json.Valid(decrypted) {
			w.Body = decrypted
		}
		if os.Getenv("GOSUNGROW_TRACE_HTTP") != "" {
			requestDump, _ := json.Marshal(reqData)
			_, _ = fmt.Fprintf(os.Stderr, "[TRACE] POST %s\n", postUrl)
			_, _ = fmt.Fprintf(os.Stderr, "[TRACE] req=%s\n", string(requestDump))
			_, _ = fmt.Fprintf(os.Stderr, "[TRACE] respRaw=%s\n", string(rawBody))
			if decErr != nil {
				_, _ = fmt.Fprintf(os.Stderr, "[TRACE] decryptErr=%v\n", decErr)
			} else {
				_, _ = fmt.Fprintf(os.Stderr, "[TRACE] respDec=%s\n", string(decrypted))
			}
		}

		if w.isExpiredRequestResponse(rawBody, w.Body) {
			shouldRetry := w.retry == 0
			if shouldRetry {
				w.retry++
				_ = w.refreshTimeOffsetFromResponse()
				return w.getApi(endpoint)
			}
			w.retry = 0
		} else {
			w.retry = 0
		}
	}

	return w.Body, w.Error
}

func (w *Web) SetCacheDir(basedir string) error {
	for range Only.Once {
		w.cacheDir = filepath.Join(basedir)

		p := cmdPath.NewPath(basedir)
		if p.DirExists() {
			break
		}

		w.Error = p.MkdirAll()
		if w.Error != nil {
			break
		}

		// _, w.Error = os.Stat(w.cacheDir)
		// if w.Error != nil {
		// 	if os.IsNotExist(w.Error) {
		// 		w.Error = nil
		// 	}
		// 	break
		// }
		//
		// w.Error = os.MkdirAll(w.cacheDir, 0700)
		// if w.Error != nil {
		// 	break
		// }
	}

	return w.Error
}

func (w *Web) GetCacheDir() string {
	return w.cacheDir
}

func (w *Web) SetCacheTimeout(duration time.Duration) {
	w.cacheTimeout = duration
}

func (w *Web) GetCacheTimeout() time.Duration {
	return w.cacheTimeout
}

// WebCacheCheck Retrieves cache data from a local file.
func (w *Web) WebCacheCheck(endpoint EndPoint) bool {
	var ok bool
	for range Only.Once {
		// fn := filepath.Join(w.cacheDir, endpoint.CacheFilename())
		//
		// var f os.FileInfo
		// f, w.Error = os.Stat(fn)
		// if w.Error != nil {
		// 	if os.IsNotExist(w.Error) {
		// 		w.Error = nil
		// 	}
		// 	break
		// }
		//
		// if f.IsDir() {
		// 	w.Error = errors.New("file is a directory")
		// 	break
		// }

		p := cmdPath.NewPath(w.cacheDir, endpoint.CacheFilename())
		if p.DirExists() {
			w.Error = errors.New("file is a directory")
			ok = false
			break
		}
		if !p.FileExists() {
			ok = false
			break
		}

		duration := w.GetCacheTimeout()
		then := p.ModTime()
		then = then.Add(duration)
		now := time.Now()
		if then.Before(now) {
			ok = false
			break
		}

		ok = true
	}

	return ok
}

// WebCacheRead Retrieves cache data from a local file.
func (w *Web) WebCacheRead(endpoint EndPoint) ([]byte, error) {
	fn := filepath.Join(w.cacheDir, endpoint.CacheFilename())
	return output.PlainFileRead(fn)
}

// WebCacheRemove Removes a cache file.
func (w *Web) WebCacheRemove(endpoint EndPoint) error {
	fn := filepath.Join(w.cacheDir, endpoint.CacheFilename())
	return output.FileRemove(fn)
}

// WebCacheWrite Saves cache data to a file path.
func (w *Web) WebCacheWrite(endpoint EndPoint, data []byte) error {
	fn := filepath.Join(w.cacheDir, endpoint.CacheFilename())
	return output.PlainFileWrite(fn, data, output.DefaultFileMode)
}


// PointCacheCheck Retrieves cache data from a local file.
func (w *Web) PointCacheCheck(data DataMap) bool {
	var ok bool
	for range Only.Once {
		p := cmdPath.NewPath(w.cacheDir, "Points.json")
		if p.DirExists() {
			w.Error = errors.New("file is a directory")
			ok = false
			break
		}
		if p.FileExists() {
			ok = true
			break
		}

		duration := w.GetCacheTimeout()
		then := p.ModTime()
		then = then.Add(duration)
		now := time.Now()
		if then.Before(now) {
			break
		}

		ok = true
	}

	return ok
}

// PointCacheRead Retrieves cache data from a local file.
func (w *Web) PointCacheRead(endpoint EndPoint) ([]byte, error) {
	fn := filepath.Join(w.cacheDir, endpoint.CacheFilename())
	return output.PlainFileRead(fn)
}

// PointCacheWrite Saves cache data to a file path.
func (w *Web) PointCacheWrite(endpoint EndPoint, data []byte) error {
	fn := filepath.Join(w.cacheDir, endpoint.CacheFilename())
	return output.PlainFileWrite(fn, data, output.DefaultFileMode)
}
