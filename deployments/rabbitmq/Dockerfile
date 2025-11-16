FROM rabbitmq:4.2.0-management

RUN apt-get update && \
apt-get install -y curl

RUN curl -L -o /opt/rabbitmq/plugins/rabbitmq_delayed_message_exchange-4.2.0.ez \
https://github.com/rabbitmq/rabbitmq-delayed-message-exchange/releases/download/v4.2.0/rabbitmq_delayed_message_exchange-4.2.0.ez
