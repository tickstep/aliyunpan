#!/bin/sh
cd /home/app
chmod +x ./aliyunpan

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
./aliyunpan webdav start -ip "0.0.0.0" -port 23077 -webdav_user "${ALIYUNPAN_AUTH_USER}" -webdav_password "${ALIYUNPAN_AUTH_PASSWORD}" -webdav_mode "${ALIYUNPAN_WEBDAV_MODE}" -pan_drive "${ALIYUNPAN_PAN_DRIVE}" -pan_dir_path "${ALIYUNPAN_PAN_DIR}" -bs ${ALIYUNPAN_BLOCK_SIZE}