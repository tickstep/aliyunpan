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
      "id": "5b2d7c10-e927-4e72-8f9d-5abb3bb04815",
      "localFolderPath": "$ALIYUNPAN_LOCAL_DIR",
      "panFolderPath": "$ALIYUNPAN_PAN_DIR",
      "mode": "$ALIYUNPAN_SYNC_MODE",
      "policy": "$ALIYUNPAN_SYNC_POLICY",
      "driveName": "$ALIYUNPAN_SYNC_DRIVE",
      "lastSyncTime": "2022-06-12 19:28:20"
     }
    ]
}
EOF
fi

sleep 2s

# device-id
#if [[ -z $ALIYUNPAN_DEVICE_ID ]];
#then
#  echo "the program use random device id"
#else
#  echo "set device id"
#  ./aliyunpan config set -device_id ${ALIYUNPAN_DEVICE_ID}
#fi

# show docker IPs
./aliyunpan tool getip

# check login already or not
./aliyunpan who
if [ $? -eq 0 ]
then
  echo "cache token is valid, not need to re-login"
else
  echo "token is invalid, please use the valid aliyunpan_config.json file and retry"
#  ./aliyunpan login -RefreshToken=${ALIYUNPAN_REFRESH_TOKEN}
fi

if [ "$ALIYUNPAN_SYNC_LOG" = "true" ]
then
  ./aliyunpan config set -file_record_config 1
else
  ./aliyunpan config set -file_record_config 2
fi

./aliyunpan sync start -dp ${ALIYUNPAN_DOWNLOAD_PARALLEL} -up ${ALIYUNPAN_UPLOAD_PARALLEL} -dbs ${ALIYUNPAN_DOWNLOAD_BLOCK_SIZE} -ubs ${ALIYUNPAN_UPLOAD_BLOCK_SIZE} -log ${ALIYUNPAN_SYNC_LOG} -ldt ${ALIYUNPAN_LOCAL_DELAY_TIME} -cycle ${ALIYUNPAN_SYNC_CYCLE} -sit ${ALIYUNPAN_SCAN_INTERVAL_TIME}
