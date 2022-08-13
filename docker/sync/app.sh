#!/bin/sh
cd /home/app
chmod +x ./aliyunpan

# sync config file
readonly sync_drive_config_file="$ALIYUNPAN_CONFIG_DIR/sync_drive/sync_drive_config.json"
if test -s $sync_drive_config_file
then
  echo "using existed sync_drive_config.json file"
else
  echo "generate sync_drive_config.json file"
  tee $sync_drive_config_file << EOF
{
    "configVer": "1.0",
    "syncTaskList": [
     {
      "name": "阿里云盘备份",
      "id": "5b2d7c10-e927-4e72-8f9d-5abb3bb04814",
      "localFolderPath": "$ALIYUNPAN_LOCAL_DIR",
      "panFolderPath": "$ALIYUNPAN_PAN_DIR",
      "mode": "$ALIYUNPAN_SYNC_MODE",
      "priority": "$ALIYUNPAN_SYNC_PRIORITY",
      "lastSyncTime": "2022-06-12 19:28:20"
     }
    ]
}
EOF
fi

sleep 2s

# check login already or not
./aliyunpan who
if [ $? -eq 0 ]
then
  echo "cache token is valid, not need to re-login"
else
  echo "login use refresh token: ${ALIYUNPAN_REFRESH_TOKEN}"
  ./aliyunpan login -RefreshToken=${ALIYUNPAN_REFRESH_TOKEN}
fi

./aliyunpan config set -transfer_url_type ${ALIYUNPAN_TRANSFER_URL_TYPE} 
./aliyunpan sync start -dp ${ALIYUNPAN_DOWNLOAD_PARALLEL} -up ${ALIYUNPAN_UPLOAD_PARALLEL} -dbs ${ALIYUNPAN_DOWNLOAD_BLOCK_SIZE} -ubs ${ALIYUNPAN_UPLOAD_BLOCK_SIZE}
