package panlogin

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/tickstep/aliyunpan/internal/global"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/library-go/crypto"
	"github.com/tickstep/library-go/getip"
	"github.com/tickstep/library-go/logger"
	"github.com/tickstep/library-go/requester"
	"net/url"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type LoginHelper struct {
	webHost string
}

type LoginHttpResult struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// QRCodeUrlResult 二维码登录页面
type QRCodeUrlResult struct {
	TokenId     string `json:"tokenId"`
	TokenUrl    string `json:"tokenUrl"`
	ExpiredTime int    `json:"expiredTime"`
}

// QRCodeLoginResult 二维码登录结果
type QRCodeLoginResult struct {
	QrCodeStatus       string `json:"qrCodeStatus"`
	SecureRefreshToken string `json:"secureRefreshToken"`
}

// LoginTokenResult Token结果
type LoginTokenResult struct {
	AccessToken string `json:"accessToken"`
	Expired     int64  `json:"expired"`
}

type CommonTokenEntity struct {
	Openapi *LoginTokenResult `json:"openapi"`
	Webapi  *LoginTokenResult `json:"webapi"`
}

func NewLoginHelper(webHost string) *LoginHelper {
	return &LoginHelper{
		webHost: webHost,
	}
}

// GetQRCodeLoginUrl 获取登录二维码链接
func (h *LoginHelper) GetQRCodeLoginUrl(keyStr string) (*QRCodeUrlResult, error) {
	if keyStr == "" {
		keyStr = utils.GetUniqueKeyStr()
	}
	fullUrl := strings.Builder{}
	ipAddr, err := getip.IPInfoFromTechainBaidu()
	if err != nil {
		ipAddr = "127.0.0.1"
	}
	fmt.Fprintf(&fullUrl, "%s/auth/tickstep/aliyunpan/token/qrcode/create?ip=%s&os=%s&arch=%s&version=%s&key=%s",
		h.webHost, ipAddr, runtime.GOOS, runtime.GOARCH, global.AppVersion, keyStr)

	logger.Verboseln("do request url: " + fullUrl.String())
	header := map[string]string{
		"accept":       "application/json, text/plain, */*",
		"content-type": "application/json;charset=UTF-8",
		"user-agent":   "aliyunpan/" + global.AppVersion,
	}
	// request
	client := requester.NewHTTPClient()
	client.SetTimeout(20 * time.Second)
	client.SetKeepAlive(false)
	body, err := client.Fetch("GET", fullUrl.String(), nil, header)
	if err != nil {
		logger.Verboseln("get qr code error ", err)
		return nil, err
	}

	errResp := &LoginHttpResult{}
	if err1 := json.Unmarshal(body, errResp); err1 != nil {
		logger.Verboseln("parse qrcode result json error ", err1)
		return nil, err1
	}
	if errResp.Code != 0 {
		return nil, fmt.Errorf(errResp.Msg)
	}

	// parse result
	r := &LoginHttpResult{}
	r.Data = &QRCodeUrlResult{}
	if err2 := json.Unmarshal(body, r); err2 != nil {
		logger.Verboseln("parse qrcode result json error ", err2)
		return nil, err2
	}

	token := r.Data.(*QRCodeUrlResult)
	if len(token.TokenUrl) > 0 {
		u, err := url.Parse(token.TokenUrl)
		if err != nil {
			return nil, err
		}
		m, _ := url.ParseQuery(u.RawQuery)
		token.TokenId = m["tokenId"][0]
	}
	return token, nil
}

