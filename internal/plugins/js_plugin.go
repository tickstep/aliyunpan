package plugins

import (
	"fmt"
	"github.com/dop251/goja"
	"github.com/tickstep/library-go/logger"
	"strings"
)

type (
	JsPlugin struct {
		Name string
		vm   *goja.Runtime
	}
)

func NewJsPlugin() *JsPlugin {
	return &JsPlugin{
		Name: "JsPlugin",
		vm:   nil,
	}
}

// jsLog 支持js中的console.log方法
func jsLog(call goja.FunctionCall) goja.Value {
	str := call.Argument(0)
	buf := &strings.Builder{}
	fmt.Fprintf(buf, "JAVASCRIPT: %+v", str.Export())
	logger.Verboseln(buf.String())
	return str
}

func (js *JsPlugin) Start() error {
	js.Name = "JsPlugin"
	js.vm = goja.New()
	js.vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	// 内置log
	console := js.vm.NewObject()
	console.Set("log", jsLog)
	js.vm.Set("console", console) // console.log()

	// 内置系统函数PluginUtil
	pluginObj := js.vm.NewObject()
	js.vm.Set("PluginUtil", pluginObj)

	// PluginUtil.Http
	httpObj := js.vm.NewObject()
	pluginObj.Set("Http", httpObj)
	httpObj.Set("get", HttpGet)   // PluginUtil.Http.get()
	httpObj.Set("post", HttpPost) // PluginUtil.Http.post()

	// PluginUtil.LocalFS
	localFS := js.vm.NewObject()
	pluginObj.Set("LocalFS", localFS)
	localFS.Set("deleteFile", DeleteLocalFile) // PluginUtil.LocalFS.deleteFile()

	return nil
}

// LoadScript 加载脚本
func (js *JsPlugin) LoadScript(script string) error {
	_, err := js.vm.RunString(script)
	if err != nil {
		logger.Verboseln("JS代码有问题！{}", err)
		return err
	}
	return nil
}

func (js *JsPlugin) isHandlerFuncExisted(fnName string) bool {
	ret := js.vm.Get(fnName)
	if ret != nil {
		return true
	}
	return false
}

// UploadFilePrepareCallback 上传文件前的回调函数
func (js *JsPlugin) UploadFilePrepareCallback(context *Context, params *UploadFilePrepareParams) (*UploadFilePrepareResult, error) {
	var fn func(*Context, *UploadFilePrepareParams) (*UploadFilePrepareResult, error)
	if !js.isHandlerFuncExisted("uploadFilePrepareCallback") {
		return nil, nil
	}
	err := js.vm.ExportTo(js.vm.Get("uploadFilePrepareCallback"), &fn)
	if err != nil {
		logger.Verboseln("Js函数映射到 Go 函数失败！")
		return nil, nil
	}
	r, er := fn(context, params)
	if er != nil {
		logger.Verboseln(er)
		return nil, er
	}
	return r, nil
}

// UploadFileFinishCallback 上传文件结束的回调函数
func (js *JsPlugin) UploadFileFinishCallback(context *Context, params *UploadFileFinishParams) error {
	var fn func(*Context, *UploadFileFinishParams) error
	if !js.isHandlerFuncExisted("uploadFileFinishCallback") {
		return nil
	}
	err := js.vm.ExportTo(js.vm.Get("uploadFileFinishCallback"), &fn)
	if err != nil {
		logger.Verboseln("Js函数映射到 Go 函数失败！")
		return nil
	}
	er := fn(context, params)
	if er != nil {
		logger.Verboseln(er)
		return nil
	}
	return nil
}

// DownloadFilePrepareCallback 下载文件前的回调函数
func (js *JsPlugin) DownloadFilePrepareCallback(context *Context, params *DownloadFilePrepareParams) (*DownloadFilePrepareResult, error) {
	var fn func(*Context, *DownloadFilePrepareParams) (*DownloadFilePrepareResult, error)
	if !js.isHandlerFuncExisted("downloadFilePrepareCallback") {
		return nil, nil
	}
	err := js.vm.ExportTo(js.vm.Get("downloadFilePrepareCallback"), &fn)
	if err != nil {
		logger.Verboseln("Js函数映射到 Go 函数失败！")
		return nil, nil
	}
	r, er := fn(context, params)
	if er != nil {
		logger.Verboseln(er)
		return nil, er
	}
	return r, nil
}

// DownloadFileFinishCallback 下载文件结束的回调函数
func (js *JsPlugin) DownloadFileFinishCallback(context *Context, params *DownloadFileFinishParams) error {
	var fn func(*Context, *DownloadFileFinishParams) error
	if !js.isHandlerFuncExisted("downloadFileFinishCallback") {
		return nil
	}
	err := js.vm.ExportTo(js.vm.Get("downloadFileFinishCallback"), &fn)
	if err != nil {
		logger.Verboseln("Js函数映射到 Go 函数失败！")
		return nil
	}
	er := fn(context, params)
	if er != nil {
		logger.Verboseln(er)
		return nil
	}
	return nil
}

// SyncScanLocalFilePrepareCallback 同步备份-扫描本地文件的回调函数
func (js *JsPlugin) SyncScanLocalFilePrepareCallback(context *Context, params *SyncScanLocalFilePrepareParams) (*SyncScanLocalFilePrepareResult, error) {
	var fn func(*Context, *SyncScanLocalFilePrepareParams) (*SyncScanLocalFilePrepareResult, error)
	if !js.isHandlerFuncExisted("syncScanLocalFilePrepareCallback") {
		return nil, nil
	}
	err := js.vm.ExportTo(js.vm.Get("syncScanLocalFilePrepareCallback"), &fn)
	if err != nil {
		logger.Verboseln("Js函数映射到 Go 函数失败！")
		return nil, nil
	}
	r, er := fn(context, params)
	if er != nil {
		logger.Verboseln(er)
		return nil, er
	}
	return r, nil
}

// SyncScanPanFilePrepareCallback 同步备份-扫描本地文件的回调函数
func (js *JsPlugin) SyncScanPanFilePrepareCallback(context *Context, params *SyncScanPanFilePrepareParams) (*SyncScanPanFilePrepareResult, error) {
	var fn func(*Context, *SyncScanPanFilePrepareParams) (*SyncScanPanFilePrepareResult, error)
	if !js.isHandlerFuncExisted("syncScanPanFilePrepareCallback") {
		return nil, nil
	}
	err := js.vm.ExportTo(js.vm.Get("syncScanPanFilePrepareCallback"), &fn)
	if err != nil {
		logger.Verboseln("Js函数映射到 Go 函数失败！")
		return nil, nil
	}
	r, er := fn(context, params)
	if er != nil {
		logger.Verboseln(er)
		return nil, er
	}
	return r, nil
}

func (js *JsPlugin) Stop() error {
	return nil
}
