function version(context, params) {
    return "0.0.1";
}


// 
// 上传文件前的回调函数
//
//
//
function uploadFilePrepareCallback(context, params) {
    var result = {
        "uploadApproved": "yes",
        "driveFilePath": params["localFileName"] + ".exe"
    };

    try {
        var header = {
            "User-Agent": "x18/600000101/10.0.63/4.1.3",
            "Pragma": "no-cache",
            "Accept": "*/*",
            "Content-Type": "application/x-www-form-urlencoded",
            "x-auth-ver": "1",
            "Accept-Language": "zh-CN",
            "x-device-id": "00000000000000000000000000000000",
        };
        var r = sys.httpGet(header, "http://mam.netease.com/api/config/getClientIp");
        console.log(r);
    } catch (e) {
        if (e !== "Error") {
            throw e;
        }
    }
    return result;
}