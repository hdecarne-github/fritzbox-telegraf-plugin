
# This will build the plugin binary and copy it to the plugin-binary folder
docker run --rm -v $(pwd)/plugin-binary:/opt/plugin-binary/ golang:1.22 bash -c "\
    mkdir /opt/fritzbox-telegraf && \
    git clone https://github.com/vlorian-de/fritzbox-telegraf-plugin.git /opt/fritzbox-telegraf && \
    cd /opt/fritzbox-telegraf && \
    make && \
    cp build/bin/fritzbox-telegraf-plugin /opt/plugin-binary/fritzbox-telegraf-plugin && \
    chmod + x /opt/plugin-binary/fritzbox-telegraf-plugin"
