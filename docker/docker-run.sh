# This will run the telegraf container with the plugin binary and the configurations from the docker/config folder
echo "Starting telegraf container"
echo "Make sure to have adapted the conf files in docker/config/ to your environment"
docker run \
    --name fritzbox-telegraf \
    --restart unless-stopped \
    -d \
    -v $(pwd)/plugin-binary:/opt/plugin-binary/:ro \
    -v $(pwd)/config:/etc/telegraf/:ro \
    -it \
    telegraf