// GetQRCodeLoginResult 获取二维码登录结果
func (h *LoginHelper) GetQRCodeLoginResult(tokenId string) (*QRCodeLoginResult, error) {
	fullUrl := strings.Builder{}
	fmt.Fprintf(&fullUrl, "%s/auth/tickstep/aliyunpan/token/qrcode/result?tokenId=%s",
		h.webHost, tokenId)

	logger.Verboseln("do request url: " + fullUrl.String())
	header := map[string]string{
		"accept":       "application/json, text/plain, */*",
		"content-type": "application/json;charset=UTF-8",
		"user-agent":   "aliyunpan/" + global.AppVersion,
	}
	// request
	client := requester.NewHTTPClient()
	client.SetTimeout(20 * time.Second)
	client.SetKeepAlive(false)
	body, err := client.Fetch("GET", fullUrl.String(), nil, header)
	if err != nil {
		logger.Verboseln("get qr code result error ", err)
		return nil, err
	}

	errResp := &LoginHttpResult{}
	if err1 := json.Unmarshal(body, errResp); err1 != nil {
		logger.Verboseln("parse qrcode result json error ", err1)
		return nil, err1
	}
	if errResp.Code != 0 {
		return nil, fmt.Errorf(errResp.Msg)
	}

	// parse result
	r := &LoginHttpResult{}
	r.Data = &QRCodeLoginResult{}
	if err2 := json.Unmarshal(body, r); err2 != nil {
		logger.Verboseln("parse qrcode result json error ", err2)
		return nil, err2
	}
	return r.Data.(*QRCodeLoginResult), nil
}

// GetRefreshToken 获取Token
func (h *LoginHelper) GetRefreshToken(tokenId string) (*QRCodeLoginResult, error) {
	fullUrl := strings.Builder{}
	fmt.Fprintf(&fullUrl, "%s/auth/tickstep/aliyunpan/token/qrcode/retrieve?tokenId=%s",
		h.webHost, tokenId)
	logger.Verboseln("do request url: " + fullUrl.String())
	header := map[string]string{
		"accept":       "application/json, text/plain, */*",
		"content-type": "application/json;charset=UTF-8",
		"user-agent":   "aliyunpan/" + global.AppVersion,
	}
	// request
	client := requester.NewHTTPClient()
	client.SetTimeout(20 * time.Second)
	client.SetKeepAlive(false)
	body, err := client.Fetch("GET", fullUrl.String(), nil, header)
	if err != nil {
		logger.Verboseln("get refresh token result error ", err)
		return nil, err
	}

	errResp := &LoginHttpResult{}
	if err1 := json.Unmarshal(body, errResp); err1 != nil {
		logger.Verboseln("parse qrcode result json error ", err1)
		return nil, err1
	}
	if errResp.Code != 0 {
		return nil, fmt.Errorf(errResp.Msg)
	}

	// parse result
	r := &LoginHttpResult{}
	r.Data = &QRCodeLoginResult{}
	if err2 := json.Unmarshal(body, r); err2 != nil {
		logger.Verboseln("parse refresh token result json error ", err2)
		return nil, err2
	}
	return r.Data.(*QRCodeLoginResult), nil
}

// ParseSecureRefreshToken 解密Token
func (h *LoginHelper) ParseSecureRefreshToken(keyStr, secureRefreshToken string) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			logger.Verboseln("decrypt string failed, maybe the key has been changed")
		}
	}()

	if len(keyStr) == 0 {
		keyStr = utils.GetUniqueKeyStr()
	}

	if secureRefreshToken == "" {
		return "", nil
	}
	d, _ := hex.DecodeString(secureRefreshToken)

	// use the machine unique id as the key
	// but in some OS, this key will be changed if you reinstall the OS
	key := []byte(keyStr)
	if len(key) > 16 {
		key = key[:16]
	}
	r, e := crypto.DecryptAES(d, key)
	if e != nil {
		return "", nil
	}

	refreshToken := string(r)
	matched, _ := regexp.MatchString(`^[\-a-zA-Z0-9]+`, refreshToken)
	if !matched {
		return "", fmt.Errorf("Token解析错误")
	}
	return refreshToken, nil
}

