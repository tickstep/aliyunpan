cd /home/app
chmod +x ./aliyunpan
./aliyunpan login -RefreshToken=${ALIYUNPAN_REFRESH_TOKEN}
./aliyunpan config set -transfer_url_type ${ALIYUNPAN_TRANSFER_URL_TYPE} 
./aliyunpan webdav start -ip "0.0.0.0" -port 23077 -webdav_user "${ALIYUNPAN_AUTH_USER}" -webdav_password "${ALIYUNPAN_AUTH_PASSWORD}" -pan_dir_path "${ALIYUNPAN_PAN_DIR}" -bs ${ALIYUNPAN_BLOCK_SIZE}