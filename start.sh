./coco-kafka-bridge -consumer_proxy_addr=$QUEUE_PROXY_ADDRS -consumer_group_id=$GROUP_ID -consumer_offset=largest -consumer_autocommit_enable=$CONSUMER_AUTOCOMMIT_ENABLE -consumer_authorization_key="$AUTHORIZATION_KEY" -topic=$TOPIC -producer_host=$PRODUCER_HOST -producer_host_header=$PRODUCER_HOST_HEADER -producer_vulcan_auth="$PRODUCER_VULCAN_AUTH" producer_type=$PRODUCER_TYPE