// GetWebapiNewToken 获取Webapi Token
func (h *LoginHelper) GetWebapiNewToken(ticketId, userId string) (*LoginTokenResult, error) {
	fullUrl := strings.Builder{}
	fmt.Fprintf(&fullUrl, "%s/auth/tickstep/aliyunpan/token/webapi/%s/refresh?userId=%s",
		h.webHost, ticketId, userId)
	logger.Verboseln("do request url: " + fullUrl.String())
	header := map[string]string{
		"accept":       "application/json, text/plain, */*",
		"content-type": "application/json;charset=UTF-8",
		"user-agent":   "aliyunpan/" + global.AppVersion,
	}
	// request
	client := requester.NewHTTPClient()
	client.SetTimeout(20 * time.Second)
	client.SetKeepAlive(false)
	body, err := client.Fetch("GET", fullUrl.String(), nil, header)
	if err != nil {
		logger.Verboseln("get web token result error ", err)
		return nil, err
	}

	errResp := &LoginHttpResult{}
	if err1 := json.Unmarshal(body, errResp); err1 != nil {
		logger.Verboseln("parse result json error ", err1)
		return nil, err1
	}
	if errResp.Code != 0 {
		return nil, fmt.Errorf(errResp.Msg)
	}

	// parse result
	r := &LoginHttpResult{}
	r.Data = &LoginTokenResult{}
	if err2 := json.Unmarshal(body, r); err2 != nil {
		logger.Verboseln("parse web token result json error ", err2)
		return nil, err2
	}
	return r.Data.(*LoginTokenResult), nil
}

// GetOpenapiNewToken 获取Openapi Token
func (h *LoginHelper) GetOpenapiNewToken(ticketId, userId string) (*LoginTokenResult, error) {
	fullUrl := strings.Builder{}
	fmt.Fprintf(&fullUrl, "%s/auth/tickstep/aliyunpan/token/openapi/%s/refresh?userId=%s",
		h.webHost, ticketId, userId)
	logger.Verboseln("do request url: " + fullUrl.String())
	header := map[string]string{
		"accept":       "application/json, text/plain, */*",
		"content-type": "application/json;charset=UTF-8",
		"user-agent":   "aliyunpan/" + global.AppVersion,
	}
	// request
	client := requester.NewHTTPClient()
	client.SetTimeout(20 * time.Second)
	client.SetKeepAlive(false)
	body, err := client.Fetch("GET", fullUrl.String(), nil, header)
	if err != nil {
		logger.Verboseln("get openapi token result error ", err)
		return nil, err
	}

	errResp := &LoginHttpResult{}
	if err1 := json.Unmarshal(body, errResp); err1 != nil {
		logger.Verboseln("parse result json error ", err1)
		return nil, err1
	}
	if errResp.Code != 0 {
		return nil, fmt.Errorf(errResp.Msg)
	}

	// parse result
	r := &LoginHttpResult{}
	r.Data = &LoginTokenResult{}
	if err2 := json.Unmarshal(body, r); err2 != nil {
		logger.Verboseln("parse openapi token result json error ", err2)
		return nil, err2
	}
	return r.Data.(*LoginTokenResult), nil
}

// GetLoginToken 获取登录后的Token
func (h *LoginHelper) GetLoginToken(ticketId string) (*CommonTokenEntity, error) {
	fullUrl := strings.Builder{}
	fmt.Fprintf(&fullUrl, "%s/auth/tickstep/aliyunpan/token/common/%s/login",
		h.webHost, ticketId)
	logger.Verboseln("do request url: " + fullUrl.String())
	header := map[string]string{
		"accept":       "application/json, text/plain, */*",
		"content-type": "application/json;charset=UTF-8",
		"user-agent":   "aliyunpan/" + global.AppVersion,
	}
	// request
	client := requester.NewHTTPClient()
	client.SetTimeout(20 * time.Second)
	client.SetKeepAlive(false)
	body, err := client.Fetch("GET", fullUrl.String(), nil, header)
	if err != nil {
		logger.Verboseln("get login token result error ", err)
		return nil, err
	}

	errResp := &LoginHttpResult{}
	if err1 := json.Unmarshal(body, errResp); err1 != nil {
		logger.Verboseln("parse result json error ", err1)
		return nil, err1
	}
	if errResp.Code != 0 {
		return nil, fmt.Errorf(errResp.Msg)
	}

	// parse result
	r := &LoginHttpResult{}
	r.Data = &CommonTokenEntity{}
	if err2 := json.Unmarshal(body, r); err2 != nil {
		logger.Verboseln("parse login token result json error ", err2)
		return nil, err2
	}
	return r.Data.(*CommonTokenEntity), nil
}
