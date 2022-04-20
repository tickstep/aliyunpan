package plugin

import (
	"fmt"
	"github.com/dop251/goja"
	"github.com/tickstep/library-go/logger"
	"strings"
)

type (
	JsPlugin struct {
		Name   string
		vm     *goja.Runtime
		logger *logger.CmdVerbose
	}
)

func NewJsPlugin(log *logger.CmdVerbose) *JsPlugin {
	return &JsPlugin{
		Name:   "JsPlugin",
		vm:     nil,
		logger: log,
	}
}

// jsLog 支持js中的console.log方法
func jsLog(call goja.FunctionCall) goja.Value {
	str := call.Argument(0)
	buf := &strings.Builder{}
	fmt.Fprintf(buf, "%+v", str.Export())
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
	js.vm.Set("console", console)

	// 内置系统函数sys
	sysObj := js.vm.NewObject()
	sysObj.Set("httpGet", HttpGet)
	sysObj.Set("httpPost", HttpPost)
	js.vm.Set("sys", sysObj)

	return nil
}

// LoadScript 加载脚本
func (js *JsPlugin) LoadScript(script string) error {
	_, err := js.vm.RunString(script)
	if err != nil {
		fmt.Println("JS代码有问题！")
		return err
	}
	return nil
}

func (js *JsPlugin) UploadFilePrepareCallback(context *Context, params *UploadFilePrepareParams) (*UploadFilePrepareResult, error) {
	var fn func(*Context, *UploadFilePrepareParams) (*UploadFilePrepareResult, error)
	err := js.vm.ExportTo(js.vm.Get("uploadFilePrepareCallback"), &fn)
	if err != nil {
		fmt.Println("Js函数映射到 Go 函数失败！")
		return nil, nil
	}
	r, er := fn(context, params)
	if er != nil {
		fmt.Println(er)
	}
	return r, nil
}

func (js *JsPlugin) Stop() error {
	return nil
}
