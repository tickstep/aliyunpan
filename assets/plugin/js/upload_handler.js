// ==========================================================================================
// aliyunpan JS插件回调处理函数
// 支持 JavaScript ECMAScript 5.1 语言规范
//
// 更多内容请查看官方文档：https://github.com/tickstep/aliyunpan
// ==========================================================================================


// ------------------------------------------------------------------------------------------
// 函数说明：上传文件前的回调函数
//
// 参数说明
// context - 当前调用的上下文信息
// {
// 	"appName": "aliyunpan",
// 	"version": "v0.1.3",
// 	"userId": "11001d48564f43b3bc5662874f04bb11",
// 	"nickname": "tickstep",
// 	"fileDriveId": "19519111",
// 	"albumDriveId": "29519122"
// }
// appName - 应用名称，当前固定为aliyunpan
// version - 版本号
// userId - 当前登录用户的ID
// nickname - 用户昵称
// fileDriveId - 用户文件网盘ID
// albumDriveId - 用户相册网盘ID
//
// params - 文件上传前参数
// {
// 	"localFilePath": "D:\\Program Files\\aliyunpan\\Downloads\\token.bat",
// 	"localFileName": "token.bat",
// 	"localFileSize": 125330,
// 	"localFileType": "file",
// 	"localFileUpdatedAt": "2022-04-14 07:05:12",
// 	"driveId": "19519221",
// 	"driveFilePath": "aliyunpan/Downloads/token.bat"
// }
// localFilePath - 本地文件绝对完整路径
// localFileName - 本地文件名
// localFileSize - 本地文件大小，单位B
// localFileType - 本地文件类型，file-文件，folder-文件夹
// localFileUpdatedAt - 文件修改时间
// driveId - 准备上传的目标网盘ID
// driveFilePath - 准备上传的目标网盘保存的路径，这个是相对路径，相对指定上传的目标文件夹
//
// 返回值说明
// {
// 	"uploadApproved": "yes",
// 	"driveFilePath": "newfolder/token.bat"
// }
// uploadApproved - 该文件是否确认上传，yes-允许上传，no-禁止上传
// driveFilePath - 文件保存的网盘路径，这个是相对路径，如果为空""代表保持原本的目标路径。
//                 这个改动要小心，会导致重名文件只会上传一个
// ------------------------------------------------------------------------------------------
function uploadFilePrepareCallback(context, params) {
    var result = {
        "uploadApproved": "yes",
        "driveFilePath": ""
    };

    // 所有的.dmg文件，网盘保存的文件增加后缀名为.exe，本地文件不改动
    if (params["localFilePath"].lastIndexOf(".dmg") > 0) {
        result["driveFilePath"] = params["driveFilePath"] + ".exe";
    }

    // 禁止.txt文件上传
    if (params["localFilePath"].lastIndexOf(".txt") > 0) {
        result["uploadApproved"] = "no";
    }

    // 禁止password.key文件上传
    if (params["localFileName"] == "password.key") {
        result["uploadApproved"] = "no";
    }

    return result;
}


// ------------------------------------------------------------------------------------------
// 函数说明：上传文件结束的回调函数
//
// 参数说明
// context - 当前调用的上下文信息
// {
// 	"appName": "aliyunpan",
// 	"version": "v0.1.3",
// 	"userId": "11001d48564f43b3bc5662874f04bb11",
// 	"nickname": "tickstep",
// 	"fileDriveId": "19519111",
// 	"albumDriveId": "29519122"
// }
// appName - 应用名称，当前固定为aliyunpan
// version - 版本号
// userId - 当前登录用户的ID
// nickname - 用户昵称
// fileDriveId - 用户文件网盘ID
// albumDriveId - 用户相册网盘ID
//
// params - 文件上传结束参数
// {
// 	"localFilePath": "D:\\Program Files\\aliyunpan\\Downloads\\token.bat",
// 	"localFileName": "token.bat",
// 	"localFileSize": 125330,
// 	"localFileType": "file",
// 	"localFileUpdatedAt": "2022-04-14 07:05:12",
// 	"localFileSha1": "08FBE28A5B8791A2F50225E2EC5CEEC3C7955A11",
// 	"uploadResult": "success",
// 	"driveId": "19519221",
// 	"driveFilePath": "/tmp/test/aliyunpan/Downloads/token.bat"
// }
// localFilePath - 本地文件绝对完整路径
// localFileName - 本地文件名
// localFileSize - 本地文件大小，单位B
// localFileType - 本地文件类型，file-文件，folder-文件夹
// localFileUpdatedAt - 文件修改时间
// localFileSha1 - 本地文件的SHA1。这个值不一定会有
// uploadResult - 上传结果，success-成功，fail-失败
// driveId - 目标网盘ID
// driveFilePath - 文件网盘保存的绝对路径
//
// 返回值说明
// （没有返回值）
// ------------------------------------------------------------------------------------------
function uploadFileFinishCallback(context, params) {
    console.log(params)
}