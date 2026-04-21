# makeline-service

This is a Golang app that provides an API for processing orders. It is meant to be used in conjunction with the store-admin app.

It is a simple REST API written with the Gin framework that allows you to process orders from a RabbitMQ queue and send them to a MongoDB database.

## Prerequisites

- [Go](https://golang.org/doc/install)
- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)
- [MongoSH](https://docs.mongodb.com/mongodb-shell/install/)

## Message queue options

This app can connect to either RabbitMQ or Azure Service Bus using AMQP 1.0. To connect to either of these services, you will need to provide appropriate environment variables for connecting to the message queue.

### Option 1: RabbitMQ

To run this against RabbitMQ. A docker-compose file is provided to make this easy. This will run RabbitMQ, the RabbitMQ Management UI, and enable the `rabbitmq_amqp1_0` plugin. The plugin is necessary to connect to RabbitMQ using AMQP 1.0.

With the services running, open a new terminal and navigate to the `makeline-service` directory.

Set the connection information for the RabbitMQ queue by running the following commands to set the environment variables:

```bash
export ORDER_QUEUE_URI=amqp://localhost
export ORDER_QUEUE_USERNAME=username
export ORDER_QUEUE_PASSWORD=password
export ORDER_QUEUE_NAME=orders
```

### Option 2: Azure Service Bus

To run this against Azure Service Bus, you will need to create a Service Bus namespace and a queue. You can do this using the Azure CLI.

```bash
RGNAME=<resource-group-name>
LOCNAME=<location>

az group create --name $RGNAME --location $LOCNAME
az servicebus namespace create --name <namespace-name> --resource-group $RGNAME
az servicebus queue create --name orders --namespace-name <namespace-name> --resource-group $RGNAME
```

Once you have created the Service Bus namespace and queue, you will need to decide on the authentication method. You can create a shared access policy with the **Send** permission for the queue or use Microsoft Entra Workload Identity for a passwordless experience (this is the recommended approach).

If you choose to use Workload Identity, you will need to assign the `Azure Service Bus Data Receiver` role to the identity that is running the app, which in this case will be your account. You can do this using the Azure CLI.

```bash
PRINCIPALID=$(az ad signed-in-user show --query objectId -o tsv)
SERVICEBUSBID=$(az servicebus namespace show --name <namespace-name> --resource-group <resource-group-name> --query id -o tsv)

az role assignment create --role "Azure Service Bus Data Receiver" --assignee $PRINCIPALID --scope $SERVICEBUSBID
```

Next, get the hostname for the Azure Service Bus.

```bash
HOSTNAME=$(az servicebus namespace show --name <namespace-name> --resource-group <resource-group-name> --query serviceBusEndpoint -o tsv | sed 's/https:\/\///;s/:443\///')
```

Finally, set the environment variables.

```bash
export ORDER_QUEUE_HOSTNAME=$HOSTNAME
export ORDER_QUEUE_NAME=orders
export USE_WORKLOAD_IDENTITY_AUTH=true
```

If you choose to use a shared access policy, you can create one using the Azure CLI. Otherwise, you can skip this step and proceed to [provision a database](#database-options).

```bash
az servicebus namespace authorization-rule create --name listener --namespace-name <namespace-name> --resource-group $RGNAME --rights Listen
```

Next, get the connection information for the Azure Service Bus.

```bash
HOSTNAME=$(az servicebus namespace show --name <namespace-name> --resource-group $RGNAME --query serviceBusEndpoint -o tsv | sed 's/https:\/\///;s/:443\///')
PASSWORD=$(az servicebus namespace authorization-rule keys list --namespace-name <namespace-name> --resource-group $RGNAME --name listener --query primaryKey -o tsv)
```

Finally, set the environment variables.

```bash
export ORDER_QUEUE_URI=amqps://$HOSTNAME
export ORDER_QUEUE_USERNAME=listener
export ORDER_QUEUE_PASSWORD=$PASSWORD
export ORDER_QUEUE_NAME=orders
```

> NOTE: If you are using Azure Service Bus, you will want your `order-service` to write orders to it instead of RabbitMQ. If that is the case, then you'll need to update the [`docker-compose.yml`](./docker-compose.yml) and modify the environment variables for the `order-service` to include the proper connection info to connect to Azure Service Bus. Also you will need to add the `ORDER_QUEUE_TRANSPORT=tls` configuration to connect over TLS.

## Database setup

This app writes orders to MongoDB only.

If you are using a local MongoDB container, run the following commands:

```bash
export ORDER_DB_URI=mongodb://localhost:27017
export ORDER_DB_NAME=orderdb
export ORDER_DB_COLLECTION_NAME=orders
```

## Running the app locally

The app relies on RabbitMQ and MongoDB. Additionally, to simulate orders, you will need to run the [order-service](../order-service) with the [virtual-customer](../virtual-customer) app. A docker-compose file is provided to make this easy.

To run the necessary services, clone the repo, open a terminal, and navigate to the `makeline-service` directory. Then run the following command:

```bash
docker compose up -d
```

Now you can run the following commands to start the application:

```bash
go get .
go run .
```

When the app is running, you should see output similar to the following:

```text
[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] GET    /order/fetch              --> main.fetchOrders (4 handlers)
[GIN-debug] GET    /order/:id                --> main.getOrder (4 handlers)
[GIN-debug] PUT    /order                    --> main.updateOrder (4 handlers)
[GIN-debug] GET    /health                   --> main.main.func1 (4 handlers)
[GIN-debug] [WARNING] You trusted all proxies, this is NOT safe. We recommend you to set a value.
Please check https://pkg.go.dev/github.com/gin-gonic/gin#readme-don-t-trust-all-proxies for details.
[GIN-debug] Listening and serving HTTP on :3001
```

Using the [`test-makeline-service.http`](./test-makeline-service.http) file in the root of the repo, you can test the API. However, you will need to use VS Code and have the [REST Client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client) extension installed.

To view the orders in MongoDB, open a terminal and run the following command:

```bash
# connect to mongodb
mongosh

# show databases and confirm orderdb exists
show dbs

# use orderdb
use orderdb

# show collections and confirm orders exists
show collections

# get the orders
db.orders.find()

# get completed orders
db.orders.findOne({status: 1})
```

