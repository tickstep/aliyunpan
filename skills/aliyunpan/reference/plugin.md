# 目录
- [简介](#简介)
- [如何使用](#如何使用)
- [JS中内置的函数](#JS中内置的函数)
    + [console.log()](#consolelog)
    + [console.println()](#consoleprintln)
    + [PluginUtil.Http.get()](#PluginUtilHttpget)
    + [PluginUtil.Http.post()](#PluginUtilHttppost)
    + [PluginUtil.LocalFS.deleteFile()](#PluginUtilLocalFSdeleteFile)
    + [PluginUtil.PanFS.deleteFile()](#PluginUtilPanFSdeleteFile)
    + [PluginUtil.Email.sendTextMail()](#PluginUtilEmailsendTextMail)
    + [PluginUtil.Email.sendHtmlMail()](#PluginUtilEmailsendHtmlMail)
    + [PluginUtil.KV.putString()](#PluginUtilKVputString)
    + [PluginUtil.KV.getString()](#PluginUtilKVgetString)
    + [PluginUtil.HashTool.md5Hex()](#PluginUtilHashToolmd5Hex)
- [常见场景样例](#常见场景样例)
    + [1.禁止特定文件上传](#1禁止特定文件上传)
    + [2.上传文件后删除本地文件](#2上传文件后删除本地文件)
    + [3.下载文件并截断过长的文件名](#3下载文件并截断过长的文件名)
    + [4.上传文件去掉文件名包含的部分字符](#4上传文件去掉文件名包含的部分字符)
    + [5.上传文件时过滤指定目录或者文件路径](#5上传文件时过滤指定目录或者文件路径)
    + [6.下载云盘文件到本地后删除云盘对应的文件](#6下载云盘文件到本地后删除云盘对应的文件)
    + [7.Token刷新失败发送外部通知](#7Token刷新失败发送外部通知)
    + [8.每次只下载指定数量的文件](#8每次只下载指定数量的文件)

# 简介
本程序支持javascript插件。通过JS插件，你可以按照自己的需要定制上传、下载、同步、删除过程中关键步骤的行为，最大程度满足自己的个性化需求。   
例如：   
1. 排除某个特定敏感文件的上传   
2. 上传文件进行改名，但是本地文件不做更改   
3. 上传完成文件，删除本地文件   
4. 上传的文件路径进行更改，但是本地的文件保持不变   
5. 上传文件成功后，通过HTTP通知其他服务   
6. 排除某些网盘文件的下载   
7. 下载的文件进行改名，但是网盘的文件保持不变   
8. 下载的文件路径进行更改，但是网盘的文件保持不变   
9. 下载文件完成后，通过HTTP通知其他服务   
10. 同步备份功能，支持过滤本地文件，或者过滤云盘文件。定制上传或者下载需要同步的文件

# 如何使用
JS插件的样本文件默认存放在程序所在的```plugin/js```文件夹下，分别为：
1. 下载插件(download_handler.js.sample)
2. 上传插件(upload_handler.js.sample)
3. 删除插件(remove_handler.js.sample)
4. 同步备份插件(sync_handler.js.sample)
5. 用户Token插件(token_handler.js.sample)

建议拷贝一份并将后缀名更改为.js，例如：upload_handler.js，不然插件不会生效。   
你必须具备一定的JS语言基础，然后按照里面的样例根据自己所需进行改动即可。如果你不会JS那也没关系，你可以提issue需求，然后我们开发成员或者网友会给你提供JS脚本代码。   
注意：如果你有通过环境变量```ALIYUNPAN_CONFIG_DIR```设置配置目录，则需要将plugin文件夹拷贝到配置的目录中才可以生效。

# JS中内置的函数
目前开放了如下函数，你可以在你的js脚本中直接调用，以用于增强JS脚本的扩展性、可玩性以及可适用性。  

## console.log()
打印日志，这个日志需要开启debug日志才会在控制台窗口显示
```js
console.log("hello world");
```
## console.println()
打印日志，这个日志会直接在控制台窗口显示，无需开启debug日志
```js
console.println("hello world");
```

## PluginUtil.Http.get()
发起HTTP的get请求
```js
    var header = {
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.88 Safari/537.36",
        "Content-Type": "application/json",
        "Accept": "application/json"
    };
    try {
        var r = PluginUtil.Http.get(header, "https://625f528c53a42eaa07f37e13.mockapi.io/files/1");
        console.log(r);
    } catch (e) {
        if (e !== "Error") {
            throw e;
        }
    }
```

## PluginUtil.Http.post()
发起HTTP的post请求
```js
    var header = {
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.88 Safari/537.36",
        "Content-Type": "application/json",
        "Accept": "application/json"
    };
    try {
        var reqDataStr = JSON.stringify({
            "id": "1",
            "localFilePath": "/usr/local/src/borders_burundi_producer.gram.aab",
            "localFileSize": 1111,
            "uploadApproved": false
        });
        var r = PluginUtil.Http.post(header, "https://625f528c53a42eaa07f37e13.mockapi.io/files", reqDataStr);
        console.log(r);
    } catch (e) {
        if (e !== "Error") {
            throw e;
        }
    }
```

## PluginUtil.LocalFS.deleteFile()
删除本地指定文件，不支持文件夹
```
PluginUtil.LocalFS.deleteFile(localFilePath);
其中：
localFilePath - 本地文件的绝对完整路径
```
样例
```js
PluginUtil.LocalFS.deleteFile("/Users/tickstep/Downloads/IMG_0884.HEIC");
```

## PluginUtil.PanFS.deleteFile()
删除云盘指定文件，支持文件、文件夹
```
PluginUtil.PanFS.deleteFile(userId, driveId, panFileId);
其中：
userId - 登录的用户ID
driveId - 网盘ID
panFileId - 网盘文件ID
```
样例
```js
var userId = context["userId"]
var driveId = params["driveId"]
var driveFileId = params["driveFileId"]
if (PluginUtil.PanFS.deleteFile(userId, driveId, driveFileId)) {
    console.println("插件删除云盘文件成功")
}
```

## PluginUtil.Email.sendTextMail()
发送文本邮件，定义如下
```
PluginUtil.Email.sendTextMail(mailServer, userName, password, to, subject, body)

其中：
mailServer - smtp服务器+端口
userName - 发件人邮箱地址
password - 发件人邮箱密码
to - 收件人邮箱地址
subject - 邮件标题
body - 邮件内容，纯文本
```
样例
```js
PluginUtil.Email.sendTextMail("smtp.qq.com:465", "123xxx@qq.com", "pwdxxxxxx", "12545xxx@qq.com", "文件上传通知", "该文件已经上传完毕");
```

## PluginUtil.Email.sendHtmlMail()
发送HTML富文本邮件，定义如下
```
PluginUtil.Email.sendHtmlMail(mailServer, userName, password, to, subject, body)

其中：
mailServer - smtp服务器+端口
userName - 发件人邮箱地址
password - 发件人邮箱密码
to - 收件人邮箱地址
subject - 邮件标题
body - 邮件内容，HTML富文本
```
样例
```js
var html = "<html>"
    +"<body>"
    +"<h1>文件上传通知</h1>"
    +"<h2>该文件已经上传完毕</h2>"
    +"</body>"
    +"</html>";
PluginUtil.Email.sendHtmlMail("smtp.qq.com:465", "123xxx@qq.com", "pwdxxxxxx", "12545xxx@qq.com", "文件上传通知", html);
```

## PluginUtil.KV.putString()
存储键值对，定义如下
```
PluginUtil.KV.putString(key, value)

其中：
key - 键，字符串
value - 值，字符串
```
样例
```js
PluginUtil.KV.putString("mykey", "1670419352");
```

## PluginUtil.KV.getString()
获取存储的键值，指定对应的键，返回存储的值，如果没有则返回空字符串，定义如下
```
PluginUtil.KV.getString(key)

其中：
key - 键，字符串
```
样例
```js
var value = PluginUtil.KV.getString("mykey");
```

## PluginUtil.HashTool.md5Hex()
计算指定字符串text的MD5值，返回hex格式的字符串MD5，定义如下
```
PluginUtil.HashTool.md5Hex(text)

其中：
text - 字符串
```
样例
```js
var md5 = PluginUtil.HashTool.md5Hex("123456");
```

# 常见场景样例
这里收集了一些常见的需求样例，可以作为插件定制的样例模板。

## 1.禁止特定文件上传
使用JavaScript上传插件中的`uploadFilePrepareCallback`函数。如下所示：
```js
function uploadFilePrepareCallback(context, params) {
    console.log(params)

    var result = {
        "uploadApproved": "yes",
        "driveFilePath": ""
    };
    if (params["localFileType"] != "file") {
        // do nothing
        return result;
    }

    // 禁止点号.开头的文件上传
    if (params["localFileName"].indexOf(".") == 0) {
        result["uploadApproved"] = "no";
    }

    // 禁止.txt文件上传
    if (params["localFileName"].search(/.txt$/i) >= 0) {
        result["uploadApproved"] = "no";
    }

    // 禁止password.key文件上传
    if (params["localFileName"] == "password.key") {
        result["uploadApproved"] = "no";
    }    

    return result;
}
```

## 2.上传文件后删除本地文件
使用JavaScript上传插件中的`uploadFileFinishCallback`函数。如下所示：
```js
function uploadFileFinishCallback(context, params) {
    console.log(params);
    if (params["localFileType"] != "file") {
        // do nothing
        return;
    }
    if (params["uploadResult"] == "success") {
        PluginUtil.LocalFS.deleteFile(params["localFilePath"]);
    }    
}
```

## 3.下载文件并截断过长的文件名
有些文件的路径或者名称太长，下载的时候可能会由于路径名称过程导致无法下载，这时候可以使用JavaScript下载插件中的`downloadFilePrepareCallback`函数定制下载保存的文件名。   
如下所示：
```js
function downloadFilePrepareCallback(context, params) {
    console.log(params)

    var result = {
        "downloadApproved": "yes",
        "localFilePath": ""
    };
    if (params["driveFileType"] != "file") {
        return result;
    }
	
    // 下面的代码都是分隔路径，方便后面修改路径时使用
    var filePath =  params["localFilePath"];
    filePath = filePath.replace(/\\/g, "/");
    
    // 目录完整路径
    var dirPath = "";
    // 文件名，不包括后缀名
    var fileName = "";
    // 文件后缀名
    var fileExt = "";
    
    var idx = filePath.lastIndexOf('/');
    if (idx > 0) {
        dirPath = filePath.substring(0,idx);
        fileName = filePath.substring(idx+1,filePath.length);
    } else {
        fileName = filePath;
    }
    idx = fileName.lastIndexOf(".")
    if (idx > 0) {
        fileExt = fileName.substring(idx,fileName.length);
        fileName = fileName.substring(0, fileName.length-fileExt.length)
    }

    // 开始按照需要截断太长的文件路径，例如下面的这个例子:
    // 
    // dirPath + "/"          ==> 这个的意思是保留前面的文件夹路径，如果不需要就去掉
    // fileName.substr(0,10)  ==> 文件名只取前面10个字符，其他的不要了
    // + fileExt              ==> 把文件的后缀名补上
    var saveFilePath = dirPath + "/" + fileName.substr(0,10) + fileExt;

    // 返回
    result["localFilePath"] = saveFilePath;
    return result;
}
```
效果如下:   
1）假如网盘源路径是：`/亚马逊书籍合集/亚马逊 kindle ebook 大合集5289册/经济金融 (2)/解密Instagram（ 《金融时报》和麦肯锡2020年度商业书籍！社交应用如何改变世界？解锁打造估值千亿美元爆品的核心方法！ ) by 莎拉·弗莱尔(z-lib.org).mobi`   
2）下载后保存本地的路径是：`D:\test\亚马逊书籍合集\亚马逊 kindle ebook 大合集5289册\经济金融 (2)\解密Instagra.mobi`   
```
[1] ----
文件ID: 630f0623e67c0a1d2e554b1994a29108d089f0d5
文件名: 解密Instagram（ 《金融时报》和麦肯锡2020年度商业书籍！社交应用如何改变世界？解锁打造估值千亿美元爆品的核心方法！ ) by 莎拉·弗莱尔(z-lib.org).mobi
文件类型: 文件
文件路径: /亚马逊书籍合集/亚马逊 kindle ebook 大合集5289册/经济金融 (2)/解密Instagram（ 《金融时报》和麦肯锡2020年度商业书籍！社交应用如何改变世界？解锁打造估值千亿美元爆品的核心方法！ ) by 莎拉·弗莱尔(z-lib.org).mobi

插件修改文件下载保存路径为: D:\test\亚马逊书籍合集/亚马逊 kindle ebook 大合集5289册/经济金融 (2)/解密Instagra.mobi
[1] 准备下载: /亚马逊书籍合集/亚马逊 kindle ebook 大合集5289册/经济金融 (2)/解密Instagram（ 《金融时报》和麦肯锡2020年度商业书籍！社交应用如何改变世界？解锁打造估值千亿美元爆品的核心方法！ ) by 莎拉·弗莱尔(z-lib.org).mobi
[1] 将会下载到路径: D:\test\亚马逊书籍合集/亚马逊 kindle ebook 大合集5289册/经济金融 (2)/解密Instagra.mobi
[1] 下载开始

[1] 下载完成, 保存位置: D:\test\亚马逊书籍合集/亚马逊 kindle ebook 大合集5289册/经济金融 (2)/解密Instagra.mobi
[1] 检验文件有效性成功: D:\test\亚马逊书籍合集/亚马逊 kindle ebook 大合集5289册/经济金融 (2)/解密Instagra.mobi
```
## 4.上传文件去掉文件名包含的部分字符
例如本地文件夹名称为```[周杰伦]范特西[mp3]```，上传到网盘希望能更改成名称```[周杰伦]范特西```但保持本地文件夹名称不变，因为文件夹有很多不希望每个手动更改，可以这样实现：
```js
function uploadFilePrepareCallback(context, params) {
    var result = {
        "uploadApproved": "yes",
        "driveFilePath": ""
    };

    // 去掉网盘保存路径中包含的[mp3]字段
    var filePath =  params["driveFilePath"];
    filePath = filePath.replace(/\[mp3\]/g, "");
    result["driveFilePath"] = filePath;

    return result;
}
```
## 5.上传文件时过滤指定目录或者文件路径
upload命令本身支持exn参数排除上传文件或目录，但是只能指定文件名称，不能指定文件的路径。如果需要排除指定路径则可以通过插件脚本实现，样例如下：
```js
function uploadFilePrepareCallback(context, params) {
    //（自行配置）禁止上传的本地【目录列表】，使用绝对路径，可以配置多个路径用逗号分隔
    var forbiddenUploadFolders = ["/Users/tickstep/Downloads/up/target","/Users/tickstep/Downloads/up/.idea"]
    //（自行配置）禁止上传的本地【文件列表】，使用绝对路径，可以配置多个路径用逗号分隔
    var forbiddenUploadFiles = ["/Users/tickstep/Downloads/up/pom.xml"]

    // -------------------- 以下代码不要修改 --------------------
    var result = {
        "uploadApproved": "yes",
        "driveFilePath": ""
    };

    // 下面的代码都是分隔路径，方便后面使用
    var filePath =  params["localFilePath"];
    filePath = filePath.replace(/\\/g, "/");
    // 目录完整路径
    var dirPath = "";
    // 文件名，不包括后缀名
    var fileName = "";
    // 文件后缀名
    var fileExt = "";
    var idx = filePath.lastIndexOf('/');
    if (idx > 0) {
        dirPath = filePath.substring(0,idx);
        fileName = filePath.substring(idx+1,filePath.length);
    } else {
        fileName = filePath;
    }
    idx = fileName.lastIndexOf(".")
    if (idx > 0) {
        fileExt = fileName.substring(idx,fileName.length);
        fileName = fileName.substring(0, fileName.length-fileExt.length)
    }

    if (params["localFileType"] == "file") {
        for (var i = 0; i < forbiddenUploadFiles.length; i++) {
            if (forbiddenUploadFiles[i].replace(/\\/g, "/") == params["localFilePath"]) {
                result["uploadApproved"] = "no"; // 禁止文件上传
                break
            }
        }
        for (var i = 0; i < forbiddenUploadFolders.length; i++) {
            if (forbiddenUploadFolders[i].replace(/\\/g, "/") == dirPath) {
                result["uploadApproved"] = "no"; // 禁止文件上传
                break
            }
        }
    } else if (params["localFileType"] == "folder") {
        for (var i = 0; i < forbiddenUploadFolders.length; i++) {
            if (forbiddenUploadFolders[i].replace(/\\/g, "/") == params["localFilePath"]) {
                result["uploadApproved"] = "no"; // 禁止文件上传
                break
            }
        }
    }

    return result;
}
```
## 6.下载云盘文件到本地后删除云盘对应的文件
```js
function downloadFileFinishCallback(context, params) {
    console.log(params)
    // 云盘文件成功下载到本地后，删除云盘的文件
    if (params["downloadResult"] == "success") {
        if (params["driveFileType"] == "file") {
            // 文件下载成功，删除该云盘文件
            var userId = context["userId"]
            var driveId = params["driveId"]
            var driveFileId = params["driveFileId"]
            if (PluginUtil.PanFS.deleteFile(userId, driveId, driveFileId)) {
                console.println("插件删除云盘文件成功：" + params["driveFilePath"])
            }
        }
    }
}
```

## 7.Token刷新失败发送外部通知
Token刷新失败发送外部通知，例如Server酱
```js
function userTokenRefreshFinishCallback(context, params) {
    var header = {
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.88 Safari/537.36",
        "Content-Type": "application/json",
        "Accept": "application/json"
    };
    try {
        if (params["result"] === "fail") {
            // 避免频繁发送
            var ONE_MINUTE = 60 * 1000;
            var lastSendEmailTime = PluginUtil.KV.getString("email_last_send_time");
            if (lastSendEmailTime != "") {
                if ((Date.now() - lastSendEmailTime) < (10 * ONE_MINUTE)) {
                    console.log("距离上次发送邮件小于10分钟，先不发送了");
                    return
                }
            }
            PluginUtil.KV.putString("email_last_send_time", Date.now());
            
            var reqData = {
                "text": "Token刷新失败",
                "desp": params["message"]
            };
            var r = PluginUtil.Http.post(header, "https://sctapi.ftqq.com/xxxxxkeyxxxxxx.send", JSON.stringify(reqData));
        }
    } catch (e) {
        if (e !== "Error") {
            throw e;
        }
    }
}
```
或者发送邮件通知
```js
function userTokenRefreshFinishCallback(context, params) {
    try {
        if (params["result"] === "fail") {
            // 避免频繁发送
            var ONE_MINUTE = 60 * 1000;
            var lastSendEmailTime = PluginUtil.KV.getString("email_last_send_time");
            if (lastSendEmailTime != "") {
                if ((Date.now() - lastSendEmailTime) < (10 * ONE_MINUTE)) {
                    console.log("距离上次发送邮件小于10分钟，先不发送了");
                    return
                }
            }
            PluginUtil.KV.putString("email_last_send_time", Date.now());

            // 发送通知邮件，请确保你的发送邮箱具备smtp发送权限，没有的需要找邮箱提供商配置开启
            console.log("发送通知邮件");
            PluginUtil.Email.sendTextMail("smtp.qq.com:465", "123xxx@qq.com", "pwdxxxxxx", "12545xxx@qq.com", "Token刷新失败了", "Token过期了，骚年快去手动恢复吧");
        }
    } catch (e) {
        if (e !== "Error") {
            throw e;
        }
    }
}
```
## 8.每次只下载指定数量的文件
每次运行download下载命令，只下载指定数量的文件，只要数量够了其他文件就跳过不再下载，其他文件等下一次运行download命令的时候再下载。
```js
function downloadFilePrepareCallback(context, params) {
    var result = {
        "downloadApproved": "yes",
        "localFilePath": ""
    };

    // 这次下载限制的最大文件总数量
    const maxCountOfDownloadAction = 3;

    // 只处理文件
    if (params["driveFileType"] != "file") {
        return
    }

    // 获取这次下载动作，已经下载的文件数量
    var keyOfThisDownloadAction = "download:" + params["downloadActionId"];
    var valueOfThisDownloadAction = PluginUtil.KV.getString(keyOfThisDownloadAction);
    if (valueOfThisDownloadAction == "") {
        valueOfThisDownloadAction = "0";
    }
    var countOfThisDownloadAction = parseInt(valueOfThisDownloadAction);
    if (countOfThisDownloadAction >= maxCountOfDownloadAction) {
        // 下载数量已达到最大，其他文件不再下载
        result["downloadApproved"] = "no";
        return result;
    }

    // 该文件需要下载，记录相关信息
    var keyOfThisDownloadFile = "file:" + PluginUtil.HashTool.md5Hex(params["driveFilePath"]);
    var valueOfThisDownloadFile = PluginUtil.KV.getString(keyOfThisDownloadFile);
    if (valueOfThisDownloadFile == "finish") {
        // 已经在下载的文件不做处理
        return result;
    }
    PluginUtil.KV.putString(keyOfThisDownloadFile, "downloading");

    // 更新下载总数量
    countOfThisDownloadAction += 1;
    PluginUtil.KV.putString(keyOfThisDownloadAction, String(countOfThisDownloadAction));

    return result;
}

function downloadFileFinishCallback(context, params) {
  var keyOfThisDownloadFile = "file:" + PluginUtil.HashTool.md5Hex(params["driveFilePath"]);
  console.log(keyOfThisDownloadFile)
  PluginUtil.KV.putString(keyOfThisDownloadFile, "finish");
}
```
